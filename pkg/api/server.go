package api

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"ms-mqtt-adapter/pkg/config"
	"ms-mqtt-adapter/pkg/transport"
	"net/http"
	"strings"
	"time"
)

// StatusProvider is implemented by the Application to decouple the API from concrete types.
type StatusProvider interface {
	GetConfig() *config.Config
	GetConfigPath() string
	GetTransportStatus() map[string]TransportStatus
	GetMQTTStatus() MQTTStatus
	GetGatewayStatus(name string) GatewayStatus
	GetAllGatewayStatus() map[string]GatewayStatus
	GetEntityStates() map[string]string
	GetMQTTClient() MQTTClientProvider
}

// MQTTClientProvider provides access to MQTT client operations
type MQTTClientProvider interface {
	GetTopicList(gateways map[string]config.MySensorsConfig) interface{}
	DeleteTopics(scope, deviceID, entityID string, gateways map[string]config.MySensorsConfig) error
	BrowseAllTopics(timeout time.Duration) (any, error)
	DeleteRetainedTopic(topic string) error
	DeleteRetainedTree(prefix string, timeout time.Duration) (int, error)
}

type TransportStatus struct {
	Connected  bool                    `json:"connected"`
	Transport  string                  `json:"transport"`
	HalfDuplex *transport.ArbiterStats `json:"half_duplex,omitempty"`
}

type MQTTStatus struct {
	Connected bool
}

type GatewayStatus struct {
	Connected        bool
	Transport        string
	SeenNodes        []int
	NodeAvailability map[int]bool
	LastSeenNodeID   int
	LastIssuedNodeID int
}

// Server is the HTTP/WebSocket server for the REST API and web dashboard.
type Server struct {
	httpServer   *http.Server
	provider     StatusProvider
	configWriter *ConfigWriter
	eventBus     *EventBus
	logger       *slog.Logger
}

func NewServer(port int, provider StatusProvider, bus *EventBus, logStreamer LogStreamer, staticFS fs.FS, logger *slog.Logger) *Server {
	cw := NewConfigWriter(provider.GetConfigPath())

	s := &Server{
		provider:     provider,
		configWriter: cw,
		eventBus:     bus,
		logger:       logger,
	}

	mux := http.NewServeMux()

	// --- REST API: Config ---
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)
	mux.HandleFunc("GET /api/config/log_level", s.handleGetLogLevel)
	mux.HandleFunc("PUT /api/config/log_level", s.handlePutLogLevel)
	mux.HandleFunc("GET /api/config/mqtt", s.handleGetMQTT)
	mux.HandleFunc("PUT /api/config/mqtt", s.handlePutMQTT)
	mux.HandleFunc("GET /api/config/adapter", s.handleGetAdapter)
	mux.HandleFunc("PUT /api/config/adapter", s.handlePutAdapter)
	mux.HandleFunc("GET /api/config/id_aliases", s.handleGetAliases)
	mux.HandleFunc("PUT /api/config/id_aliases", s.handlePutAliases)
	mux.HandleFunc("GET /api/config/mysensors", s.handleGetMySensors)
	mux.HandleFunc("PUT /api/config/mysensors", s.handlePutMySensors)
	mux.HandleFunc("GET /api/config/mysensors/{name}", s.handleGetGatewayConfig)
	mux.HandleFunc("PUT /api/config/mysensors/{name}", s.handlePutGatewayConfig)
	mux.HandleFunc("DELETE /api/config/mysensors/{name}", s.handleDeleteGatewayConfig)
	mux.HandleFunc("GET /api/config/devices", s.handleGetDevices)
	mux.HandleFunc("PUT /api/config/devices", s.handlePutDevices)
	mux.HandleFunc("POST /api/config/devices", s.handlePostDevice)
	mux.HandleFunc("GET /api/config/devices/{id}", s.handleGetDevice)
	mux.HandleFunc("PUT /api/config/devices/{id}", s.handlePutDevice)
	mux.HandleFunc("DELETE /api/config/devices/{id}", s.handleDeleteDevice)
	mux.HandleFunc("GET /api/config/devices/{id}/entities", s.handleGetEntities)
	mux.HandleFunc("POST /api/config/devices/{id}/entities", s.handlePostEntity)
	mux.HandleFunc("GET /api/config/devices/{id}/entities/{eid}", s.handleGetEntity)
	mux.HandleFunc("PUT /api/config/devices/{id}/entities/{eid}", s.handlePutEntity)
	mux.HandleFunc("DELETE /api/config/devices/{id}/entities/{eid}", s.handleDeleteEntity)
	mux.HandleFunc("POST /api/config/validate", s.handleValidateConfig)
	mux.HandleFunc("GET /api/config/raw", s.handleGetRawConfig)
	mux.HandleFunc("PUT /api/config/raw", s.handlePutRawConfig)

	// --- REST API: Status ---
	mux.HandleFunc("GET /api/status", s.handleGetStatus)
	mux.HandleFunc("GET /api/status/entities", s.handleGetEntityStates)
	mux.HandleFunc("GET /api/status/gateways/{name}", s.handleGetGatewayStatus)

	// --- REST API: MQTT Topics ---
	mux.HandleFunc("GET /api/mqtt/topics", s.handleGetMQTTTopics)
	mux.HandleFunc("DELETE /api/mqtt/topics", s.handleDeleteMQTTTopics)

	// --- WebSocket ---
	mux.HandleFunc("/ws/events", handleWebSocket(provider, bus, logger))
	mux.HandleFunc("/ws/logs", handleLogStream(logStreamer, logger))
	mux.HandleFunc("/ws/traffic", handleTrafficWebSocket(bus, logger))

	// --- Frontend static files ---
	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Serve index.html for SPA routes (non-file paths)
			path := r.URL.Path
			if path == "/" || (!strings.Contains(path, ".") && !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws/")) {
				// Inject ingress base path into the HTML
				s.serveIndex(w, r, staticFS)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	handler := ingressMiddleware(logger, loggingMiddleware(logger, mux))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", "addr", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(shutdownCtx)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request, staticFS fs.FS) {
	data, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}

	// Inject the ingress base path from the header
	basePath := r.Header.Get("X-Ingress-Path")

	// Log ingress path for debugging - log both when present and when missing
	s.logger.Debug("Serving index.html",
		"ingress_path", basePath,
		"request_path", r.URL.Path,
		"has_ingress_header", basePath != "",
		"all_headers", r.Header)

	html := strings.ReplaceAll(string(data), "{{BASE_PATH}}", basePath)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// currentConfig returns a deep-ish copy of the live config by re-reading from disk.
// This ensures we always work with the file as source of truth.
func (s *Server) currentConfig() (*config.Config, error) {
	return config.LoadConfig(s.configWriter.Path())
}

// saveConfig saves the config and returns an error suitable for HTTP responses.
func (s *Server) saveConfig(cfg *config.Config) error {
	return s.configWriter.Save(cfg)
}
