package api

import (
	"io"
	"log/slog"
	"ms-mqtt-adapter/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// mockStatusProvider implements StatusProvider for testing
type mockStatusProvider struct{}

func (m *mockStatusProvider) GetConfig() *config.Config                      { return nil }
func (m *mockStatusProvider) GetConfigPath() string                          { return "" }
func (m *mockStatusProvider) GetTransportStatus() map[string]TransportStatus { return nil }
func (m *mockStatusProvider) GetMQTTStatus() MQTTStatus                      { return MQTTStatus{} }
func (m *mockStatusProvider) GetGatewayStatus(name string) GatewayStatus     { return GatewayStatus{} }
func (m *mockStatusProvider) GetAllGatewayStatus() map[string]GatewayStatus  { return nil }
func (m *mockStatusProvider) GetEntityStates() map[string]string             { return nil }
func (m *mockStatusProvider) GetMQTTClient() MQTTClientProvider              { return nil }

func TestIngressIndexServing(t *testing.T) {
	// Create mock file system
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte(`<html><head><base href="{{BASE_PATH}}/"><script>window.__BASE_PATH__ = "{{BASE_PATH}}";</script></head><body></body></html>`)},
		"bundle.js":  {Data: []byte(`console.log("bundle loaded");`)},
	}

	// Create server instance (simplified for testing)
	// We only need the handler part that's constructed in NewServer, but NewServer has many dependencies.
	// Instead of full NewServer, we can test the ingressMiddleware and ServeHTTP logic directly if we can access them.
	// However, ServeHTTP is on the http.Server which is private in the Server struct.
	// But Server has ServeIndex method which is private but we can test via the public handler if we reconstruct it.

	// Let's create a server using NewServer with nil dependencies where possible.
	// We need a logger.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create a minimal server
	server := NewServer(8080, &mockStatusProvider{}, nil, nil, mockFS, logger)

	// The server creates an http.Server with a handler that wraps the mux with middleware.
	// We can access that handler via server.httpServer.Handler
	handler := server.httpServer.Handler

	tt := []struct {
		name         string
		path         string
		ingressPath  string
		expectCode   int
		expectBase   string
		expectWindow string
	}{
		{
			name:         "Root request without ingress",
			path:         "/",
			ingressPath:  "",
			expectCode:   http.StatusOK,
			expectBase:   `<base href="/">`,
			expectWindow: `window.__BASE_PATH__ = "";`,
		},
		{
			name:         "Root request with ingress",
			path:         "/",
			ingressPath:  "/api/ingress",
			expectCode:   http.StatusOK,
			expectBase:   `<base href="/api/ingress/">`,
			expectWindow: `window.__BASE_PATH__ = "/api/ingress";`,
		},
		{
			name:         "Subpath request with ingress",
			path:         "/dashboard",
			ingressPath:  "/api/ingress",
			expectCode:   http.StatusOK,
			expectBase:   `<base href="/api/ingress/">`,
			expectWindow: `window.__BASE_PATH__ = "/api/ingress";`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			if tc.ingressPath != "" {
				req.Header.Set("X-Ingress-Path", tc.ingressPath)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tc.expectCode)
			}

			body := rr.Body.String()

			// If we expect index.html content
			if tc.expectBase != "" {
				if !strings.Contains(body, tc.expectBase) {
					t.Errorf("handler returned unexpected body: got %v want it to contain %v", body, tc.expectBase)
				}
			}

			if tc.expectWindow != "" {
				if !strings.Contains(body, tc.expectWindow) {
					t.Errorf("handler returned unexpected body: got %v want it to contain %v", body, tc.expectWindow)
				}
			}
		})
	}
}

func TestIngressAssetServing(t *testing.T) {
	// Create mock file system
	mockFS := fstest.MapFS{
		"index.html": {Data: []byte(`index`)},
		"bundle.js":  {Data: []byte(`console.log("bundle loaded");`)},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(8080, &mockStatusProvider{}, nil, nil, mockFS, logger)
	handler := server.httpServer.Handler

	tt := []struct {
		name        string
		path        string
		ingressPath string
		expectCode  int
		expectBody  string
	}{
		{
			name:        "Asset request with ingress",
			path:        "/api/ingress/bundle.js",
			ingressPath: "/api/ingress",
			expectCode:  http.StatusOK,
			expectBody:  `console.log("bundle loaded");`,
		},
		{
			name:        "Asset request without ingress",
			path:        "/bundle.js",
			ingressPath: "",
			expectCode:  http.StatusOK,
			expectBody:  `console.log("bundle loaded");`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			if tc.ingressPath != "" {
				req.Header.Set("X-Ingress-Path", tc.ingressPath)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expectCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tc.expectCode)
			}

			if tc.expectBody != "" {
				if rr.Body.String() != tc.expectBody {
					t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), tc.expectBody)
				}
			}
		})
	}
}
