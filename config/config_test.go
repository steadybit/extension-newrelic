/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package config

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGraphQlStringEscaping(t *testing.T) {
	got := graphQlString(`a"b\c` + "\n")
	want := `"a\"b\\c\n"`
	if got != want {
		t.Errorf("graphQlString = %s, want %s", got, want)
	}
}

func TestCreateMutingRuleEscapesInjection(t *testing.T) {
	var captured []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"alertsMutingRuleCreate":{"id":"42"}}}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	// A name/description that tries to break out of the GraphQL string literal.
	maliciousName := `exp", enabled: false, x: "`
	maliciousDescription := "line1\nline2\\\"end"

	id, err := s.CreateMutingRule(context.Background(), 123, maliciousName, maliciousDescription, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == nil || *id != "42" {
		t.Fatalf("expected muting rule id 42, got %v", id)
	}

	// The request envelope must remain valid JSON despite the quotes/newlines/backslashes.
	var envelope map[string]string
	if err := json.Unmarshal(captured, &envelope); err != nil {
		t.Fatalf("request body is not valid JSON (injection broke the envelope): %v\nbody: %s", err, captured)
	}

	// The malicious inputs must be carried as escaped GraphQL string literals, not as raw query text.
	query := envelope["query"]
	if !strings.Contains(query, "name: "+graphQlString(maliciousName)) {
		t.Errorf("name was not embedded as an escaped literal: %s", query)
	}
	if !strings.Contains(query, "description: "+graphQlString(maliciousDescription)) {
		t.Errorf("description was not embedded as an escaped literal: %s", query)
	}
}
