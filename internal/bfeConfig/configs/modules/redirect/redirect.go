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
	"net/url"

	"github.com/bfenetworks/bfe/bfe_modules/mod_redirect"
	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/annotations"
	netv1 "k8s.io/api/networking/v1"

	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/configs"
	"github.com/bfenetworks/ingress-bfe/internal/bfeConfig/util"
)

const (
	ConfigNameRedirect = "mod_redirect"
	RedirectRuleData   = "mod_redirect/redirect.data"
)

type ModRedirectConfig struct {
	version           string
	redirectRuleCache *redirectRuleCache
	redirectConfFile  *mod_redirect.RedirectConfFile
}

func NewRedirectConfig(version string) *ModRedirectConfig {
	return &ModRedirectConfig{
		version:           version,
		redirectRuleCache: newRedirectRuleCache(),
		redirectConfFile:  newRedirectConfFile(version),
	}
}

func newRedirectConfFile(version string) *mod_redirect.RedirectConfFile {
	ruleFileList := make([]mod_redirect.RedirectRuleFile, 0)
	productRulesFile := make(mod_redirect.ProductRulesFile)
	productRulesFile[configs.DefaultProduct] = (*mod_redirect.RuleFileList)(&ruleFileList)
	return &mod_redirect.RedirectConfFile{
		Version: &version,
		Config:  &productRulesFile,
	}
}

func (r *ModRedirectConfig) UpdateIngress(ingress *netv1.Ingress) error {
	if len(ingress.Spec.Rules) == 0 {
		return nil
	}

	ingressName := util.NamespacedName(ingress.Namespace, ingress.Name)
	if r.redirectRuleCache.ContainsIngress(ingressName) {
		r.redirectRuleCache.DeleteByIngress(ingressName)
	}

	// update cache
	if err := r.redirectRuleCache.UpdateByIngress(ingress); err != nil {
		return err
	}

	if err := r.updateRedirectConfFile(); err != nil {
		r.redirectRuleCache.DeleteByIngress(ingressName)
		return err
	}
	return nil
}

func (r *ModRedirectConfig) updateRedirectConfFile() error {
	ruleList := r.redirectRuleCache.GetRules()
	redirectRuleList := make(mod_redirect.RuleFileList, 0, len(ruleList))
	for _, rule := range ruleList {
		rule := rule.(*redirectRule)
		cond, err := rule.GetCond()
		if err != nil {
			return err
		}
		redirectRuleList = append(redirectRuleList, mod_redirect.RedirectRuleFile{
			Cond:    &cond,
			Actions: rule.action,
			Status:  &(rule.statusCode),
		})
	}

	redirectConfFile := newRedirectConfFile(util.NewVersion())
	(*redirectConfFile.Config)[configs.DefaultProduct] = &redirectRuleList
	if err := mod_redirect.RedirectConfCheck(*redirectConfFile); err != nil {
		return err
	}

	r.redirectConfFile = redirectConfFile
	return nil
}

func (r *ModRedirectConfig) DeleteIngress(namespace, name string) {
	ingressName := util.NamespacedName(namespace, name)

	if !r.redirectRuleCache.ContainsIngress(ingressName) {
		return
	}

	r.redirectRuleCache.DeleteByIngress(ingressName)
	_ = r.updateRedirectConfFile()
}

func (r *ModRedirectConfig) Reload() error {
	reload := false
	if *r.redirectConfFile.Version != r.version {
		if err := r.updateRedirectConfFile(); err != nil {
			return fmt.Errorf("dump %s error: %v", RedirectRuleData, err)
		}
		err := util.DumpBfeConf(RedirectRuleData, r.redirectConfFile)
		if err != nil {
			return fmt.Errorf("dump %s error: %v", RedirectRuleData, err)
		}
		reload = true
	}
	if reload {
		if err := util.ReloadBfe(ConfigNameRedirect); err != nil {
			return err
		}
		r.version = *r.redirectConfFile.Version
	}

	return nil
}

func (r ModRedirectConfig) Name() string {
	return "mod_redirect"
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
		if err := checkAction(cmd, param); err != nil {
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

func checkAction(cmd, param string) error {
	switch cmd {
	case "URL_SET":
		if _, err := url.ParseRequestURI(param); err != nil {
			return fmt.Errorf("the value of %s shoud be a valid URL string: %w", annotations.RedirectURLSetAnnotation, err)
		}
		return nil

	case "URL_FROM_QUERY":
		return nil

	case "URL_PREFIX_ADD":
		if _, err := url.ParseRequestURI(param); err != nil {
			return fmt.Errorf("the value of %s shoud be a valid URL string without fragment: %w", annotations.RedirectURLPrefixAddAnnotation, err)
		} else if parsedURL, err := url.Parse(param); err != nil || parsedURL.Fragment != "" {
			return fmt.Errorf("the value of %s shoud be a valid RL string without fragement, but found %s", annotations.RedirectURLPrefixAddAnnotation, param)
		}
		return nil

	case "SCHEME_SET":
		if param != "https" && param != "http" {
			return fmt.Errorf("scheme %s invalid, only http|https supported now", param)
		}
		return nil

	default:
		return fmt.Errorf("unsupported cmd for redirection action: %s", cmd)
	}
}
