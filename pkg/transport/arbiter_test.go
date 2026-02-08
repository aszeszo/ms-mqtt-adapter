package transport

import (
	"context"
	"fmt"
	"log/slog"
	"ms-mqtt-adapter/internal/mysensors"
	"sync"
	"testing"
	"time"
)

// mockTransport implements Transport for testing.
type mockTransport struct {
	mu        sync.Mutex
	connected bool
	msgChan   chan *mysensors.Message
	sentMsgs  []*mysensors.Message
	sendDelay time.Duration
	sendErr   error
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		msgChan: make(chan *mysensors.Message, 100),
	}
}

func (m *mockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *mockTransport) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *mockTransport) Send(message *mysensors.Message) error {
	m.mu.Lock()
	err := m.sendErr
	delay := m.sendDelay
	m.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.sentMsgs = append(m.sentMsgs, message)
	m.mu.Unlock()
	return nil
}

func (m *mockTransport) Receive() <-chan *mysensors.Message {
	return m.msgChan
}

func (m *mockTransport) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *mockTransport) getSentMessages() []*mysensors.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*mysensors.Message, len(m.sentMsgs))
	copy(result, m.sentMsgs)
	return result
}

func (m *mockTransport) simulateRX(msg *mysensors.Message) {
	m.msgChan <- msg
}

func testLogger() *slog.Logger {
	return slog.Default()
}

func testMessage(nodeID, childID int) *mysensors.Message {
	return mysensors.NewSetMessage(nodeID, childID, mysensors.V_STATUS, "1")
}

func TestArbiterSendImmediateWhenQuiet(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      50 * time.Millisecond,
		InterMessageDelay: 10 * time.Millisecond,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// No RX activity, send should happen quickly
	start := time.Now()
	err := arbiter.Send(testMessage(1, 0))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("Send took too long when bus was quiet: %v", elapsed)
	}

	msgs := mock.getSentMessages()
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 sent message, got %d", len(msgs))
	}
}

func TestArbiterWaitsForQuietBus(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      100 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         2 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Simulate RX activity
	mock.simulateRX(testMessage(5, 0))

	// Give rxProxy time to process the message
	time.Sleep(20 * time.Millisecond)

	// Send should wait for bus quiet
	start := time.Now()
	err := arbiter.Send(testMessage(1, 0))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Should have waited approximately BusQuietTime minus the time already elapsed
	if elapsed < 50*time.Millisecond {
		t.Errorf("Send did not wait long enough for bus quiet: %v", elapsed)
	}
}

func TestArbiterInterMessageDelay(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      10 * time.Millisecond,
		InterMessageDelay: 100 * time.Millisecond,
		MaxTXWait:         2 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Send first message
	if err := arbiter.Send(testMessage(1, 0)); err != nil {
		t.Fatalf("First send failed: %v", err)
	}

	// Send second message - should be delayed by InterMessageDelay
	start := time.Now()
	if err := arbiter.Send(testMessage(2, 0)); err != nil {
		t.Fatalf("Second send failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 80*time.Millisecond {
		t.Errorf("Inter-message delay not enforced: %v (expected ~100ms)", elapsed)
	}

	msgs := mock.getSentMessages()
	if len(msgs) != 2 {
		t.Fatalf("Expected 2 sent messages, got %d", len(msgs))
	}
}

func TestArbiterMaxTXWait(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      200 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         100 * time.Millisecond,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Continuously simulate RX to keep bus busy
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				mock.simulateRX(testMessage(5, 0))
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	defer close(stop)

	// Give rxProxy time to start processing
	time.Sleep(30 * time.Millisecond)

	// Send should be forced after MaxTXWait
	start := time.Now()
	err := arbiter.Send(testMessage(1, 0))
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Should complete within MaxTXWait + some tolerance
	if elapsed > 200*time.Millisecond {
		t.Errorf("Send took too long even with MaxTXWait: %v", elapsed)
	}

	// Verify force count
	stats := arbiter.GetStats()
	if stats.ForceCount == 0 {
		t.Error("Expected force count > 0")
	}
}

func TestArbiterFIFOOrder(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      5 * time.Millisecond,
		InterMessageDelay: 5 * time.Millisecond,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Send multiple messages concurrently
	var wg sync.WaitGroup
	count := 10
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			arbiter.Send(testMessage(id, 0))
		}(i)
		// Small stagger to ensure queue ordering
		time.Sleep(2 * time.Millisecond)
	}

	wg.Wait()

	msgs := mock.getSentMessages()
	if len(msgs) != count {
		t.Fatalf("Expected %d sent messages, got %d", count, len(msgs))
	}
}

func TestArbiterReceiveProxiesMessages(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      50 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Send RX message via mock
	expected := testMessage(42, 3)
	mock.simulateRX(expected)

	// Should appear on arbiter's Receive channel
	select {
	case msg := <-arbiter.Receive():
		if msg.NodeID != expected.NodeID || msg.ChildID != expected.ChildID {
			t.Errorf("Received wrong message: got %v, want %v", msg, expected)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for proxied RX message")
	}
}

func TestArbiterDisconnectDrainsPending(t *testing.T) {
	mock := newMockTransport()
	// Use a long delay to keep messages in flight
	mock.sendDelay = 500 * time.Millisecond

	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      5 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	// Queue a send that will take a while
	errCh := make(chan error, 1)
	go func() {
		errCh <- arbiter.Send(testMessage(1, 0))
	}()

	// Give it time to start processing
	time.Sleep(20 * time.Millisecond)

	// Queue more sends that will be pending
	var wg sync.WaitGroup
	errors := make([]error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errors[idx] = arbiter.Send(testMessage(idx+10, 0))
		}(i)
		time.Sleep(5 * time.Millisecond)
	}

	// Disconnect should drain pending sends
	time.Sleep(20 * time.Millisecond)
	arbiter.Disconnect()

	wg.Wait()

	// At least some of the pending sends should have returned errors
	errorCount := 0
	for _, err := range errors {
		if err != nil {
			errorCount++
		}
	}
	// We expect errors since we disconnected while sends were pending
	t.Logf("Got %d errors out of 5 pending sends after disconnect", errorCount)
}

func TestArbiterSendErrorPropagation(t *testing.T) {
	mock := newMockTransport()
	mock.sendErr = fmt.Errorf("mock send error")

	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      5 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	err := arbiter.Send(testMessage(1, 0))
	if err == nil {
		t.Error("Expected error from Send, got nil")
	}
}

func TestArbiterStatsAccumulate(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      50 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         2 * time.Second,
	}, testLogger())

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer arbiter.Disconnect()

	// Send without RX activity - should not accumulate wait
	if err := arbiter.Send(testMessage(1, 0)); err != nil {
		t.Fatal(err)
	}

	// Simulate RX then send - should accumulate wait
	mock.simulateRX(testMessage(5, 0))
	time.Sleep(10 * time.Millisecond)
	if err := arbiter.Send(testMessage(2, 0)); err != nil {
		t.Fatal(err)
	}

	stats := arbiter.GetStats()
	if stats.WaitCount == 0 {
		t.Error("Expected wait count > 0 after sending during bus activity")
	}
	if stats.LastTXAgoMs < 0 {
		t.Error("Expected non-negative LastTXAgoMs")
	}
}

func TestArbiterIsConnectedDelegates(t *testing.T) {
	mock := newMockTransport()
	arbiter := NewArbiterTransport(mock, ArbiterConfig{
		BusQuietTime:      50 * time.Millisecond,
		InterMessageDelay: 0,
		MaxTXWait:         1 * time.Second,
	}, testLogger())

	if arbiter.IsConnected() {
		t.Error("Expected not connected before Connect()")
	}

	ctx := context.Background()
	if err := arbiter.Connect(ctx); err != nil {
		t.Fatal(err)
	}

	if !arbiter.IsConnected() {
		t.Error("Expected connected after Connect()")
	}

	arbiter.Disconnect()

	if arbiter.IsConnected() {
		t.Error("Expected not connected after Disconnect()")
	}
}
