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
	"github.com/bfenetworks/bfe/bfe_modules/mod_redirect"
	netv1 "k8s.io/api/networking/v1"

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
		func(ingress *netv1.Ingress, host, path string, _ netv1.HTTPIngressPath) cache.Rule {
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
