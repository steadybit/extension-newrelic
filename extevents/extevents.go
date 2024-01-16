// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extevents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jellydator/ttlcache/v3"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/event-kit/go/event_kit_api"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-newrelic/config"
	"github.com/steadybit/extension-newrelic/types"
	"net/http"
	"sync"
	"time"
)

func RegisterEventListenerHandlers() {
	loader := ttlcache.LoaderFunc[string, []int64](
		func(c *ttlcache.Cache[string, []int64], key string) *ttlcache.Item[string, []int64] {
			log.Debug().Msg("loading accounts")
			accounts, err := config.Config.GetAccountIds(context.Background())
			if err != nil {
				log.Err(err).Msgf("Failed to load accounts.")
			} else {
				log.Debug().Msgf("Successfully loaded %d account(s)", len(accounts))
				item := c.Set(accountCacheKey, accounts, ttlcache.DefaultTTL)
				return item
			}
			return nil
		},
	)
	accountCache = ttlcache.New[string, []int64](
		ttlcache.WithLoader[string, []int64](loader),
		ttlcache.WithTTL[string, []int64](30*time.Minute),
	)
	go accountCache.Start()

	exthttp.RegisterHttpHandler("/events/experiment-started", handle(onExperimentStarted))
	exthttp.RegisterHttpHandler("/events/experiment-completed", handle(onExperimentCompleted))
	exthttp.RegisterHttpHandler("/events/experiment-step-started", handle(onExperimentStepStarted))
	exthttp.RegisterHttpHandler("/events/experiment-target-started", handle(onExperimentTargetStarted))
	exthttp.RegisterHttpHandler("/events/experiment-target-completed", handle(onExperimentTargetCompleted))
}

type PostEventApi interface {
	PostEvent(ctx context.Context, event types.EventIngest, accountId int64) error
}

var (
	stepExecutions = sync.Map{}

	accountCache *ttlcache.Cache[string, []int64]
)

const accountCacheKey = "accountCache"

type eventHandler func(event *event_kit_api.EventRequestBody) (*types.EventIngest, error)

func handle(handler eventHandler) func(w http.ResponseWriter, r *http.Request, body []byte) {
	return func(w http.ResponseWriter, r *http.Request, body []byte) {

		event, err := parseBodyToEventRequestBody(body)
		if err != nil {
			exthttp.WriteError(w, extension_kit.ToError("Failed to decode event request body", err))
			return
		}

		if request, err := handler(&event); err == nil {
			if request != nil {
				for _, accountId := range accountCache.Get(accountCacheKey).Value() {
					eventErr := config.Config.PostEvent(r.Context(), request, accountId)
					if eventErr != nil {
						log.Err(eventErr).Int64("accountId", accountId).Msgf("Failed to send event to New Relic.")
					}
				}
			}
		} else {
			exthttp.WriteError(w, extension_kit.ToError(err.Error(), err))
			return
		}

		exthttp.WriteBody(w, "{}")
	}
}

func onExperimentStarted(event *event_kit_api.EventRequestBody) (*types.EventIngest, error) {
	newRelicEvent := types.EventIngest{
		EventType: types.EventTypeExperimentStarted,
	}
	addBaseProperties(&newRelicEvent, event)
	addExperimentExecutionProperties(&newRelicEvent, event.ExperimentExecution)
	return &newRelicEvent, nil
}

func onExperimentCompleted(event *event_kit_api.EventRequestBody) (*types.EventIngest, error) {
	stepExecutions.Range(func(key, value interface{}) bool {
		stepExecution := value.(event_kit_api.ExperimentStepExecution)
		if stepExecution.ExecutionId == event.ExperimentExecution.ExecutionId {
			log.Debug().Msgf("Delete step execution data for id %.0f", stepExecution.ExecutionId)
			stepExecutions.Delete(key)
		}
		return true
	})

	newRelicEvent := types.EventIngest{
		EventType: types.EventTypeExperimentEnded,
	}
	addBaseProperties(&newRelicEvent, event)
	addExperimentExecutionProperties(&newRelicEvent, event.ExperimentExecution)
	newRelicEvent.State = string(event.ExperimentExecution.State)
	return &newRelicEvent, nil
}

func onExperimentStepStarted(event *event_kit_api.EventRequestBody) (*types.EventIngest, error) {
	if event.ExperimentStepExecution == nil {
		return nil, errors.New("missing ExperimentStepExecution in event")
	}
	stepExecutions.Store(event.ExperimentStepExecution.Id, *event.ExperimentStepExecution)
	return nil, nil
}

func onExperimentTargetStarted(event *event_kit_api.EventRequestBody) (*types.EventIngest, error) {
	if event.ExperimentStepTargetExecution == nil {
		return nil, errors.New("missing ExperimentStepTargetExecution in event")
	}

	var v, ok = stepExecutions.Load(event.ExperimentStepTargetExecution.StepExecutionId)
	if !ok {
		log.Warn().Msgf("Could not find step infos for step execution id %s", event.ExperimentStepTargetExecution.StepExecutionId)
		return nil, nil
	}
	stepExecution := v.(event_kit_api.ExperimentStepExecution)
	if stepExecution.ActionKind == nil || *stepExecution.ActionKind != event_kit_api.Attack {
		return nil, nil
	}

	newRelicEvent := types.EventIngest{
		EventType: types.EventTypeAttackStarted,
	}
	addBaseProperties(&newRelicEvent, event)
	addStepExecutionProperties(&newRelicEvent, &stepExecution)
	addTargetExecutionProperties(&newRelicEvent, event.ExperimentStepTargetExecution)

	return &newRelicEvent, nil
}

func onExperimentTargetCompleted(event *event_kit_api.EventRequestBody) (*types.EventIngest, error) {
	if event.ExperimentStepTargetExecution == nil {
		return nil, errors.New("missing ExperimentStepTargetExecution in event")
	}

	var v, ok = stepExecutions.Load(event.ExperimentStepTargetExecution.StepExecutionId)
	if !ok {
		log.Warn().Msgf("Could not find step infos for step execution id %s", event.ExperimentStepTargetExecution.StepExecutionId)
		return nil, nil
	}
	stepExecution := v.(event_kit_api.ExperimentStepExecution)
	if stepExecution.ActionKind == nil || *stepExecution.ActionKind != event_kit_api.Attack {
		return nil, nil
	}

	newRelicEvent := types.EventIngest{
		EventType: types.EventTypeAttackEnded,
	}
	addBaseProperties(&newRelicEvent, event)
	addStepExecutionProperties(&newRelicEvent, &stepExecution)
	addTargetExecutionProperties(&newRelicEvent, event.ExperimentStepTargetExecution)
	newRelicEvent.TargetState = string(event.ExperimentStepTargetExecution.State)

	return &newRelicEvent, nil
}

func getTargetName(target event_kit_api.ExperimentStepTargetExecution) string {
	if values, ok := target.TargetAttributes["steadybit.label"]; ok {
		return values[0]
	}
	return target.TargetName
}

func addBaseProperties(newRelicEvent *types.EventIngest, event *event_kit_api.EventRequestBody) {
	newRelicEvent.EnvironmentName = event.Environment.Name
	if event.Team != nil {
		newRelicEvent.TeamName = event.Team.Name
		newRelicEvent.TeamKey = event.Team.Key
	}
	userPrincipal, isUserPrincipal := event.Principal.(event_kit_api.UserPrincipal)
	if isUserPrincipal {
		newRelicEvent.PrincipalType = userPrincipal.PrincipalType
		newRelicEvent.PrincipalUsername = userPrincipal.Username
		newRelicEvent.PrincipalName = userPrincipal.Name
	}
	accessTokenPrincipal, isAccessTokenPrincipal := event.Principal.(event_kit_api.AccessTokenPrincipal)
	if isAccessTokenPrincipal {
		newRelicEvent.PrincipalType = accessTokenPrincipal.PrincipalType
		newRelicEvent.PrincipalName = accessTokenPrincipal.Name
	}
	batchPrincipal, isBatchPrincipal := event.Principal.(event_kit_api.BatchPrincipal)
	if isBatchPrincipal {
		newRelicEvent.PrincipalType = batchPrincipal.PrincipalType
		newRelicEvent.PrincipalUsername = batchPrincipal.Username
	}
}

func addExperimentExecutionProperties(newRelicEvent *types.EventIngest, experimentExecution *event_kit_api.ExperimentExecution) {
	if experimentExecution == nil {
		return
	}
	newRelicEvent.ExperimentKey = experimentExecution.ExperimentKey
	newRelicEvent.ExperimentName = experimentExecution.Name
	newRelicEvent.ExecutionId = fmt.Sprintf("%g", experimentExecution.ExecutionId)
}

func addStepExecutionProperties(newRelicEvent *types.EventIngest, stepExecution *event_kit_api.ExperimentStepExecution) {
	if stepExecution == nil {
		return
	}
	if stepExecution.Type == event_kit_api.Action {
		newRelicEvent.ActionId = *stepExecution.ActionId
	}
	if stepExecution.ActionName != nil {
		newRelicEvent.ActionName = *stepExecution.ActionName
	}
	if stepExecution.CustomLabel != nil {
		newRelicEvent.ActionCustomLabel = *stepExecution.CustomLabel
	}
}

func addTargetExecutionProperties(newRelicEvent *types.EventIngest, targetExecution *event_kit_api.ExperimentStepTargetExecution) {
	if targetExecution == nil {
		return
	}
	newRelicEvent.ExperimentKey = targetExecution.ExperimentKey
	newRelicEvent.ExecutionId = fmt.Sprintf("%g", targetExecution.ExecutionId)
	newRelicEvent.Target = getTargetName(*targetExecution)
	newRelicEvent.TargetType = targetExecution.TargetType
}

func parseBodyToEventRequestBody(body []byte) (event_kit_api.EventRequestBody, error) {
	var event event_kit_api.EventRequestBody
	err := json.Unmarshal(body, &event)
	return event, err
}
