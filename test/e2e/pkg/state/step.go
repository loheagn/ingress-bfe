// Copyright (c) 2022 The BFE Authors.
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

package state

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/bfenetworks/ingress-bfe/test/e2e/pkg/kubernetes"
	"github.com/cucumber/godog"
	netv1 "k8s.io/api/networking/v1"
)

func (s *Scenario) AnIngressResourceInANewRandomNamespace(spec *godog.DocString) error {
	ingress, err := s.beforeCreateAnIngressResourceInANewRandomNamespace(spec)
	if err != nil {
		log.Printf("err beforeCreateAnIngressResourceInANewRandomNamespace: %s", err.Error())
		return err
	}
	log.Println("beforeCreateAnIngressResourceInANewRandomNamespace")

	err = kubernetes.NewIngress(kubernetes.KubeClient, s.Namespace, ingress)
	if err != nil {
		log.Printf("err kubernetes.NewIngress: %s", err.Error())
		return err
	}
	log.Println("kubernetes.NewIngress")

	s.IngressName = ingress.GetName()
	log.Println("ingress.GetName()")

	return nil
}

func (s *Scenario) AnIngressResourceInANewRandomNamespaceShouldNotCreate(spec *godog.DocString) error {
	ingress, err := s.beforeCreateAnIngressResourceInANewRandomNamespace(spec)
	if err != nil {
		return err
	}

	err = kubernetes.NewIngress(kubernetes.KubeClient, s.Namespace, ingress)
	if err == nil {
		return fmt.Errorf("create ingress should return error")
	}

	s.IngressName = ingress.GetName()

	return nil
}

func (s *Scenario) beforeCreateAnIngressResourceInANewRandomNamespace(spec *godog.DocString) (*netv1.Ingress, error) {
	ns, err := kubernetes.NewNamespace(kubernetes.KubeClient)
	if err != nil {
		log.Printf("kubernetes.NewNamespace: %s", err.Error())
		return nil, err
	}
	log.Println("kubernetes.NewNamespace")
	log.Println(s == nil)
	log.Println(s.Namespace)

	s.Namespace = ns

	log.Println("s.Namespace = ns")

	log.Println(s.Namespace, spec.Content)

	ingress, err := kubernetes.IngressFromManifest(s.Namespace, spec.Content)
	if err != nil {
		log.Printf("kubernetes.IngressFromManifest: %s", err.Error())
		return nil, err
	}
	log.Println("kubernetes.IngressFromManifest")

	err = kubernetes.DeploymentsFromIngress(kubernetes.KubeClient, ingress)
	if err != nil {
		log.Printf("kubernetes.DeploymentsFromIngress: %s", err.Error())
		return nil, err
	}
	log.Println("kubernetes.DeploymentsFromIngress")
	return ingress, nil
}

func (s *Scenario) TheIngressStatusShowsTheIPAddressOrFQDNWhereItIsExposed() error {
	ingress, err := kubernetes.WaitForIngressAddress(kubernetes.KubeClient, s.Namespace, s.IngressName)
	if err != nil {
		return err
	}

	s.IPOrFQDN = ingress

	time.Sleep(3 * time.Second)

	return err
}

func (s *Scenario) ISendARequestTo(method string, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return s.CaptureRoundTrip(method, u.Scheme, u.Host, u.Path, nil)
}

func (s *Scenario) ISendARequestToWithHeader(method, rawURL string, header *godog.DocString) error {
	var headerInfo http.Header
	if header.Content != "" {
		if err := json.Unmarshal([]byte(header.Content), &headerInfo); err != nil {
			return fmt.Errorf("err in jsonEncoder.Encode: %s", err.Error())
		}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return s.CaptureRoundTrip(method, u.Scheme, u.Host, u.Path, headerInfo)
}

func (s *Scenario) TheIngressStatusShouldNotBeSuccess() error {
	_, err := kubernetes.WaitForIngressAddress(kubernetes.KubeClient, s.Namespace, s.IngressName)
	if err == nil {
		return fmt.Errorf("create ingress should return error")
	}

	return nil
}

func (s *Scenario) TheResponseStatuscodeMustBe(statusCode int) error {
	return s.AssertStatusCode(statusCode)
}

func (s *Scenario) TheResponseLocationMustBe(location string) error {
	return s.AssertResponseHeader("Location", location)
}

func (s *Scenario) TheResponseMustBeServedByTheService(service string) error {
	return s.AssertServedBy(service)
}

func (s *Scenario) TheBackendDeploymentForTheIngressResourceIsScaledTo(deployment string, replicas int) error {
	return kubernetes.ScaleIngressBackendDeployment(kubernetes.KubeClient, s.Namespace, s.IngressName, deployment, replicas)
}
