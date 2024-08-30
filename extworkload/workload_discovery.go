// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extworkload

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
	"github.com/steadybit/extension-newrelic/types"
	"time"
)

type workloadDiscovery struct {
}

var (
	_ discovery_kit_sdk.TargetDescriber    = (*workloadDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*workloadDiscovery)(nil)
)

func NewWorkloadDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &workloadDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 1*time.Minute),
	)
}
func (d *workloadDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: WorkloadTargetId,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *workloadDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       WorkloadTargetId,
		Label:    discovery_kit_api.PluralLabel{One: "New Relic Workload", Other: "New Relic Workloads"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(workloadIcon),
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "steadybit.label"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "steadybit.label",
					Direction: "ASC",
				},
			},
		},
	}
}

func (d *workloadDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "new-relic.workload.name",
			Label: discovery_kit_api.PluralLabel{
				One:   "New Relic Workload Name",
				Other: "New Relic Workload Names",
			},
		},
	}
}

func (d *workloadDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllWorkloads(ctx, &config.Config), nil
}

type GetWorkloadsApi interface {
	GetAccountIds(ctx context.Context) ([]int64, error)
	GetWorkloads(ctx context.Context, accountId int64) ([]types.Workload, error)
}

func getAllWorkloads(ctx context.Context, api GetWorkloadsApi) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 100)

	accounts, err := api.GetAccountIds(ctx)
	if err != nil {
		log.Err(err).Msgf("Failed to get accounts from New Relic.")
		return result
	}

	for _, accountId := range accounts {
		workloads, err := api.GetWorkloads(ctx, accountId)
		if err != nil {
			log.Err(err).Int64("accountId", accountId).Msgf("Failed to get workloads from New Relic.")
			return result
		}

		for _, workload := range workloads {
			result = append(result, toTarget(workload, accountId))
		}
	}

	return result
}

func toTarget(workload types.Workload, accountId int64) discovery_kit_api.Target {
	label := fmt.Sprintf("%s (%d)", workload.Name, accountId)

	attributes := make(map[string][]string)
	attributes["steadybit.label"] = []string{label}
	attributes["new-relic.workload.name"] = []string{workload.Name}
	attributes["new-relic.workload.guid"] = []string{workload.Guid}
	attributes["new-relic.workload.permalink"] = []string{workload.Permalink}
	attributes["new-relic.workload.account"] = []string{fmt.Sprintf("%d", accountId)}

	return discovery_kit_api.Target{
		Id:         workload.Guid,
		Label:      label,
		TargetType: WorkloadTargetId,
		Attributes: attributes,
	}
}
