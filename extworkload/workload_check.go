// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extworkload

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
)

type WorkloadCheckAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[WorkloadCheckState]           = (*WorkloadCheckAction)(nil)
	_ action_kit_sdk.ActionWithStatus[WorkloadCheckState] = (*WorkloadCheckAction)(nil)
)

type WorkloadCheckState struct {
	Start              time.Time
	End                time.Time
	Target             action_kit_api.Target
	ExpectedStates     []string
	ConditionCheckMode string
	ObservedStates     map[string]bool
}

func NewWorkloadCheckAction() action_kit_sdk.Action[WorkloadCheckState] {
	return &WorkloadCheckAction{}
}

func (m *WorkloadCheckAction) NewEmptyState() WorkloadCheckState {
	return WorkloadCheckState{}
}

func (m *WorkloadCheckAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          WorkloadCheckActionId,
		Label:       "Workload Check",
		Description: "Checks the status of a workload.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        new(workloadCheckActionIcon),
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType:          WorkloadTargetId,
			QuantityRestriction: extutil.Ptr(action_kit_api.QuantityRestrictionAll),
			SelectionTemplates: new([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "workload name",
					Query: "new-relic.workload.name=\"\"",
				},
			}),
		}),
		Technology: new("New Relic"),

		Kind:        action_kit_api.Check,
		TimeControl: action_kit_api.TimeControlInternal,
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  new(""),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("30s"),
				Order:        new(1),
				Required:     new(true),
			},
			{
				Name:        "expectedStates",
				Label:       "Expected States",
				Description: new("Which states are expected? If you select all states, the action will always succeed and just show the current state in a graph."),
				Type:        action_kit_api.ActionParameterTypeStringArray,
				Order:       new(3),
				Required:    new(true),
				Advanced:    new(false),
				Options: new([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Operational",
						Value: "OPERATIONAL",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Degraded",
						Value: "DEGRADED",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Disrupted",
						Value: "DISRUPTED",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Critical",
						Value: "CRITICAL",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Unknown",
						Value: "UNKNOWN",
					},
				}),
				DefaultValue: new("[\"OPERATIONAL\",\"DEGRADED\",\"DISRUPTED\",\"CRITICAL\",\"UNKNOWN\"]"),
			},
			{
				Name:         "conditionCheckMode",
				Label:        "Condition Check Mode",
				Description:  new("Should the step succeed if the condition is met at least once or all the time?"),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: new(conditionCheckModeAllTheTime),
				Options: new([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "All the time",
						Value: conditionCheckModeAllTheTime,
					},
					action_kit_api.ExplicitParameterOption{
						Label: "At least once",
						Value: conditionCheckModeAtLeastOnce,
					},
				}),
				Required: new(true),
				Order:    new(4),
			},
		},
		Widgets: new([]action_kit_api.Widget{
			action_kit_api.StateOverTimeWidget{
				Type:  action_kit_api.ComSteadybitWidgetStateOverTime,
				Title: "New Relic Workload State",
				Identity: action_kit_api.StateOverTimeWidgetIdentityConfig{
					From: "newrelic.workload-id",
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
				Url: new(action_kit_api.StateOverTimeWidgetUrlConfig{
					From: new("url"),
				}),
				Value: new(action_kit_api.StateOverTimeWidgetValueConfig{
					Hide: new(true),
				}),
			},
		}),
		Prepare: action_kit_api.MutatingEndpointReference{},
		Start:   action_kit_api.MutatingEndpointReference{},
		Status: new(action_kit_api.MutatingEndpointReferenceWithCallInterval{
			CallInterval: new("5s"),
		}),
	}
}

func (m *WorkloadCheckAction) Prepare(_ context.Context, state *WorkloadCheckState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	duration := request.Config["duration"].(float64)
	state.Start = time.Now()
	state.End = time.Now().Add(time.Millisecond * time.Duration(duration))
	state.Target = *request.Target
	state.ExpectedStates = extutil.ToStringArray(request.Config["expectedStates"])
	state.ObservedStates = make(map[string]bool)
	if request.Config["conditionCheckMode"] != nil {
		state.ConditionCheckMode = fmt.Sprintf("%v", request.Config["conditionCheckMode"])
	}
	return nil, nil
}

func (m *WorkloadCheckAction) Start(ctx context.Context, state *WorkloadCheckState) (*action_kit_api.StartResult, error) {
	statusResult, err := WorkloadCheckStatus(ctx, state, &config.Config)
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

func (m *WorkloadCheckAction) Status(ctx context.Context, state *WorkloadCheckState) (*action_kit_api.StatusResult, error) {
	return WorkloadCheckStatus(ctx, state, &config.Config)
}

type WorkloadStatusApi interface {
	GetWorkloadStatus(ctx context.Context, workloadGuid string, accountId int64) (*string, error)
}

func WorkloadCheckStatus(ctx context.Context, state *WorkloadCheckState, api WorkloadStatusApi) (*action_kit_api.StatusResult, error) {
	now := time.Now()
	guid := state.Target.Attributes["new-relic.workload.guid"][0]
	accountId := extutil.ToInt64(state.Target.Attributes["new-relic.workload.account"][0])
	status, err := api.GetWorkloadStatus(ctx, guid, accountId)
	if err != nil {
		return nil, extension_kit.ToError("Failed to get workload status from New Relic.", err)
	}

	completed := now.After(state.End)
	var checkError *action_kit_api.ActionKitError
	if state.ConditionCheckMode == conditionCheckModeAllTheTime {
		if !slices.Contains(state.ExpectedStates, *status) {
			checkError = new(action_kit_api.ActionKitError{
				Title:  fmt.Sprintf("Unexpected status %s", *status),
				Status: extutil.Ptr(action_kit_api.Failed),
			})
		}
	} else if state.ConditionCheckMode == conditionCheckModeAtLeastOnce {
		state.ObservedStates[*status] = true
		if completed {
			checkSuccess := false
			for _, expectedState := range state.ExpectedStates {
				if state.ObservedStates[expectedState] {
					checkSuccess = true
					break
				}
			}
			if !checkSuccess {
				checkError = new(action_kit_api.ActionKitError{
					Title:  fmt.Sprintf("Expected state missing. Expected: %s, Observed: %s", strings.Join(state.ExpectedStates, ", "), keysToString(state.ObservedStates)),
					Status: extutil.Ptr(action_kit_api.Failed),
				})
			}
		}
	}

	return &action_kit_api.StatusResult{
		Completed: completed,
		Error:     checkError,
		Metrics:   createMetric(state.Target, status, now),
	}, nil
}

func keysToString(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func createMetric(target action_kit_api.Target, status *string, now time.Time) *action_kit_api.Metrics {
	tooltip := fmt.Sprintf("Status: %s", *status)
	metric := action_kit_api.Metric{
		Name: new("new_relic_workload"),
		Metric: map[string]string{
			"newrelic.workload-id": target.Attributes["new-relic.workload.guid"][0],
			"title":                target.Attributes["new-relic.workload.name"][0],
			"state":                getState(status),
			"tooltip":              tooltip,
			"url":                  target.Attributes["new-relic.workload.permalink"][0],
		},
		Timestamp: now,
		Value:     0,
	}
	return new(action_kit_api.Metrics{metric})
}

func getState(status *string) string {
	if status == nil {
		return "info"
	} else if *status == "OPERATIONAL" {
		return "success"
	} else if *status == "DISRUPTED" || *status == "CRITICAL" {
		return "danger"
	}
	return "info" //UNKNOWN,DEGRADED
}
