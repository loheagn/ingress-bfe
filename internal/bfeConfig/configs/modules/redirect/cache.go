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

package redirect

import (
	"errors"
	"fmt"

	"github.com/bfenetworks/bfe/bfe_modules/mod_redirect"
	netv1 "k8s.io/api/networking/v1"

	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/annotations"
	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/configs/cache"
	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/util"
)

type redirectRule struct {
	*cache.BaseRule
	statusCode int
	action     *mod_redirect.ActionFileList
}

type redirectRuleCache struct {
	*cache.BaseCache
}

func newRedirectRuleCache() *redirectRuleCache {
	return &redirectRuleCache{
		BaseCache: cache.NewBaseCache(),
	}
}

func (c redirectRuleCache) UpdateByIngress(ingress *netv1.Ingress) error {
	action, statusCode, err := parseRedirectActionFromAnnotations(ingress.Annotations)
	return c.BaseCache.UpdateByIngressFramework(
		ingress,
		func() (bool, error) {
			return action != nil, err
		},
		func(ingress *netv1.Ingress, host, path string) cache.Rule {
			return &redirectRule{
				BaseRule: cache.NewBaseRule(
					util.NamespacedName(ingress.Namespace, ingress.Name),
					host,
					path,
					ingress.Annotations,
					ingress.CreationTimestamp.Time,
				),
				statusCode: statusCode,
				action:     action,
			}
		},
		func() error {
			return nil
		},
	)
}

func parseRedirectActionFromAnnotations(annots map[string]string) (*mod_redirect.ActionFileList, int, error) {
	annotationList := []string{annotations.RedirectURLSetAnnotation, annotations.RedirectURLFromQueryAnnotation, annotations.RedirectURLPrefixAddAnnotation, annotations.RedirectSchemeSetSetAnnotation}

	cnt := 0
	for _, annotation := range annotationList {
		if _, ok := annots[annotation]; ok {
			cnt++
		}
	}

	switch {
	case cnt == 0:
		if annots[annotations.RedirectResponseStatusAnnotation] != "" {
			return nil, 0, fmt.Errorf("unexpected annotation: {%s:%s}", annotations.RedirectResponseStatusAnnotation, annots[annotations.RedirectResponseStatusAnnotation])
		}
		return nil, 0, nil

	case cnt == 1:
		cmd, param, err := annotations.GetRedirectAction(annots)
		if err != nil {
			return nil, 0, err
		}
		actions := &mod_redirect.ActionFileList{mod_redirect.ActionFile{
			Cmd:    &cmd,
			Params: []string{param},
		}}
		if err = mod_redirect.ActionFileListCheck(actions); err != nil {
			return nil, 0, err
		}

		statusCode, err := annotations.GetRedirectStatusCode(annots)
		if err != nil {
			return nil, 0, err
		}
		return actions, statusCode, nil

	default:
		return nil, 0, errors.New("setting multiple redirection-related annotations at the same time is not supported")
	}
}
