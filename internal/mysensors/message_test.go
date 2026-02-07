package mysensors

import (
	"testing"
)

func TestParseMessage(t *testing.T) {
	// Test I_DISCOVER_RESPONSE message
	data := "31;255;3;0;21;0"
	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Failed to parse message: %v", err)
	}
	
	if msg.NodeID != 31 {
		t.Errorf("Expected NodeID 31, got %d", msg.NodeID)
	}
	
	if msg.ChildID != 255 {
		t.Errorf("Expected ChildID 255, got %d", msg.ChildID)
	}
	
	if msg.MessageType != INTERNAL {
		t.Errorf("Expected MessageType INTERNAL, got %d", msg.MessageType)
	}
	
	if msg.SubType != 21 {
		t.Errorf("Expected SubType 21 (I_DISCOVER_RESPONSE), got %d", msg.SubType)
	}
	
	if msg.Payload != "0" {
		t.Errorf("Expected Payload '0', got '%s'", msg.Payload)
	}
}

func TestInternalTypeConstants(t *testing.T) {
	// Verify that I_DISCOVER_RESPONSE is correctly defined
	if I_DISCOVER_RESPONSE != 21 {
		t.Errorf("Expected I_DISCOVER_RESPONSE to be 21, got %d", I_DISCOVER_RESPONSE)
	}
	
	// Verify that I_TIME is correctly defined
	if I_TIME != 1 {
		t.Errorf("Expected I_TIME to be 1, got %d", I_TIME)
	}
}