/*
Copyright 2021 The BFE Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cookie

import (
	"github.com/bfenetworks/ingress-bfe/test/e2e/pkg/kubernetes"
	tstate "github.com/bfenetworks/ingress-bfe/test/e2e/pkg/state"
	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v16"
)

// IMPORTANT: Steps definitions are generated and should not be modified
// by hand but rather through make codegen. DO NOT EDIT.

// InitializeScenario configures the Feature to test
func InitializeScenario(ctx *godog.ScenarioContext) {
	var state *tstate.Scenario
	ctx.Step(`^an Ingress resource in a new random namespace$`, state.AnIngressResourceInANewRandomNamespace)
	ctx.Step(`^The Ingress status shows the IP address or FQDN where it is exposed$`, state.TheIngressStatusShowsTheIPAddressOrFQDNWhereItIsExposed)
	ctx.Step(`^I send a "([^"]*)" request to "([^"]*)" with header$`, state.ISendARequestToWithHeader)
	ctx.Step(`^the response status-code must be (\d+)$`, state.TheResponseStatuscodeMustBe)
	ctx.Step(`^the response must be served by the "([^"]*)" service$`, state.TheResponseMustBeServedByTheService)
	ctx.Step(`^I send a "([^"]*)" request to "([^"]*)"$`, state.ISendARequestTo)

	ctx.BeforeScenario(func(*godog.Scenario) {
		state = tstate.New()
	})

	ctx.AfterScenario(func(*messages.Pickle, error) {
		// delete namespace an all the content
		_ = kubernetes.DeleteNamespace(kubernetes.KubeClient, state.Namespace)
	})
}
