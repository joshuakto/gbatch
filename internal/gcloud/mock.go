package gcloud

import (
	"context"
	"encoding/json"
	"fmt"
)

// MockExecutor replays canned responses for testing.
type MockExecutor struct {
	// Responses maps gcloud subcommands to their JSON responses.
	// Key is the first arg (e.g., "batch", "logging", "compute").
	Responses map[string]json.RawMessage
	// Errors maps gcloud subcommands to errors.
	Errors map[string]error
	// Calls records all calls made for assertion.
	Calls [][]string
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Responses: make(map[string]json.RawMessage),
		Errors:    make(map[string]error),
	}
}

func (m *MockExecutor) Run(_ context.Context, args ...string) (json.RawMessage, error) {
	m.Calls = append(m.Calls, args)
	key := ""
	if len(args) > 0 {
		key = args[0]
	}
	if err, ok := m.Errors[key]; ok {
		return nil, err
	}
	if resp, ok := m.Responses[key]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("mock: no response configured for gcloud %s", key)
}

func (m *MockExecutor) RunRaw(_ context.Context, args ...string) ([]byte, error) {
	resp, err := m.Run(nil, args...)
	if err != nil {
		return nil, err
	}
	return []byte(resp), nil
}
