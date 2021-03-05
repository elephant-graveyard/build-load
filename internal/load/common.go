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

	buildclient "github.com/shipwright-io/build/pkg/client/build/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/wrap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Debug enables or disables additional output
var Debug bool

func lookUpKubeConfigFilePath() (string, error) {
	if value, present := os.LookupEnv("KUBECONFIG"); present {
		return value, nil
	}

	homedir, err := os.UserHomeDir()
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

	// the default QPS is 5 which is not very much to make load on the system
	restConfig.QPS = 500
	restConfig.Burst = 500

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	buildClient, err := buildclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	tektonClient, err := tektonclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &KubeAccess{
		RestConfig:   restConfig,
		Client:       client,
		BuildClient:  buildClient,
		TektonClient: tektonClient,
	}, nil
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

func warn(format string, a ...interface{}) {
	bunt.Printf("DarkOrange{*Warning:*} %s\n", bunt.Sprintf(format, a...))
}

func debug(format string, a ...interface{}) {
	if Debug {
		bunt.Printf("DimGray{*Debug:*} %s\n", bunt.Sprintf(format, a...))
	}
}
