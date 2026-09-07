// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extevents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
	"github.com/steadybit/extension-newrelic/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseEvent() event_kit_api.EventRequestBody {
	return event_kit_api.EventRequestBody{
		Environment: new(event_kit_api.Environment{Id: "test", Name: "gateway"}),
		EventTime:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Id:          uuid.New(),
		Principal: event_kit_api.BatchPrincipal{
			Username:      "batch-user",
			PrincipalType: string(event_kit_api.BatchJob),
		},
		Tenant: event_kit_api.Tenant{Key: "key", Name: "name"},
	}
}

func experimentExecution() *event_kit_api.ExperimentExecution {
	return new(event_kit_api.ExperimentExecution{
		ExecutionId:   42,
		ExperimentKey: "ExperimentKey",
		Name:          "Name",
		State:         event_kit_api.ExperimentExecutionState("COMPLETED"),
		PreparedTime:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		StartedTime:   time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC),
	})
}

func TestOnExperimentStarted(t *testing.T) {
	event := baseEvent()
	event.EventName = "experiment.started"
	event.ExperimentExecution = experimentExecution()

	got, err := onExperimentStarted(&event)

	require.NoError(t, err)
	assert.Equal(t, &types.EventIngest{
		EventType:         types.EventTypeExperimentStarted,
		EnvironmentName:   "gateway",
		PrincipalType:     "batch_job",
		PrincipalUsername: "batch-user",
		ExecutionId:       "42",
		ExperimentKey:     "ExperimentKey",
		ExperimentName:    "Name",
	}, got)
}

func TestOnExperimentCompleted(t *testing.T) {
	stepId := uuid.New()
	otherStepId := uuid.New()

	// A step of the completing execution and one of a different execution. Only the
	// former must be dropped from the step cache.
	stepExecutions.Store(stepId, event_kit_api.ExperimentStepExecution{ExecutionId: 42, Id: stepId})
	stepExecutions.Store(otherStepId, event_kit_api.ExperimentStepExecution{ExecutionId: 43, Id: otherStepId})
	t.Cleanup(func() { stepExecutions.Delete(otherStepId) })

	event := baseEvent()
	event.EventName = "experiment.completed"
	event.ExperimentExecution = experimentExecution()

	got, err := onExperimentCompleted(&event)

	require.NoError(t, err)
	assert.Equal(t, types.EventTypeExperimentEnded, got.EventType)
	assert.Equal(t, "COMPLETED", got.State)
	assert.Equal(t, "ExperimentKey", got.ExperimentKey)

	_, stillCached := stepExecutions.Load(stepId)
	assert.False(t, stillCached, "the completed execution's step must be evicted")
	_, otherStillCached := stepExecutions.Load(otherStepId)
	assert.True(t, otherStillCached, "another execution's step must be kept")
}

func TestOnExperimentStepStartedRequiresAStepExecution(t *testing.T) {
	event := baseEvent()
	_, err := onExperimentStepStarted(&event)
	require.EqualError(t, err, "missing ExperimentStepExecution in event")
}

func TestOnExperimentTargetHandlers(t *testing.T) {
	targetEvent := func(stepExecutionId uuid.UUID) event_kit_api.EventRequestBody {
		event := baseEvent()
		event.ExperimentStepTargetExecution = new(event_kit_api.ExperimentStepTargetExecution{
			ExecutionId:     42,
			ExperimentKey:   "ExperimentKey",
			StepExecutionId: stepExecutionId,
			State:           "failed",
			TargetType:      "type",
			TargetName:      "fallback-name",
			TargetAttributes: map[string][]string{
				"steadybit.label": {"nice-label"},
			},
		})
		return event
	}

	t.Run("require a target execution", func(t *testing.T) {
		event := baseEvent()
		_, err := onExperimentTargetStarted(&event)
		require.EqualError(t, err, "missing ExperimentStepTargetExecution in event")
		_, err = onExperimentTargetCompleted(&event)
		require.EqualError(t, err, "missing ExperimentStepTargetExecution in event")
	})

	t.Run("skip targets whose step is unknown", func(t *testing.T) {
		event := targetEvent(uuid.New())
		got, err := onExperimentTargetStarted(&event)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = onExperimentTargetCompleted(&event)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("skip steps that are not attacks", func(t *testing.T) {
		stepId := uuid.New()
		stepExecutions.Store(stepId, event_kit_api.ExperimentStepExecution{
			Id:         stepId,
			ActionKind: extutil.Ptr(event_kit_api.Check),
		})
		t.Cleanup(func() { stepExecutions.Delete(stepId) })

		event := targetEvent(stepId)
		got, err := onExperimentTargetStarted(&event)
		require.NoError(t, err)
		assert.Nil(t, got)
		got, err = onExperimentTargetCompleted(&event)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("skip steps without an action kind", func(t *testing.T) {
		stepId := uuid.New()
		stepExecutions.Store(stepId, event_kit_api.ExperimentStepExecution{Id: stepId})
		t.Cleanup(func() { stepExecutions.Delete(stepId) })

		event := targetEvent(stepId)
		got, err := onExperimentTargetStarted(&event)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("emit an attack ended event with the target state", func(t *testing.T) {
		stepId := uuid.New()
		stepExecutions.Store(stepId, event_kit_api.ExperimentStepExecution{
			Id:          stepId,
			Type:        event_kit_api.Action,
			ActionId:    new("some_action_id"),
			ActionName:  new("the step"),
			CustomLabel: new("custom label"),
			ActionKind:  extutil.Ptr(event_kit_api.Attack),
		})
		t.Cleanup(func() { stepExecutions.Delete(stepId) })

		event := targetEvent(stepId)
		got, err := onExperimentTargetCompleted(&event)

		require.NoError(t, err)
		assert.Equal(t, &types.EventIngest{
			EventType:         types.EventTypeAttackEnded,
			EnvironmentName:   "gateway",
			PrincipalType:     "batch_job",
			PrincipalUsername: "batch-user",
			ExecutionId:       "42",
			ExperimentKey:     "ExperimentKey",
			ActionId:          "some_action_id",
			ActionName:        "the step",
			ActionCustomLabel: "custom label",
			Target:            "nice-label",
			TargetType:        "type",
			TargetState:       "failed",
		}, got)
	})
}

func TestGetTargetNameFallsBackToTheTargetName(t *testing.T) {
	assert.Equal(t, "fallback", getTargetName(event_kit_api.ExperimentStepTargetExecution{
		TargetName: "fallback",
	}))
	assert.Equal(t, "labelled", getTargetName(event_kit_api.ExperimentStepTargetExecution{
		TargetName:       "fallback",
		TargetAttributes: map[string][]string{"steadybit.label": {"labelled"}},
	}))
}

func TestParseBodyToEventRequestBodyFailsOnGarbage(t *testing.T) {
	_, err := parseBodyToEventRequestBody([]byte("{not json"))
	require.Error(t, err)
}

// setAccountCache replaces the package-level account cache for the duration of a test.
func setAccountCache(t *testing.T, accountIds []int64) {
	t.Helper()
	original := accountCache
	t.Cleanup(func() { accountCache = original })
	accountCache = ttlcache.New[string, []int64](ttlcache.WithTTL[string, []int64](time.Minute))
	if accountIds != nil {
		accountCache.Set(accountCacheKey, accountIds, ttlcache.DefaultTTL)
	}
}

func TestHandle(t *testing.T) {
	okEvent := func() []byte {
		event := baseEvent()
		event.EventName = "experiment.started"
		event.ExperimentExecution = experimentExecution()
		body, err := json.Marshal(event)
		require.NoError(t, err)
		return body
	}

	t.Run("posts the event to every account", func(t *testing.T) {
		var posted atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			posted.Add(1)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		originalConfig := config.Config
		t.Cleanup(func() { config.Config = originalConfig })
		config.Config.InsightsCollectorApiBaseUrl = server.URL
		setAccountCache(t, []int64{1234, 5678})

		recorder := httptest.NewRecorder()
		handle(onExperimentStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-started", nil), okEvent())

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `"{}"`, recorder.Body.String())
		assert.Equal(t, int32(2), posted.Load())
	})

	t.Run("keeps going when New Relic rejects the event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		originalConfig := config.Config
		t.Cleanup(func() { config.Config = originalConfig })
		config.Config.InsightsCollectorApiBaseUrl = server.URL
		setAccountCache(t, []int64{1234})

		recorder := httptest.NewRecorder()
		handle(onExperimentStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-started", nil), okEvent())

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("skips delivery when no accounts are known", func(t *testing.T) {
		originalConfig := config.Config
		t.Cleanup(func() { config.Config = originalConfig })
		// Any request would fail; the handler must not make one.
		config.Config.InsightsCollectorApiBaseUrl = "http://127.0.0.1:1"
		setAccountCache(t, nil)

		recorder := httptest.NewRecorder()
		handle(onExperimentStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-started", nil), okEvent())

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handle(onExperimentStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-started", nil), []byte("{not json"))

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "Failed to decode event request body")
	})

	t.Run("reports a handler error", func(t *testing.T) {
		body, err := json.Marshal(baseEvent())
		require.NoError(t, err)

		recorder := httptest.NewRecorder()
		handle(onExperimentStepStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-step-started", nil), body)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "missing ExperimentStepExecution in event")
	})

	t.Run("does nothing when the handler produces no event", func(t *testing.T) {
		event := baseEvent()
		stepId := uuid.New()
		event.ExperimentStepExecution = new(event_kit_api.ExperimentStepExecution{Id: stepId, ExecutionId: 42})
		t.Cleanup(func() { stepExecutions.Delete(stepId) })
		body, err := json.Marshal(event)
		require.NoError(t, err)

		setAccountCache(t, []int64{1234})
		recorder := httptest.NewRecorder()
		handle(onExperimentStepStarted)(recorder, httptest.NewRequest(http.MethodPost, "/events/experiment-step-started", nil), body)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `"{}"`, recorder.Body.String())
	})
}
