// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extincident

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
	"github.com/steadybit/extension-newrelic/extaccount"
	"github.com/steadybit/extension-newrelic/types"
	"k8s.io/utils/strings/slices"
	"time"
)

type IncidentCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[IncidentCheckState]           = (*IncidentCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[IncidentCheckState] = (*IncidentCheckAction)(nil)
)

type IncidentCheckState struct {
	End                    time.Time
	IncidentPriorityFilter []string
	EntityTagFilter        map[string]string
	AccountId              int64
	Condition              string
	ConditionCheckMode     string
	ConditionCheckSuccess  bool
}

func NewIncidentCheckAction() action_kit_sdk.Action[IncidentCheckState] {
	return &IncidentCheckAction{}
}

func (m *IncidentCheckAction) NewEmptyState() IncidentCheckState {
	return IncidentCheckState{}
}

func (m *IncidentCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          IncidentCheckActionId,
		Label:       "Incident Check",
		Description: "Checks for the existence of incidents in New Relic.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(incidentCheckActionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          extaccount.AccountTargetId,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "by account id",
					Query: "new-relic.account.id=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("monitoring"),
		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr(""),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Order:        extutil.Ptr(1),
				Required:     extutil.Ptr(true),
			},
			{
				Name:        "incidentPriorityFilter",
				Label:       "Incident Priority Filter",
				Description: extutil.Ptr("Filter incidents by priority."),
				Type:        action_kit_api.StringArray,
				Order:       extutil.Ptr(2),
				Required:    extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Low",
						Value: "LOW",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Medium",
						Value: "MEDIUM",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "High",
						Value: "HIGH",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Critical",
						Value: "CRITICAL",
					},
				}),
				DefaultValue: extutil.Ptr("[\"LOW\",\"MEDIUM\",\"HIGH\",\"CRITICAL\"]"),
			},
			{
				Name:        "entityTagFilter",
				Label:       "Entity Tag Filter",
				Description: extutil.Ptr("Filter incidents by a list of required tags of their related entities"),
				Type:        action_kit_api.KeyValue,
				Order:       extutil.Ptr(3),
				Required:    extutil.Ptr(false),
			},
			{
				Name:        "condition",
				Label:       "Condition",
				Description: extutil.Ptr(""),
				Type:        action_kit_api.String,
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "No check, only show incidents",
						Value: conditionShowOnly,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "No incidents expected",
						Value: conditionNoIncidents,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least one incident expected",
						Value: conditionAtLeastOneIncident,
					},
				}),
				DefaultValue: extutil.Ptr(conditionShowOnly),
				Order:        extutil.Ptr(4),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "conditionCheckMode",
				Label:        "Condition Check Mode",
				Description:  extutil.Ptr("Should the step succeed if the condition is met at least once or all the time?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr(conditionCheckModeAllTheTime),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "All the time",
						Value: conditionCheckModeAllTheTime,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least once",
						Value: conditionCheckModeAtLeastOnce,
					},
				}),
				Required: extutil.Ptr(true),
				Order:    extutil.Ptr(5),
			},
		},
		Widgets: extutil.Ptr([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "New Relic Incidents",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: "id",
				},
				Label: action_kit_api.StateOverTimeWidgetLabelConfig{
					From: "title",
				},
				State: action_kit_api.StateOverTimeWidgetStateConfig{
					From: "state",
				},
				Tooltip: action_kit_api.StateOverTimeWidgetTooltipConfig{
					From: "tooltip",
				},
				Url: extutil.Ptr(action_kit_api.StateOverTimeWidgetUrlConfig{
					From: extutil.Ptr("url"),
				}),
				Value: extutil.Ptr(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: extutil.Ptr(true),
				}),
			},
		}),
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: extutil.Ptr(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("5s"),
		}),
	}
}

func (m *IncidentCheckAction) Prepare(_ context.Context, state *IncidentCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	duration := request.Config["duration"].(float64)
	state.End = time.Now().Add(time.Millisecond * time.Duration(duration))
	state.IncidentPriorityFilter = extutil.ToStringArray(request.Config["incidentPriorityFilter"])
	state.AccountId = extutil.ToInt64(request.Target.Attributes["new-relic.account.id"][0])

	if request.Config["entityTagFilter"] != nil {
		entityTagFilter, err := extutil.ToKeyValue(request.Config, "entityTagFilter")
		if err != nil {
			log.Error().Err(err).Msg("Failed to parse entityTagFilter")
			return nil, err
		}
		state.EntityTagFilter = entityTagFilter
	}

	if request.Config["condition"] != nil {
		state.Condition = fmt.Sprintf("%v", request.Config["condition"])
	}
	if request.Config["conditionCheckMode"] != nil {
		state.ConditionCheckMode = fmt.Sprintf("%v", request.Config["conditionCheckMode"])
	}

	return nil, nil
}

func (m *IncidentCheckAction) Start(ctx context.Context, state *IncidentCheckState) (*action_kit_api.StartResult, error) {
	statusResult, err := IncidentCheckStatus(ctx, state, &config.Config)
	if statusResult == nil {
		return nil, err
	}
	startResult := action_kit_api.StartResult{
		Artifacts: statusResult.Artifacts,
		Error:     statusResult.Error,
		Messages:  statusResult.Messages,
		Metrics:   statusResult.Metrics,
	}
	return &startResult, err
}

func (m *IncidentCheckAction) Status(ctx context.Context, state *IncidentCheckState) (*action_kit_api.StatusResult, error) {
	return IncidentCheckStatus(ctx, state, &config.Config)
}

type IncidentsApi interface {
	GetIncidents(ctx context.Context, incidentPriorityFilter []string, accountId int64) ([]types.Incident, error)
	GetEntityTags(ctx context.Context, guid string) (map[string][]string, error)
}

func IncidentCheckStatus(ctx context.Context, state *IncidentCheckState, api IncidentsApi) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	incidents, err := api.GetIncidents(ctx, state.IncidentPriorityFilter, state.AccountId)
	if err != nil {
		return nil, extension_kit.ToError("Failed to get incidents from New Relic.", err)
	}

	filteredIncidents := make([]types.Incident, 0)
	if len(state.EntityTagFilter) == 0 {
		filteredIncidents = incidents
	} else {
		for _, incident := range incidents {
			if matchesEntityTagFilter(ctx, api, incident, state.EntityTagFilter) {
				filteredIncidents = append(filteredIncidents, incident)
			}
		}
	}

	completed := now.After(state.End)
	var checkError *action_kit_api.ActionKitError
	if state.ConditionCheckMode == conditionCheckModeAllTheTime {
		if state.Condition == conditionNoIncidents && len(filteredIncidents) > 0 {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("No incident expected, but %d incidents found.", len(filteredIncidents)),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}
		if state.Condition == conditionAtLeastOneIncident && len(filteredIncidents) == 0 {
			checkError = extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "At least one incident expected, but no incidents found.",
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}

	} else if state.ConditionCheckMode == conditionCheckModeAtLeastOnce {
		if state.Condition == conditionNoIncidents && len(filteredIncidents) == 0 {
			state.ConditionCheckSuccess = true
		}
		if state.Condition == conditionAtLeastOneIncident && len(filteredIncidents) > 0 {
			state.ConditionCheckSuccess = true
		}
		if completed && !state.ConditionCheckSuccess {
			if state.Condition == conditionNoIncidents {
				checkError = extutil.Ptr(action_kit_api.ActionKitError{
					Title:  "No incident expected, but incidents found.",
					Status: extutil.Ptr(action_kit_api.Failed),
				})
			} else if state.Condition == conditionAtLeastOneIncident {
				checkError = extutil.Ptr(action_kit_api.ActionKitError{
					Title:  "At least one incident expected, but no incidents found.",
					Status: extutil.Ptr(action_kit_api.Failed),
				})
			}
		}
	}

	metrics := make([]action_kit_api.Metric, 0)
	for _, incident := range filteredIncidents {
		metrics = append(metrics, toMetric(incident, now))
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics:   extutil.Ptr(metrics),
	}, nil
}

func matchesEntityTagFilter(ctx context.Context, api IncidentsApi, incident types.Incident, filter map[string]string) bool {
	tags, err := api.GetEntityTags(ctx, incident.EntityGuids)
	if err != nil {
		log.Error().Str("entityGuid", incident.EntityGuids).Str("incident", incident.IncidentId).Err(err).Msg("Failed to get entity tags from New Relic - ignoring incident.")
		return false
	}

	for key, value := range filter {
		if _, ok := tags[key]; !ok {
			log.Debug().Str("entityGuid", incident.EntityGuids).Str("incident", incident.IncidentId).Str("key", key).Msg("Entity does not have tag - ignoring incident.")
			return false
		}
		if !slices.Contains(tags[key], value) {
			log.Debug().Str("entityGuid", incident.EntityGuids).Str("incident", incident.IncidentId).Str("key", key).Str("value", value).Msg("Entity does not have tag value - ignoring incident.")
			return false
		}
	}
	return true
}

func toMetric(incident types.Incident, now time.Time) action_kit_api.Metric {
	title := incident.EntityNames
	if len(title) == 0 {
		title = incident.Title
	}
	return action_kit_api.Metric{
		Name: extutil.Ptr("new_relic_incidents"),
		Metric: map[string]string{
			"id":      incident.IncidentId,
			"title":   title,
			"state":   "danger",
			"tooltip": fmt.Sprintf("Priority: %s\nTitle: %s\nDescription: %s\nEntity: %s", incident.Priority, incident.Title, incident.Description[0], incident.EntityNames),
		},
		Timestamp: now,
		Value:     0,
	}
}
