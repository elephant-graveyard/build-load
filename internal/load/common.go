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
	"os"
	"path/filepath"

	buildv1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	"github.com/gonvenience/wrap"
	"github.com/mitchellh/go-homedir"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ClusterBuildStrategy GroupVersionResource for dynamic client
	ClusterBuildStrategy = schema.GroupVersionResource{
		Group:    buildv1.SchemeGroupVersion.Group,
		Version:  buildv1.SchemeGroupVersion.Version,
		Resource: "clusterbuildstrategies",
	}

	// BuildResource GroupVersionResource for dynamic client
	BuildResource = schema.GroupVersionResource{
		Group:    buildv1.SchemeGroupVersion.Group,
		Version:  buildv1.SchemeGroupVersion.Version,
		Resource: "builds",
	}

	// BuildRunResource GroupVersionResource for dynamic client
	BuildRunResource = schema.GroupVersionResource{
		Group:    buildv1.SchemeGroupVersion.Group,
		Version:  buildv1.SchemeGroupVersion.Version,
		Resource: "buildruns",
	}

	// TaskRunResource GroupVersionResource for dynamic client
	TaskRunResource = schema.GroupVersionResource{
		Group:    pipelinev1.SchemeGroupVersion.Group,
		Version:  pipelinev1.SchemeGroupVersion.Version,
		Resource: "taskruns",
	}
)

func lookUpKubeConfigFilePath() (string, error) {
	if value, present := os.LookupEnv("KUBECONFIG"); present {
		return value, nil
	}

	homedir, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	defaultLocation := filepath.Join(homedir, ".kube", "config")
	if _, err := os.Stat(defaultLocation); err != nil {
		return "", err
	}

	return defaultLocation, nil
}

// NewKubeAccess creates a new kubernetes access handle
func NewKubeAccess() (*KubeAccess, error) {
	kubeConfigFilePath, err := lookUpKubeConfigFilePath()
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFilePath)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &KubeAccess{
		RestConfig: restConfig,
		Client:     client,
		DynClient:  dynClient,
	}, nil
}

func cbsptr(o buildv1.BuildStrategyKind) *buildv1.BuildStrategyKind {
	return &o
}

func wrapErrorChanResults(errors chan error, format string, a ...interface{}) error {
	errorList := []error{}
	for err := range errors {
		if err != nil {
			errorList = append(errorList, err)
		}
	}

	switch len(errorList) {
	case 0:
		return nil

	case 1:
		return errorList[0]

	default:
		return wrap.Errorsf(errorList, format, a...)
	}
}
