// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package extevents

import (
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func Test_addBaseProperties(t *testing.T) {
	type args struct {
		event event_kit_api.EventRequestBody
	}

	eventTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		args args
		want types.EventIngest
	}{
		{
			name: "Successfully add base properties",
			args: args{
				event: event_kit_api.EventRequestBody{
					Environment: extutil.Ptr(event_kit_api.Environment{
						Id:   "test",
						Name: "gateway",
					}),
					EventName: "experiment.started",
					EventTime: eventTime,
					Id:        uuid.MustParse("ccf6a26e-588f-446e-8eaa-d16b086e150e"),
					Principal: event_kit_api.UserPrincipal{
						Email:         extutil.Ptr("email"),
						Name:          "Peter",
						Username:      "Pan",
						PrincipalType: string(event_kit_api.User),
					},
					Team: extutil.Ptr(event_kit_api.Team{
						Id:   "test",
						Key:  "test",
						Name: "gateway",
					}),
					Tenant: event_kit_api.Tenant{
						Key:  "key",
						Name: "name",
					},
				},
			},
			want: types.EventIngest{
				EnvironmentName:   "gateway",
				PrincipalName:     "Peter",
				PrincipalType:     "user",
				PrincipalUsername: "Pan",
				TeamKey:           "test",
				TeamName:          "gateway",
			},
		},
		{
			name: "Successfully add base properties without Principal",
			args: args{
				event: event_kit_api.EventRequestBody{
					Environment: extutil.Ptr(event_kit_api.Environment{
						Id:   "test",
						Name: "gateway",
					}),
					EventName: "experiment.started",
					EventTime: eventTime,
					Id:        uuid.MustParse("ccf6a26e-588f-446e-8eaa-d16b086e150e"),
					Principal: event_kit_api.AccessTokenPrincipal{
						Name:          "MyFancyToken",
						PrincipalType: string(event_kit_api.AccessToken),
					},
					Team: extutil.Ptr(event_kit_api.Team{
						Id:   "test",
						Key:  "test",
						Name: "gateway",
					}),
					Tenant: event_kit_api.Tenant{
						Key:  "key",
						Name: "name",
					},
				},
			},
			want: types.EventIngest{
				EnvironmentName: "gateway",
				PrincipalName:   "MyFancyToken",
				PrincipalType:   "access_token",
				TeamKey:         "test",
				TeamName:        "gateway",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := types.EventIngest{}
			addBaseProperties(&props, &tt.args.event)
			assert.Equalf(t, tt.want, props, "addBaseProperties(%v)", tt.args.event)
		})
	}
}

func Test_addExperimentExecutionProperties(t *testing.T) {
	type args struct {
		event event_kit_api.EventRequestBody
	}

	eventTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	endedTime := time.Date(2021, 1, 1, 0, 2, 0, 0, time.UTC)
	startedTime := time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC)
	tests := []struct {
		name string
		args args
		want types.EventIngest
	}{
		{
			name: "Successfully add execution properties",
			args: args{
				event: event_kit_api.EventRequestBody{
					ExperimentExecution: extutil.Ptr(event_kit_api.ExperimentExecution{
						EndedTime:     extutil.Ptr(endedTime),
						ExecutionId:   42,
						ExperimentKey: "ExperimentKey",
						Reason:        extutil.Ptr("Reason"),
						ReasonDetails: extutil.Ptr("ReasonDetails"),
						Hypothesis:    "Hypothesis",
						Name:          "Name",
						PreparedTime:  eventTime,
						StartedTime:   startedTime,
					}),
				},
			},
			want: types.EventIngest{
				ExecutionId:    "42",
				ExperimentKey:  "ExperimentKey",
				ExperimentName: "Name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := types.EventIngest{}
			addExperimentExecutionProperties(&props, tt.args.event.ExperimentExecution)
			assert.Equalf(t, tt.want, props, "addExperimentExecutionProperties(%v)", tt.args.event)
		})
	}
}

func Test_addStepExecutionProperties(t *testing.T) {
	type args struct {
		w             http.ResponseWriter
		stepExecution event_kit_api.ExperimentStepExecution
	}

	endedTime := time.Date(2021, 1, 1, 0, 2, 0, 0, time.UTC)
	startedTime := time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC)
	tests := []struct {
		name string
		args args
		want types.EventIngest
	}{
		{
			name: "Successfully add properties for started attack",
			args: args{
				stepExecution: event_kit_api.ExperimentStepExecution{
					Id:          uuid.UUID{},
					Type:        event_kit_api.Action,
					ActionId:    extutil.Ptr("com.steadybit.action.example"),
					ActionName:  extutil.Ptr("example-action"),
					ActionKind:  extutil.Ptr(event_kit_api.Attack),
					CustomLabel: extutil.Ptr("My very own label"),
					State:       event_kit_api.ExperimentStepExecutionStateFailed,
					EndedTime:   extutil.Ptr(endedTime),
					StartedTime: extutil.Ptr(startedTime),
				},
			},
			want: types.EventIngest{
				ActionCustomLabel: "My very own label",
				ActionId:          "com.steadybit.action.example",
				ActionName:        "example-action",
			},
		},
		{
			name: "Successfully add properties for not yet started attack",
			args: args{
				stepExecution: event_kit_api.ExperimentStepExecution{
					Id:         uuid.UUID{},
					Type:       event_kit_api.Action,
					ActionId:   extutil.Ptr("com.steadybit.action.example"),
					ActionKind: extutil.Ptr(event_kit_api.Attack),
					State:      event_kit_api.ExperimentStepExecutionStateCompleted,
				},
			},
			want: types.EventIngest{
				ActionId: "com.steadybit.action.example",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := types.EventIngest{}
			addStepExecutionProperties(&props, &tt.args.stepExecution)
			assert.Equalf(t, tt.want, props, "addStepExecutionProperties(%v)", tt.args.stepExecution)
		})
	}
}

func Test_addTargetExecutionProperties(t *testing.T) {
	mockLoader := ttlcache.LoaderFunc[string, []int64](
		func(c *ttlcache.Cache[string, []int64], key string) *ttlcache.Item[string, []int64] {
			return c.Set(key, []int64{1234, 5678}, ttlcache.DefaultTTL)
		},
	)
	accountCache = ttlcache.New[string, []int64](
		ttlcache.WithLoader[string, []int64](mockLoader),
		ttlcache.WithTTL[string, []int64](30*time.Minute),
	)

	type args struct {
		w      http.ResponseWriter
		target event_kit_api.ExperimentStepTargetExecution
	}

	endedTime := time.Date(2021, 1, 1, 0, 2, 0, 0, time.UTC)
	startedTime := time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC)
	id := uuid.New()
	tests := []struct {
		name string
		args args
		want types.EventIngest
	}{
		{
			name: "Successfully get properties for container targets",
			args: args{
				target: event_kit_api.ExperimentStepTargetExecution{
					ExecutionId:   42,
					ExperimentKey: "ExperimentKey",
					Id:            id,
					State:         "completed",
					AgentHostname: "Agent-1",
					TargetAttributes: map[string][]string{
						"steadybit.label": {"example-label"},
					},
					TargetName:  "Container",
					TargetType:  "com.steadybit.extension_container.container",
					StartedTime: &startedTime,
					EndedTime:   &endedTime,
				},
			},
			want: types.EventIngest{
				ExecutionId:   "42",
				ExperimentKey: "ExperimentKey",
				Target:        "example-label",
				TargetType:    "com.steadybit.extension_container.container",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := types.EventIngest{}
			addTargetExecutionProperties(&props, &tt.args.target)
			assert.Equalf(t, tt.want, props, "addTargetExecutionProperties(%v)", tt.args.target)
		})
	}
}

func Test_onExperimentStepStarted(t *testing.T) {
	eventTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	startedTime := time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC)
	endedTime := time.Date(2021, 1, 1, 0, 7, 0, 0, time.UTC)
	stepId := uuid.MustParse("ccf6a26e-588f-446e-8eaa-d16b086e150e")

	type args struct {
		stepEvent   event_kit_api.EventRequestBody
		targetEvent event_kit_api.EventRequestBody
	}
	tests := []struct {
		name string
		args args
		want *types.EventIngest
	}{
		{
			name: "should emit event for experiment target started",
			args: args{
				stepEvent: event_kit_api.EventRequestBody{
					Environment: extutil.Ptr(event_kit_api.Environment{
						Id:   "test",
						Name: "gateway",
					}),
					EventName: "experiment.step.started",
					EventTime: eventTime,
					Id:        stepId,
					ExperimentStepExecution: extutil.Ptr(event_kit_api.ExperimentStepExecution{
						ExecutionId:   42,
						ExperimentKey: "ExperimentKey",
						Id:            stepId,
						ActionId:      extutil.Ptr("some_action_id"),
						ActionName:    extutil.Ptr("started step"),
						CustomLabel:   extutil.Ptr("custom label"),
						ActionKind:    extutil.Ptr(event_kit_api.Attack),
					}),
					Tenant: event_kit_api.Tenant{
						Key:  "key",
						Name: "name",
					},
				},
				targetEvent: event_kit_api.EventRequestBody{
					Environment: extutil.Ptr(event_kit_api.Environment{
						Id:   "test",
						Name: "gateway",
					}),
					EventName: "experiment.step.target.started",
					EventTime: eventTime,
					Id:        stepId,
					ExperimentStepTargetExecution: extutil.Ptr(event_kit_api.ExperimentStepTargetExecution{
						ExecutionId:     42,
						ExperimentKey:   "ExperimentKey",
						StepExecutionId: stepId,
						State:           "completed",
						TargetType:      "type",
						TargetName:      "test",
						StartedTime:     &startedTime,
						EndedTime:       &endedTime,
					}),
					Tenant: event_kit_api.Tenant{
						Key:  "key",
						Name: "name",
					},
				},
			},
			want: &types.EventIngest{
				EventType:         types.EventTypeAttackStarted,
				EnvironmentName:   "gateway",
				ExecutionId:       "42",
				ExperimentKey:     "ExperimentKey",
				ActionCustomLabel: "custom label",
				ActionName:        "started step",
				Target:            "test",
				TargetType:        "type",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := onExperimentStepStarted(&tt.args.stepEvent)
			require.NoError(t, err)
			got, err := onExperimentTargetStarted(&tt.args.targetEvent)
			require.NoError(t, err)
			assert.Equalf(t, tt.want, got, "onExperimentTargetStarted - Something is different, take a very close look ;-)")
		})
	}
}
