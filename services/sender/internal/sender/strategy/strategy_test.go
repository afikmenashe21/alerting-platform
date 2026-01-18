package strategy

import (
	"context"
	"fmt"
	"testing"

	"sender/internal/database"
)

// mockSender is a mock implementation of NotificationSender for testing
type mockSender struct {
	senderType string
	sendErr    error
}

func (m *mockSender) Send(ctx context.Context, endpointValue string, notification *database.Notification) error {
	return m.sendErr
}

func (m *mockSender) Type() string {
	return m.senderType
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.senders == nil {
		t.Error("NewRegistry() senders map should not be nil")
	}

	if len(registry.senders) != 0 {
		t.Errorf("NewRegistry() senders map should be empty, got %d", len(registry.senders))
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	sender1 := &mockSender{senderType: "email"}
	sender2 := &mockSender{senderType: "slack"}

	registry.Register(sender1)
	registry.Register(sender2)

	if len(registry.senders) != 2 {
		t.Errorf("Register() should have 2 senders, got %d", len(registry.senders))
	}

	// Test overwriting
	sender3 := &mockSender{senderType: "email"}
	registry.Register(sender3)

	if len(registry.senders) != 2 {
		t.Errorf("Register() should still have 2 senders after overwrite, got %d", len(registry.senders))
	}

	// Verify the overwritten sender
	retrieved, ok := registry.Get("email")
	if !ok {
		t.Fatal("Register() should have email sender after overwrite")
	}
	if retrieved != sender3 {
		t.Error("Register() should overwrite existing sender")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	sender := &mockSender{senderType: "email"}
	registry.Register(sender)

	tests := []struct {
		name      string
		senderType string
		wantOk    bool
	}{
		{
			name:      "existing sender",
			senderType: "email",
			wantOk:    true,
		},
		{
			name:      "non-existent sender",
			senderType: "webhook",
			wantOk:    false,
		},
		{
			name:      "empty type",
			senderType: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := registry.Get(tt.senderType)
			if ok != tt.wantOk {
				t.Errorf("Registry.Get() ok = %v, want %v", ok, tt.wantOk)
			}
			if tt.wantOk && got == nil {
				t.Error("Registry.Get() should return non-nil sender when ok is true")
			}
			if tt.wantOk && got.Type() != tt.senderType {
				t.Errorf("Registry.Get() sender type = %v, want %v", got.Type(), tt.senderType)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	types := registry.List()
	if len(types) != 0 {
		t.Errorf("Registry.List() should return empty slice for empty registry, got %v", types)
	}

	// Add senders
	sender1 := &mockSender{senderType: "email"}
	sender2 := &mockSender{senderType: "slack"}
	sender3 := &mockSender{senderType: "webhook"}

	registry.Register(sender1)
	registry.Register(sender2)
	registry.Register(sender3)

	types = registry.List()
	if len(types) != 3 {
		t.Errorf("Registry.List() should return 3 types, got %d", len(types))
	}

	// Check that all types are present
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	expectedTypes := []string{"email", "slack", "webhook"}
	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("Registry.List() should contain %s", expected)
		}
	}
}

func TestMockSender_Interface(t *testing.T) {
	// Test that mockSender implements NotificationSender interface
	var _ NotificationSender = &mockSender{}

	sender := &mockSender{
		senderType: "test",
		sendErr:    nil,
	}

	if sender.Type() != "test" {
		t.Errorf("mockSender.Type() = %v, want test", sender.Type())
	}

	ctx := context.Background()
	notification := &database.Notification{
		NotificationID: "test",
	}

	err := sender.Send(ctx, "endpoint", notification)
	if err != nil {
		t.Errorf("mockSender.Send() error = %v, want nil", err)
	}

	// Test with error
	senderWithErr := &mockSender{
		senderType: "test",
		sendErr:    fmt.Errorf("test error"),
	}

	err = senderWithErr.Send(ctx, "endpoint", notification)
	if err == nil {
		t.Error("mockSender.Send() should return error when sendErr is set")
	}
}
