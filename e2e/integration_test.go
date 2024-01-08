// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_test/validate"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
	})
}

func validateDiscovery(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	assert.NoError(t, validate.ValidateEndpointReferences("/", e.Client))
}
