/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	"github.com/rs/zerolog"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/event-kit/go/event_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthealth"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/steadybit/extension-kit/extsignals"
	"github.com/steadybit/extension-newrelic/config"
	"github.com/steadybit/extension-newrelic/extaccount"
	"github.com/steadybit/extension-newrelic/extevents"
	"github.com/steadybit/extension-newrelic/extincident"
	"github.com/steadybit/extension-newrelic/extworkload"
)

func main() {
	extlogging.InitZeroLog()
	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)
	config.ParseConfiguration()
	config.ValidateConfiguration()

	exthealth.SetReady(false)
	exthealth.StartProbes(8091)

	exthttp.RegisterHttpHandler("/", exthttp.GetterAsHandler(getExtensionList))
	discovery_kit_sdk.Register(extworkload.NewWorkloadDiscovery())
	discovery_kit_sdk.Register(extaccount.NewAccountDiscovery())
	action_kit_sdk.RegisterAction(extworkload.NewWorkloadCheckAction())
	action_kit_sdk.RegisterAction(extaccount.NewCreateMutingRuleAction())
	action_kit_sdk.RegisterAction(extincident.NewIncidentCheckAction())
	extevents.RegisterEventListenerHandlers()

	extsignals.ActivateSignalHandlers()
	action_kit_sdk.RegisterCoverageEndpoints()
	exthealth.SetReady(true)

	exthttp.Listen(exthttp.ListenOpts{
		Port: 8090,
	})
}

// ExtensionListResponse exists to merge the possible root path responses supported by the
// various extension kits. In this case, the response for ActionKit, DiscoveryKit and EventKit.
type ExtensionListResponse struct {
	action_kit_api.ActionList       `json:",inline"`
	discovery_kit_api.DiscoveryList `json:",inline"`
	event_kit_api.EventListenerList `json:",inline"`
}

func getExtensionList() ExtensionListResponse {
	return ExtensionListResponse{
		ActionList:    action_kit_sdk.GetActionList(),
		DiscoveryList: discovery_kit_sdk.GetDiscoveryList(),
		EventListenerList: event_kit_api.EventListenerList{
			EventListeners: []event_kit_api.EventListener{
				{
					Method:   "POST",
					Path:     "/events/experiment-started",
					ListenTo: []string{"experiment.execution.created"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-completed",
					ListenTo: []string{"experiment.execution.completed", "experiment.execution.failed", "experiment.execution.canceled", "experiment.execution.errored"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-step-started",
					ListenTo: []string{"experiment.execution.step-started"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-target-started",
					ListenTo: []string{"experiment.execution.target-started"},
				},
				{
					Method:   "POST",
					Path:     "/events/experiment-target-completed",
					ListenTo: []string{"experiment.execution.target-completed", "experiment.execution.target-canceled", "experiment.execution.target-errored", "experiment.execution.target-failed"},
				},
			},
		},
	}
}
