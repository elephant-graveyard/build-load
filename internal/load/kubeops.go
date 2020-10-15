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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonvenience/wrap"
	buildv1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
)

var (
	defaultBuildRunWaitTimeout = time.Duration(5 * time.Minute)
)

func newBuild(name string, config BuildRunSettings) buildv1.Build {
	build := buildv1.Build{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Build",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.Namespace,
		},

		Spec: buildv1.BuildSpec{
			Source: buildv1.GitSource{
				URL:        config.Source.URL,
				ContextDir: &config.Source.ContextDir,
			},

			StrategyRef: &buildv1.StrategyRef{
				Name: config.ClusterBuildStrategy,
				Kind: cbsptr(buildv1.ClusterBuildStrategyKind),
			},

			Output: buildv1.Image{
				ImageURL: fmt.Sprintf("%s/%s/%s",
					config.Output.RegistryHostname,
					config.Output.RegistryNamespace,
					name,
				),
			},
		},
	}

	// Optional: source registry access credentials
	if len(config.Source.SecretRef) > 0 {
		build.Spec.Source.SecretRef = &corev1.LocalObjectReference{
			Name: config.Source.SecretRef,
		}
	}

	// Optional: target/output registry access credentials
	if len(config.Output.SecretRef) > 0 {
		build.Spec.Output.SecretRef = &corev1.LocalObjectReference{
			Name: config.Output.SecretRef,
		}
	}

	// build type specific spec updates
	switch config.BuildType {
	case "kaniko":
		build.Spec.Dockerfile = &config.Source.Dockerfile

	case "buildpacks":
		// nothing additional to set
	}

	return build
}

func newBuildRun(name string, build buildv1.Build) buildv1.BuildRun {
	return buildv1.BuildRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BuildRun",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: build.Namespace,
		},

		Spec: buildv1.BuildRunSpec{
			BuildRef: &buildv1.BuildRef{
				Name: build.Name,
			},

			ServiceAccount: &buildv1.ServiceAccount{
				Generate: false,
			},
		},
	}
}

func applyBuild(kubeAccess KubeAccess, build buildv1.Build) (*buildv1.Build, error) {
	return kubeAccess.BuildClient.
		BuildV1alpha1().
		Builds(build.Namespace).
		Create(&build)
}

func applyBuildRun(kubeAccess KubeAccess, buildRun buildv1.BuildRun) (*buildv1.BuildRun, error) {
	return kubeAccess.BuildClient.
		BuildV1alpha1().
		BuildRuns(buildRun.Namespace).
		Create(&buildRun)
}

func waitForBuildRunCompletion(kubeAccess KubeAccess, buildRun *buildv1.BuildRun) (*buildv1.BuildRun, error) {
	var (
		timeout   = defaultBuildRunWaitTimeout
		namespace = buildRun.Namespace
		name      = buildRun.Name
	)

	err := wait.PollImmediate(5*time.Second, timeout, func() (done bool, err error) {
		buildRun, err = kubeAccess.BuildClient.BuildV1alpha1().BuildRuns(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch buildRun.Status.Succeeded {
		case corev1.ConditionTrue:
			if buildRun.Status.CompletionTime != nil {
				return true, nil
			}

		case corev1.ConditionFalse:
			return false, buildRunError(kubeAccess, *buildRun)
		}

		return false, nil
	})

	return buildRun, err
}

func lookUpTaskRun(kubeAccess KubeAccess, buildRun buildv1.BuildRun) (*pipelinev1.TaskRun, error) {
	if buildRun.Status.LatestTaskRunRef == nil {
		return nil, fmt.Errorf("failed to find taskrun of buildrun %s, because taskrun reference is not set", buildRun.Name)
	}

	return kubeAccess.TektonClient.
		TektonV1alpha1().
		TaskRuns(buildRun.Namespace).
		Get(*buildRun.Status.LatestTaskRunRef, metav1.GetOptions{})
}

func lookUpPod(client kubernetes.Interface, namespace string, taskRun pipelinev1.TaskRun) (*corev1.Pod, error) {
	return client.
		CoreV1().
		Pods(namespace).
		Get(taskRun.Status.PodName, metav1.GetOptions{})
}

func lookUpDockerCredentialsFromSecret(kubeAccess KubeAccess, namespace string, secretRef string) (string, string, error) {
	secret, err := kubeAccess.Client.CoreV1().Secrets(namespace).Get(secretRef, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	jsonData, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		return "", "", fmt.Errorf("failed to find docker configuration in secret %s", secret.Name)
	}

	var dockerconfig struct {
		Auths map[string]struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Auth     string `json:"auth"`
		} `json:"auths"`
	}

	if err := json.Unmarshal(jsonData, &dockerconfig); err != nil {
		return "", "", err
	}

	for _, entry := range dockerconfig.Auths {
		return entry.Username, entry.Password, nil
	}

	return "", "", fmt.Errorf("failed to find authentication credentials in secret data")
}

func buildRunError(kubeAccess KubeAccess, buildRun buildv1.BuildRun) error {
	if buildRun.Status.Succeeded == corev1.ConditionTrue {
		return nil
	}

	if buildRun.Status.LatestTaskRunRef != nil {
		if taskRun, err := lookUpTaskRun(kubeAccess, buildRun); err == nil {
			if taskRunPod, err := lookUpPod(kubeAccess.Client, buildRun.Namespace, *taskRun); err == nil {
				var buf bytes.Buffer

				for _, container := range append(taskRunPod.Spec.InitContainers, taskRunPod.Spec.Containers...) {
					reader, err := kubeAccess.Client.
						CoreV1().
						RESTClient().
						Get().
						Namespace(taskRunPod.Namespace).
						Name(taskRunPod.Name).
						Resource("pods").
						SubResource("log").
						Param("container", container.Name).
						Stream()

					if err == nil {
						defer reader.Close()

						var scanner = bufio.NewScanner(reader)
						for scanner.Scan() {
							for _, line := range strings.Split(scanner.Text(), "\n") {
								fmt.Fprintf(&buf, "[%s] %s\n", container.Name, line)
							}
						}
					}
				}

				return wrap.Errorf(
					fmt.Errorf(buf.String()),
					"buildRun %s failed",
					buildRun.Name,
				)
			}
		}
	}

	// default error with not much more details other than the status reason
	return wrap.Errorf(
		fmt.Errorf(buildRun.Status.Reason),
		"buildRun %s failed",
		buildRun.Name,
	)
}
