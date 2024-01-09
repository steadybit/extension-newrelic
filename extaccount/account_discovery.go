// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 Steadybit GmbH

package extaccount

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/steadybit/extension-newrelic/config"
	"time"
)

type accountDiscovery struct {
}

var (
	_ discovery_kit_sdk.TargetDescriber    = (*accountDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber = (*accountDiscovery)(nil)
)

func NewAccountDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &accountDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 30*time.Minute),
	)
}
func (d *accountDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         AccountTargetId,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr("10m"),
		},
	}
}

func (d *accountDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:       AccountTargetId,
		Label:    discovery_kit_api.PluralLabel{One: "New Relic Account", Other: "New Relic Account"},
		Category: extutil.Ptr("monitoring"),
		Version:  extbuild.GetSemverVersionStringOrUnknown(),
		Icon:     extutil.Ptr(accountIcon),
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

func (d *accountDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "new-relic.account.id",
			Label: discovery_kit_api.PluralLabel{
				One:   "New Relic Account ID",
				Other: "New Relic Account IDs",
			},
		},
	}
}

func (d *accountDiscovery) DiscoverTargets(ctx context.Context) ([]discovery_kit_api.Target, error) {
	return getAllAccounts(ctx, &config.Config), nil
}

type GetAccountsApi interface {
	GetAccountIds(ctx context.Context) ([]int64, error)
}

func getAllAccounts(ctx context.Context, api GetAccountsApi) []discovery_kit_api.Target {
	result := make([]discovery_kit_api.Target, 0, 100)

	accounts, err := api.GetAccountIds(ctx)
	if err != nil {
		log.Err(err).Msgf("Failed to get accounts from New Relic.")
		return result
	}

	for _, account := range accounts {
		result = append(result, toTarget(account))
	}

	return result
}

func toTarget(accountId int64) discovery_kit_api.Target {
	label := fmt.Sprintf("%d", accountId)

	attributes := make(map[string][]string)
	attributes["steadybit.label"] = []string{label}
	attributes["new-relic.account.id"] = []string{label}

	return discovery_kit_api.Target{
		Id:         label,
		Label:      label,
		TargetType: AccountTargetId,
		Attributes: attributes,
	}
}
