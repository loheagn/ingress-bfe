// Copyright (c) 2021 The BFE Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/* test for balance builder*/
package builder

import (
	"reflect"
	"testing"
)

import (
	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
)

import (
	k8s "github.com/bfenetworks/ingress-bfe/internal/kubernetes_client"
)

var (
	testServices = map[string]service{
		"service1": {
			Endpoints: []string{"172.0.0.1"},
			Port:      8081,
		},
		"service2": {
			Endpoints: []string{"172.0.0.2"},
			Port:      8082,
		},
	}
)

// service defines its endpoints for testing
type service struct {
	Endpoints []string
	Port      int32
}

func TestBfeBalanceConfigBuilder_Build(t *testing.T) {
	testCases := map[string]interface{}{
		"single": map[string]interface{}{
			"annotation": map[string]interface{}{
				"load_balance": TestBfeBalanceConfigBuilder_Build_CaseLoadBalance,
				"other":        TestBfeBalanceConfigBuilder_Build_CaseNoLoadBalance,
			},
		},
	}

	traverseTestCases(t, testCases)
	monkey.UnpatchAll()
}

// mock function for k8s.KubernetesClient.GetEndpoints()
func mockGetEndpoints(_ *k8s.KubernetesClient, namespace, name string) (*core.Endpoints, error) {
	service, ok := testServices[name]
	if !ok {
		return nil, nil
	}

	addresses := make([]core.EndpointAddress, 0)
	for _, endpoint := range service.Endpoints {
		address := core.EndpointAddress{IP: endpoint}
		addresses = append(addresses, address)
	}
	return &core.Endpoints{
		Subsets: []core.EndpointSubset{
			{
				Addresses: addresses,
				Ports:     []core.EndpointPort{{Port: service.Port}},
			},
		},
	}, nil
}

func TestBfeBalanceConfigBuilder_Build_CaseLoadBalance(t *testing.T) {
	b, err := balanceConfigBuilderGenerator("single/annotation/load_balance")
	if err != nil {
		t.Fatalf("balanceConfigBuilderGenerator(%s): %s",
			"single/annotation/load_balance", err)
	}

	// invoke Build()
	if err := b.Build(); err != nil {
		t.Errorf("Build(): %s", err)
	}

	// verify
	assert.NotNil(t, b.balanceConf.clusterTableConf, "clusterTableConf is empty")
	gslbConf := b.balanceConf.gslbConf
	assert.NotNil(t, gslbConf, "gslbConf is empty")
	assert.NotNil(t, gslbConf.Clusters, "GSLB clusters is empty")
	assert.Equal(t, 1, len(*gslbConf.Clusters))
	t.Logf("clusterTableConf: %s", jsonify(b.balanceConf.clusterTableConf))
	t.Logf("gslbConf: %s", jsonify(b.balanceConf.gslbConf))
}

func TestBfeBalanceConfigBuilder_Build_CaseNoLoadBalance(t *testing.T) {
	b, err := balanceConfigBuilderGenerator("single/annotation/other")
	if err != nil {
		t.Fatalf("balanceConfigBuilderGenerator(%s): %s",
			"single/annotation/load_balance", err)
	}

	// invoke Build()
	if err := b.Build(); err != nil {
		t.Errorf("Build(): %s", err)
	}

	// verify
	assert.NotNil(t, b.balanceConf.clusterTableConf, "clusterTableConf is empty")
	gslbConf := b.balanceConf.gslbConf
	assert.NotNil(t, gslbConf, "gslbConf is empty")
	assert.NotNil(t, gslbConf.Clusters, "GSLB clusters is empty")
	assert.Equal(t, 2, len(*gslbConf.Clusters))
	t.Logf("clusterTableConf: %s", jsonify(b.balanceConf.clusterTableConf))
	t.Logf("gslbConf: %s", jsonify(b.balanceConf.gslbConf))
}

// balanceConfigBuilderGenerator generate balance config builder from file
// Params:
//		name: file name prefix
// Returns:
//		*BfeBalanceConfigBuilder: builder generated by non-conflicting ingresses
// 		error: error for last conflict/wrong ingress
func balanceConfigBuilderGenerator(name string) (*BfeBalanceConfigBuilder, error) {
	client := &k8s.KubernetesClient{}
	monkey.PatchInstanceMethod(reflect.TypeOf(client), "GetEndpoints", mockGetEndpoints)

	// load ingress from file
	ingresses := loadIngress(name)

	// submit ingress to builder
	var submitErr error
	builder := NewBfeBalanceConfigBuilder(client, "0", nil, nil)
	for _, ingress := range ingresses {
		err := builder.Submit(ingress)
		if err != nil {
			submitErr = err
		}
	}
	return builder, submitErr
}