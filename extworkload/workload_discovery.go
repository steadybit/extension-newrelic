// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extworkload

import (
	"context"
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
		Id:         WorkloadTargetId,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func (d *workloadDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       WorkloadTargetId,
		Label:    discovery_kit_api.PluralLabel{One: "New Relic workload", Other: "New Relic workloads"},
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
				One:   "New Relic workload name",
				Other: "New Relic workload names",
			},
		},
	}
}

func (d *workloadDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllWorkloads(ctx, &config.Config), nil
}

type GetWorkloadsApi interface {
	GetWorkloads(ctx context.Context) ([]types.Workload, error)
}

func getAllWorkloads(ctx context.Context, api GetWorkloadsApi) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 500)

	workloads, err := api.GetWorkloads(ctx)
	if err != nil {
		log.Err(err).Msgf("Failed to get workloads from New Relic.")
		return result
	}

	for _, workload := range workloads {
		result = append(result, toTarget(workload))
	}

	return result
}

func toTarget(workload types.Workload) discovery_kit_api.Target {
	id := workload.Guid
	label := workload.Name

	attributes := make(map[string][]string)
	attributes["steadybit.label"] = []string{label}
	attributes["new-relic.workload.name"] = []string{workload.Name}

	return discovery_kit_api.Target{
		Id:         id,
		Label:      label,
		TargetType: WorkloadTargetId,
		Attributes: attributes,
	}
}
