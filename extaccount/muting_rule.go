// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extaccount

import (
	"context"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
	"time"
)

type CreateMutingRuleAction struct{}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[CreateMutingRuleState]         = (*CreateMutingRuleAction)(nil)
	_ action_kit_sdk.ActionWithStop[CreateMutingRuleState] = (*CreateMutingRuleAction)(nil)
)

type CreateMutingRuleState struct {
	AccountId     int64
	End           time.Time
	MutingRuleId  *string
	ExperimentKey *string
	ExecutionId   *int
	ExecutionUri  *string
}

func NewCreateMutingRuleAction() action_kit_sdk.Action[CreateMutingRuleState] {
	return &CreateMutingRuleAction{}
}
func (m *CreateMutingRuleAction) NewEmptyState() CreateMutingRuleState {
	return CreateMutingRuleState{}
}

func (m *CreateMutingRuleAction) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          CreateMutingRuleActionId,
		Label:       "Create Muting Rule",
		Description: "Mute your alerts for a given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(createMutingRuleIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			TargetType:          AccountTargetId,
			QuantityRestriction: extutil.Ptr(action_kit_api.All),
			SelectionTemplates: extutil.Ptr([]action_kit_api.TargetSelectionTemplate{
				{
					Label: "by account id",
					Query: "new-relic.account.id=\"\"",
				},
			}),
		}),
		Category:    extutil.Ptr("monitoring"),
		Kind:        action_kit_api.Other,
		TimeControl: action_kit_api.TimeControlExternal,
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
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func (m *CreateMutingRuleAction) Prepare(_ context.Context, state *CreateMutingRuleState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	duration := request.Config["duration"].(float64)
	end := time.Now().Add(time.Millisecond * time.Duration(duration))

	state.AccountId = extutil.ToInt64(request.Target.Attributes["new-relic.account.id"][0])
	state.End = end
	state.ExperimentKey = request.ExecutionContext.ExperimentKey
	state.ExecutionId = request.ExecutionContext.ExecutionId
	state.ExecutionUri = request.ExecutionContext.ExecutionUri
	return nil, nil
}

func (m *CreateMutingRuleAction) Start(ctx context.Context, state *CreateMutingRuleState) (*action_kit_api.StartResult, error) {
	return CreateMutingRuleStart(ctx, state, &config.Config)
}

func (m *CreateMutingRuleAction) Stop(ctx context.Context, state *CreateMutingRuleState) (*action_kit_api.StopResult, error) {
	return CreateMutingRuleStop(ctx, state, &config.Config)
}

type MutingRuleApi interface {
	CreateMutingRule(ctx context.Context, accountId int64, name string, description string, end time.Time) (*string, error)
	DeleteMutingRule(ctx context.Context, accountId int64, mutingRuleId string) error
}

func CreateMutingRuleStart(ctx context.Context, state *CreateMutingRuleState, api MutingRuleApi) (*action_kit_api.StartResult, error) {
	name := fmt.Sprintf("Steadybit %s (%d)", *state.ExperimentKey, *state.ExecutionId)

	mutingRuleId, err := api.CreateMutingRule(ctx, state.AccountId, name, *state.ExecutionUri, state.End)
	if err != nil {
		return nil, extension_kit.ToError("Failed to create muting rule in New Relic.", err)
	}

	state.MutingRuleId = mutingRuleId

	return &action_kit_api.StartResult{
		Messages: &action_kit_api.Messages{
			action_kit_api.Message{Level: extutil.Ptr(action_kit_api.Info), Message: fmt.Sprintf("Muting rule created. (id %s)", *state.MutingRuleId)},
		},
	}, nil
}

func CreateMutingRuleStop(ctx context.Context, state *CreateMutingRuleState, api MutingRuleApi) (*action_kit_api.StopResult, error) {
	if state.MutingRuleId == nil {
		return nil, nil
	}

	err := api.DeleteMutingRule(ctx, state.AccountId, *state.MutingRuleId)
	if err != nil {
		return nil, extension_kit.ToError("Failed to delete muting rule in New Relic.", err)
	}
	return &action_kit_api.StopResult{
		Messages: &action_kit_api.Messages{
			action_kit_api.Message{Level: extutil.Ptr(action_kit_api.Info), Message: fmt.Sprintf("Muting rule deleted. (id %s)", *state.MutingRuleId)},
		},
	}, nil
}
