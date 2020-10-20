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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	buildv1alpha "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildclient "github.com/shipwright-io/build/pkg/client/build/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
)

// KubeAccess contains Kubernetes cluster access objects in a single place
type KubeAccess struct {
	RestConfig   *rest.Config
	Client       kubernetes.Interface
	BuildClient  buildclient.Interface
	TektonClient tektonclient.Interface
}

// NamingConfig contains all fields required for proper naming of buildRuns
type NamingConfig struct {
	Namespace string
	Prefix    string
}

// BuildConfig contains all fields required to setup a buildRun
type BuildConfig struct {
	ClusterBuildStrategy string
	SourceURL            string
	SourceContextDir     string
	SourceSecretRef      string
	SourceDockerfile     string
	OutputImageURL       string
	OutputSecretRef      string
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

// TestPlan is a plan with steps that define tests
type TestPlan struct {
	Namespace string `yaml:"namespace" json:"namespace"`
	Steps     []struct {
		Name      string                 `yaml:"name" json:"name"`
		BuildSpec buildv1alpha.BuildSpec `yaml:"buildSpec" json:"buildSpec"`
	} `yaml:"steps" json:"steps"`
}

// NewTestPlan creates a test plan based on the provided input
func NewTestPlan(in io.Reader) (*TestPlan, error) {
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	// As long as the BuildSpec does only support JSON tags in its struct, the
	// direct unmarshal will not work, because there is a mix of YAML and JSON
	// tags in the structs. Based on https://git.io/JTRxN, the only option is
	// to first translate the YAML input into pure JSON and then let the JSON
	// package do the unmarshalling entirely.

	var tmp interface{}
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}

	var testplan TestPlan
	if err := json.Unmarshal(jsonBytes, &testplan); err != nil {
		return nil, err
	}

	return &testplan, nil
}

func createNamespaceAndName(namingCfg NamingConfig, buildCfg BuildConfig, idx int) (string, string) {
	return namingCfg.Namespace, fmt.Sprintf("%s-%s-%d", namingCfg.Prefix, buildCfg.ClusterBuildStrategy, idx)
}

func createBuildSpec(name string, buildCfg BuildConfig) (*buildv1alpha.BuildSpec, error) {
	var (
		dockerfile = func() *string {
			if strings.Contains(buildCfg.ClusterBuildStrategy, "kaniko") {
				return &buildCfg.SourceDockerfile
			}

			return nil
		}

		strategyRefKind = func(kind buildv1alpha.BuildStrategyKind) *buildv1alpha.BuildStrategyKind {
			return &kind
		}

		secrefRef = func(name string) *corev1.LocalObjectReference {
			if len(name) > 0 {
				return &corev1.LocalObjectReference{
					Name: name,
				}
			}

			return nil
		}
	)

	outputImageURL, err := getOutputImageURL(name, buildCfg.OutputImageURL)
	if err != nil {
		return nil, err
	}

	return &buildv1alpha.BuildSpec{
		StrategyRef: &buildv1alpha.StrategyRef{
			Name: buildCfg.ClusterBuildStrategy,
			Kind: strategyRefKind(buildv1alpha.ClusterBuildStrategyKind),
		},

		Source: buildv1alpha.GitSource{
			URL:        buildCfg.SourceURL,
			ContextDir: pointer.StringPtr(buildCfg.SourceContextDir),
			SecretRef:  secrefRef(buildCfg.SourceSecretRef),
		},

		Dockerfile: dockerfile(),

		Output: buildv1alpha.Image{
			ImageURL:  outputImageURL,
			SecretRef: secrefRef(buildCfg.OutputSecretRef),
		},
	}, nil
}

func getOutputImageURL(name string, outputImageURL string) (string, error) {
	const invalidImageURLErrMsg = "failed to use output image URL %s, it should look like server.com/org, or server.com/org/image, or server.com/org/image:tag"

	switch len(strings.Split(outputImageURL, "/")) {
	case 2: // image URL only contains server and organization
		outputImageURL = fmt.Sprintf("%s/%s:latest", outputImageURL, name)

	case 3: // image URL has three parts, which should be server, org, and image
		switch len(strings.Split(outputImageURL, ":")) {
		case 1: // tag is missing
			outputImageURL = fmt.Sprintf("%s:latest", outputImageURL)

		case 2: // tag is available

		default:
			return "", fmt.Errorf(invalidImageURLErrMsg, outputImageURL)
		}

	default: // not enough information, or invalid content
		return "", fmt.Errorf(invalidImageURLErrMsg, outputImageURL)
	}

	return outputImageURL, nil
}
