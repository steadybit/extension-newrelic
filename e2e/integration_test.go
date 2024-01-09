// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-newrelic/extworkload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestWithMinikube(t *testing.T) {
	extlogging.InitZeroLog()
	server := createMockNewRelicServer()
	defer server.Close()
	split := strings.SplitAfter(server.URL, ":")
	port := split[len(split)-1]

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-newrelic",
		Port: 8090,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", "logging.level=debug",
				"--set", "newrelic.apiKey=api-key-123",
				"--set", fmt.Sprintf("newrelic.apiBaseUrl=http://host.minikube.internal:%s", port),
				"--set", "newrelic.accountId=41470",
				"--set", fmt.Sprintf("newrelic.insightsApiBaseUrl=http://host.minikube.internal:%s", port),
				"--set", "newrelic.insightsInsertKey=insert-key-123",
			}
		},
	}

	e2e.WithDefaultMinikube(t, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "validate discovery",
			Test: validateDiscovery,
		},
		{
			Name: "check workload",
			Test: testCheckWorkload,
		},
	})
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}

func testCheckWorkload(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	defer func() { Requests = []string{} }()

	target := &action_kit_api.Target{
		Name: "Example Workload",
		Attributes: map[string][]string{
			"new-relic.workload.name":      {"Example Workload"},
			"new-relic.workload.guid":      {"guid-11111"},
			"new-relic.workload.permalink": {"https://one.newrelic.com/redirect/entity/xyz"},
		},
	}

	config := struct {
		Duration           int      `json:"duration"`
		ExpectedStates     []string `json:"expectedStates"`
		ConditionCheckMode string   `json:"conditionCheckMode"`
	}{Duration: 1000, ExpectedStates: []string{"OPERATIONAL", "DEGRADED", "DISRUPTED"}, ConditionCheckMode: "showOnly"}

	executionContext := &action_kit_api.ExecutionContext{}

	action, err := e.RunAction(extworkload.WorkloadCheckActionId, target, config, executionContext)
	defer func() { _ = action.Cancel() }()
	require.NoError(t, err)
	err = action.Wait()
	require.NoError(t, err)

	assert.Eventually(t, func() bool {
		metrics := action.Metrics()
		if metrics == nil {
			return false
		}
		return len(metrics) > 0
	}, 5*time.Second, 500*time.Millisecond)
	metrics := action.Metrics()

	for _, metric := range metrics {
		assert.Equal(t, "guid-11111", metric.Metric["id"])
		assert.Equal(t, "Example Workload", metric.Metric["title"])
	}
}
