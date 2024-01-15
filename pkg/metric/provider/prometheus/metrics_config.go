/*
 Copyright 2023 The Kapacity Authors.

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

package prometheus

import (
	"fmt"
	"io"
	"os"

	prommodel "github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
	promadaptercfg "sigs.k8s.io/prometheus-adapter/pkg/config"
)

// MetricsDiscoveryConfig is an extension of promadaptercfg.MetricsDiscoveryConfig
// which includes extra configurations for advanced metrics queries used by Kapacity.
type MetricsDiscoveryConfig struct {
	ResourceRules           *ResourceRules                 `json:"resourceRules,omitempty" yaml:"resourceRules,omitempty"`
	Rules                   []promadaptercfg.DiscoveryRule `json:"rules,omitempty" yaml:"rules,omitempty"`
	ExternalRules           []promadaptercfg.DiscoveryRule `json:"externalRules,omitempty" yaml:"externalRules,omitempty"`
	WorkloadPodNamePatterns []WorkloadPodNamePattern       `json:"workloadPodNamePatterns,omitempty" yaml:"workloadPodNamePatterns,omitempty"`
}

// ResourceRules is an extension of promadaptercfg.ResourceRules
// which includes extra configurations for advanced metrics queries used by Kapacity.
type ResourceRules struct {
	CPU    ResourceRule       `json:"cpu" yaml:"cpu"`
	Memory ResourceRule       `json:"memory" yaml:"memory"`
	Window prommodel.Duration `json:"window" yaml:"window"`
}

// ResourceRule is an extension of promadaptercfg.ResourceRule
// which includes extra configurations for advanced metrics queries used by Kapacity.
type ResourceRule struct {
	promadaptercfg.ResourceRule `json:",inline" yaml:",inline"`
	// ReadyPodsOnlyContainerQuery is the query used to fetch the metrics for containers of ready Pods only.
	ReadyPodsOnlyContainerQuery string `json:"readyPodsOnlyContainerQuery" yaml:"readyPodsOnlyContainerQuery"`
}

// WorkloadPodNamePattern describes the pod name pattern of a specific kind of workload.
type WorkloadPodNamePattern struct {
	// GroupKind is the group-kind of the workload.
	GroupKind `json:",inline" yaml:",inline"`
	// Pattern is a regex expression which matches all the pods belonging to a specific workload.
	// The workload's name placeholder should be "%s" and would be replaced by the name
	// of a specific workload during runtime.
	Pattern string `json:"pattern" yaml:"pattern"`
}

// GroupKind represents a Kubernetes group-kind.
type GroupKind struct {
	Group string `json:"group,omitempty" yaml:"group,omitempty"`
	Kind  string `json:"kind" yaml:"kind"`
}

// MetricsConfigFromFile loads MetricsDiscoveryConfig from a particular file.
func MetricsConfigFromFile(filename string) (*MetricsDiscoveryConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to load metrics discovery config file: %v", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to load metrics discovery config file: %v", err)
	}
	return MetricsConfigFromYAML(contents)
}

// MetricsConfigFromYAML loads MetricsDiscoveryConfig from a blob of YAML.
func MetricsConfigFromYAML(contents []byte) (*MetricsDiscoveryConfig, error) {
	var cfg MetricsDiscoveryConfig
	if err := yaml.UnmarshalStrict(contents, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse metrics discovery config: %v", err)
	}
	return &cfg, nil
}
