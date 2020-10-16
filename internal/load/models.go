/*
Copyright © 2020 The Homeport Team

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

package load

import (
	"time"

	buildclient "github.com/shipwright-io/build/pkg/client/build/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeAccess contains Kubernetes cluster access objects in a single place
type KubeAccess struct {
	RestConfig   *rest.Config
	Client       kubernetes.Interface
	BuildClient  buildclient.Interface
	TektonClient tektonclient.Interface
}

// BuildRunResultSet is an aggregated result set based on multiple
// buildrun results
type BuildRunResultSet struct {
	NumberOfResults int

	Minimum BuildRunResult
	Maximum BuildRunResult
	Mean    BuildRunResult
	Median  BuildRunResult
}

// BuildRunResult contains the raw time results of a buildrun
type BuildRunResult struct {
	TotalBuildRunTime      time.Duration
	BuildRunRampUpDuration time.Duration
	TaskRunRampUpDuration  time.Duration
	PodRampUpDuration      time.Duration
	InternalProcessingTime time.Duration
}

// BuildRunSettings contains all required settings for a buildrun
type BuildRunSettings struct {
	// type settings
	ClusterBuildStrategy string `yaml:"clusterBuildStrategy"`
	BuildType            string `yaml:"buildType"`

	// location and test resource naming settings
	Namespace string `yaml:"namespace"`
	Prefix    string `yaml:"prefix"`
	Name      string `yaml:"name"`

	// build source settings
	Source struct {
		SecretRef  string `yaml:"secretRef"`
		URL        string `yaml:"url"`
		ContextDir string `yaml:"contextDir"`
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"source"`

	// build output settings
	Output struct {
		SecretRef         string `yaml:"secretRef"`
		RegistryHostname  string `yaml:"registryHostname"`
		RegistryNamespace string `yaml:"registryNamespace"`
	} `yaml:"output"`
}

// TestPlan is a plan with steps that define tests
type TestPlan struct {
	Steps []struct {
		Name             string           `yaml:"name"`
		BuildRunSettings BuildRunSettings `yaml:"buildRunConfig"`
	} `yaml:"steps"`
}
