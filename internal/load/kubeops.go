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
	"time"

	buildv1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// TODO Make timeouts configurable
var (
	defaultBuildRunWaitTimeout = time.Duration(600) * time.Second
	defaultDeleteWaitTimeout   = time.Duration(10) * time.Second
)

func newBuild(name string, config BuildRunSettings) buildv1.Build {
	build := buildv1.Build{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Build",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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

func newBuildRun(name string, buildRef string) buildv1.BuildRun {
	return buildv1.BuildRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BuildRun",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},

		Spec: buildv1.BuildRunSpec{
			BuildRef: &buildv1.BuildRef{
				Name: buildRef,
			},

			ServiceAccount: &buildv1.ServiceAccount{
				Generate: false,
			},
		},
	}
}

func applyBuild(dynClient dynamic.Interface, namespace string, build buildv1.Build) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&build)
	if err != nil {
		return nil, err
	}

	return applyUnstructured(dynClient, namespace, BuildResource, obj)
}

func applyBuildRun(dynClient dynamic.Interface, namespace string, buildRun buildv1.BuildRun) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&buildRun)
	if err != nil {
		return nil, err
	}

	return applyUnstructured(dynClient, namespace, BuildRunResource, obj)
}

func applyUnstructured(dynClient dynamic.Interface, namespace string, resource schema.GroupVersionResource, obj map[string]interface{}) (*unstructured.Unstructured, error) {
	return dynClient.
		Resource(resource).
		Namespace(namespace).
		Create(&unstructured.Unstructured{Object: obj}, metav1.CreateOptions{})
}

func waitForBuildRunCompletion(kubeAccess KubeAccess, namespace string, name string) (*buildv1.BuildRun, error) {
	watcher, err := kubeAccess.DynClient.Resource(BuildRunResource).Namespace(namespace).Watch(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	defer watcher.Stop()

	timeout := time.After(defaultBuildRunWaitTimeout)

	for {
		select {
		case event := <-watcher.ResultChan():
			switch event.Type {
			case watch.Modified:
				obj := event.Object.(*unstructured.Unstructured)

				if obj.GetName() == name {
					buildRun, err := asBuildRun(*obj)
					if err != nil {
						return nil, err
					}

					switch buildRun.Status.Succeeded {
					case corev1.ConditionTrue:
						if buildRun.Status.CompletionTime != nil {
							return buildRun, nil
						}

					case corev1.ConditionFalse:
						return nil, fmt.Errorf(buildRun.Status.Reason)
					}
				}
			}

		case <-timeout:
			return nil, fmt.Errorf("timeout occurred while waiting for buildrun %s to complete", name)
		}
	}
}

func lookUpTaskRun(dynClient dynamic.Interface, namespace string, buildRun buildv1.BuildRun) (*pipelinev1.TaskRun, error) {
	if buildRun.Status.LatestTaskRunRef == nil {
		return nil, fmt.Errorf("failed to find taskrun of buildrun %s, because taskrun reference is not set", buildRun.Name)
	}

	obj, err := dynClient.Resource(TaskRunResource).Namespace(namespace).Get(*buildRun.Status.LatestTaskRunRef, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return asTaskRun(*obj)
}

func lookUpPod(client kubernetes.Interface, namespace string, taskRun pipelinev1.TaskRun) (*corev1.Pod, error) {
	return client.
		CoreV1().
		Pods(namespace).
		Get(taskRun.Status.PodName, metav1.GetOptions{})
}

func asBuild(obj unstructured.Unstructured) (*buildv1.Build, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var build buildv1.Build
	if err := json.Unmarshal(data, &build); err != nil {
		return nil, err
	}

	return &build, nil
}

func asBuildRun(obj unstructured.Unstructured) (*buildv1.BuildRun, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var buildRun buildv1.BuildRun
	if err := json.Unmarshal(data, &buildRun); err != nil {
		return nil, err
	}

	return &buildRun, nil
}

func asTaskRun(obj unstructured.Unstructured) (*pipelinev1.TaskRun, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var taskRun pipelinev1.TaskRun
	if err := json.Unmarshal(data, &taskRun); err != nil {
		return nil, err
	}

	return &taskRun, nil
}

func asClusterBuildStrategy(obj unstructured.Unstructured) (*buildv1.ClusterBuildStrategy, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}

	var clusterBuildStrategy buildv1.ClusterBuildStrategy
	if err := json.Unmarshal(data, &clusterBuildStrategy); err != nil {
		return nil, err
	}

	return &clusterBuildStrategy, nil
}

func deleteResource(kubeAccess KubeAccess, resource schema.GroupVersionResource, namespace string, name string) error {
	watcher, err := kubeAccess.DynClient.Resource(resource).Namespace(namespace).Watch(metav1.ListOptions{})
	if err != nil {
		return err
	}

	defer watcher.Stop()
	timeout := time.After(defaultDeleteWaitTimeout)

	if err := kubeAccess.DynClient.Resource(resource).Namespace(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	for {
		select {
		case event := <-watcher.ResultChan():
			switch event.Type {
			case watch.Deleted:
				obj := event.Object.(*unstructured.Unstructured)

				if obj.GetName() == name {
					return nil
				}
			}

		case <-timeout:
			return fmt.Errorf("timeout occurred while waiting for buildrun to complete")
		}
	}
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
