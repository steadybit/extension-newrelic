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

	"github.com/steadybit/extension-newrelic/types"
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

// New Relic reports authorization problems with HTTP 200 and a populated errors array.
// The rendered error must name the rejected field so the log identifies the query.
func TestGraphQlErrorsIncludePathAndErrorClass(t *testing.T) {
	var result types.GraphQlResponse
	body := `{"data":{"actor":{"account":{"aiIssues":null}}},"errors":[{"extensions":{"errorClass":"UNAUTHORIZED"},"path":["actor","account","aiIssues","incidents"],"message":"user's role doesn't permit this action"}]}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := graphQlErrors(&result)
	want := "actor.account.aiIssues.incidents: user's role doesn't permit this action (UNAUTHORIZED)"
	if got != want {
		t.Errorf("graphQlErrors = %q, want %q", got, want)
	}
}

// Path elements may be list indices rather than field names - those must not break parsing.
func TestGraphQlErrorsWithListIndexInPath(t *testing.T) {
	var result types.GraphQlResponse
	body := `{"errors":[{"path":["actor","entities",0,"tags"],"message":"boom"}]}`
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := graphQlErrors(&result)
	want := "actor.entities.0.tags: boom"
	if got != want {
		t.Errorf("graphQlErrors = %q, want %q", got, want)
	}
}

func TestGraphQlErrorsEmptyWhenNoErrors(t *testing.T) {
	var result types.GraphQlResponse
	if err := json.Unmarshal([]byte(`{"data":{"actor":{"accounts":[{"id":1}]}}}`), &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := graphQlErrors(&result); got != "" {
		t.Errorf("graphQlErrors = %q, want empty", got)
	}
}

// A permission error must not degrade to "no incidents" - that would let an incident
// check pass while it cannot see incidents at all.
func TestGetIncidentsFailsOnUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"account":{"aiIssues":null}}},"errors":[{"extensions":{"errorClass":"UNAUTHORIZED"},"path":["actor","account","aiIssues"],"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	incidents, err := s.GetIncidents(context.Background(), []string{"CRITICAL"}, 123)
	if err == nil {
		t.Fatalf("expected an error, got incidents %+v", incidents)
	}
	if !strings.Contains(err.Error(), "user's role doesn't permit this action") {
		t.Errorf("error does not carry the API message: %v", err)
	}
}

// An account without incidents legitimately answers with an empty list - still no error.
func TestGetIncidentsEmptyWithoutErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"account":{"aiIssues":{"incidents":{"incidents":[]}}}}}}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	incidents, err := s.GetIncidents(context.Background(), []string{"CRITICAL"}, 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(incidents) != 0 {
		t.Errorf("expected no incidents, got %+v", incidents)
	}
}

// A top-level error can come with `data: null` - discovery must error out, not panic.
func TestGetAccountIdsWithNullData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	if _, err := s.GetAccountIds(context.Background()); err == nil {
		t.Error("expected an error for a response without data")
	}
}

// The organization's managed accounts are authoritative: the storage account
// (4942247 here) must not be operated on, only the managed account (2847806).
func TestGetAccountIdsUsesManagedAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"accounts":[{"id":2847806},{"id":4942247}],"organization":{"accountManagement":{"managedAccounts":[{"id":2847806,"isCanceled":false}]},"storageAccountId":4942247}}}}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	accounts, err := s.GetAccountIds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 || accounts[0] != 2847806 {
		t.Errorf("expected only the managed account 2847806, got %v", accounts)
	}
}

func TestGetAccountIdsSkipsCanceledAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"organization":{"accountManagement":{"managedAccounts":[{"id":1,"isCanceled":false},{"id":2,"isCanceled":true}]},"storageAccountId":9}}}}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	accounts, err := s.GetAccountIds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 || accounts[0] != 1 {
		t.Errorf("expected only the active account 1, got %v", accounts)
	}
}

// An API key whose user may not read the organization must keep working: fall back to
// actor.accounts, still without the storage account if that field alone was readable.
func TestGetAccountIdsFallsBackToActorAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"accounts":[{"id":2847806},{"id":4942247}],"organization":{"accountManagement":null,"storageAccountId":4942247}}},"errors":[{"path":["actor","organization","accountManagement"],"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	accounts, err := s.GetAccountIds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 || accounts[0] != 2847806 {
		t.Errorf("expected the storage account to be filtered out, got %v", accounts)
	}
}

// Without any organization data there is nothing to filter by - keep the old behaviour
// rather than reporting no accounts at all.
func TestGetAccountIdsFallsBackWithoutOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"accounts":[{"id":2847806}],"organization":null}},"errors":[{"path":["actor","organization"],"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	accounts, err := s.GetAccountIds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 || accounts[0] != 2847806 {
		t.Errorf("expected the accounts from actor.accounts, got %v", accounts)
	}
}

// A permission error on the workload query yields `workload: null` - must not panic.
func TestGetWorkloadsWithNullWorkload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"account":{"workload":null}}},"errors":[{"path":["actor","account","workload"],"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	if _, err := s.GetWorkloads(context.Background(), 123); err == nil {
		t.Error("expected an error for a response without workloads")
	}
}

// The same for the status query, which falls back to UNKNOWN rather than failing.
func TestGetWorkloadStatusWithNullWorkload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"actor":{"account":{"workload":null}}},"errors":[{"path":["actor","account","workload"],"message":"user's role doesn't permit this action"}]}`))
	}))
	defer server.Close()

	s := &Specification{ApiBaseUrl: server.URL, ApiKey: "test-key"}

	status, err := s.GetWorkloadStatus(context.Background(), "guid-1", 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status == nil || *status != "UNKNOWN" {
		t.Errorf("expected status UNKNOWN, got %v", status)
	}
}
