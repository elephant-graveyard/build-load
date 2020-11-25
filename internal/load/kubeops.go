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

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/wrap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/ptr"
)

var (
	defaultBuildRunWaitTimeout = time.Duration(5 * time.Minute)
)

func newBuild(namespace string, name string, buildSpec buildv1alpha1.BuildSpec) buildv1alpha1.Build {
	return buildv1alpha1.Build{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Build",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},

		Spec: buildSpec,
	}
}

func newBuildRun(name string, build buildv1alpha1.Build, generateServiceAccount bool) buildv1alpha1.BuildRun {
	return buildv1alpha1.BuildRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BuildRun",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: build.Namespace,
		},

		Spec: buildv1alpha1.BuildRunSpec{
			BuildRef: &buildv1alpha1.BuildRef{
				Name: build.Name,
			},

			ServiceAccount: &buildv1alpha1.ServiceAccount{
				Generate: generateServiceAccount,
			},
		},
	}
}

func applyBuild(kubeAccess KubeAccess, build buildv1alpha1.Build) (*buildv1alpha1.Build, error) {
	if err := deleteBuild(kubeAccess, build.Namespace, build.Name, &metav1.DeleteOptions{GracePeriodSeconds: ptr.Int64(0)}); err != nil {
		return nil, err
	}

	debug("Create build %s", build.Name)
	return kubeAccess.BuildClient.
		BuildV1alpha1().
		Builds(build.Namespace).
		Create(&build)
}

func applyBuildRun(kubeAccess KubeAccess, buildRun buildv1alpha1.BuildRun) (*buildv1alpha1.BuildRun, error) {
	if err := deleteBuildRun(kubeAccess, buildRun.Namespace, buildRun.Name, &metav1.DeleteOptions{GracePeriodSeconds: ptr.Int64(0)}); err != nil {
		return nil, err
	}

	debug("Create buildrun %s", buildRun.Name)
	return kubeAccess.BuildClient.
		BuildV1alpha1().
		BuildRuns(buildRun.Namespace).
		Create(&buildRun)
}

func deleteBuild(kubeAccess KubeAccess, namespace string, name string, deleteOptions *metav1.DeleteOptions) error {
	_, err := kubeAccess.BuildClient.BuildV1alpha1().Builds(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	debug("Delete build %s", name)
	if err := kubeAccess.BuildClient.BuildV1alpha1().Builds(namespace).Delete(name, deleteOptions); err != nil {
		return err
	}

	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = kubeAccess.BuildClient.BuildV1alpha1().Builds(namespace).Get(name, metav1.GetOptions{})
		return errors.IsNotFound(err), nil
	})
}

func deleteBuildRun(kubeAccess KubeAccess, namespace string, name string, deleteOptions *metav1.DeleteOptions) error {
	buildRun, err := kubeAccess.BuildClient.BuildV1alpha1().BuildRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	_, pod := lookUpTaskRunAndPod(kubeAccess, *buildRun)

	debug("Delete buildrun %s", name)
	if err := kubeAccess.BuildClient.BuildV1alpha1().BuildRuns(namespace).Delete(name, deleteOptions); err != nil {
		return err
	}

	err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = kubeAccess.BuildClient.BuildV1alpha1().BuildRuns(namespace).Get(name, metav1.GetOptions{})
		return errors.IsNotFound(err), nil
	})

	if err != nil {
		return err
	}

	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = kubeAccess.Client.CoreV1().Pods(namespace).Get(pod.Name, metav1.GetOptions{})
		return errors.IsNotFound(err), nil
	})
}

func waitForBuildRunCompletion(kubeAccess KubeAccess, buildRun *buildv1alpha1.BuildRun) (*buildv1alpha1.BuildRun, error) {
	var (
		timeout   = defaultBuildRunWaitTimeout
		interval  = 5 * time.Second
		namespace = buildRun.Namespace
		name      = buildRun.Name
	)

	if buildRun.Spec.Timeout != nil {
		timeout = buildRun.Spec.Timeout.Duration
	}

	debug("Polling every %v to wait for completion of buildrun %s", interval, buildRun.Name)
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
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

func lookUpTaskRunAndPod(kubeAccess KubeAccess, buildRun buildv1alpha1.BuildRun) (taskRun *pipelinev1alpha1.TaskRun, taskRunPod *corev1.Pod) {
	if buildRun.Status.LatestTaskRunRef != nil {
		tmp, err := kubeAccess.TektonClient.
			TektonV1alpha1().
			TaskRuns(buildRun.Namespace).
			Get(*buildRun.Status.LatestTaskRunRef, metav1.GetOptions{})

		if err == nil {
			taskRun = tmp
		}
	}

	// In case the taskRun could be looked up, use it to also get the
	// respective taskRun pod
	if taskRun != nil {
		tmp, err := kubeAccess.Client.
			CoreV1().
			Pods(taskRun.Namespace).
			Get(taskRun.Status.PodName, metav1.GetOptions{})

		if err == nil {
			taskRunPod = tmp
		}

		return taskRun, taskRunPod
	}

	// For scenarios where the taskRun cannot be obtains, look up the
	// pod by a well known label that should lead to the pod
	listResp, err := kubeAccess.Client.
		CoreV1().
		Pods(buildRun.Namespace).
		List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("buildrun.build.dev/name=%s", buildRun.Name)},
		)

	if err == nil && len(listResp.Items) == 1 {
		taskRunPod = &listResp.Items[0]
	}

	return taskRun, taskRunPod
}

func lookUpDockerCredentialsFromSecret(kubeAccess KubeAccess, namespace string, secretRef *corev1.LocalObjectReference) (string, string, error) {
	secret, err := kubeAccess.Client.CoreV1().Secrets(namespace).Get(secretRef.Name, metav1.GetOptions{})
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

func buildRunError(kubeAccess KubeAccess, buildRun buildv1alpha1.BuildRun) error {
	if buildRun.Status.Succeeded == corev1.ConditionTrue {
		return nil
	}

	if taskRun, taskRunPod := lookUpTaskRunAndPod(kubeAccess, buildRun); taskRun != nil && taskRunPod != nil {
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

	// default error with not much more details other than the status reason
	return wrap.Errorf(
		fmt.Errorf(buildRun.Status.Reason),
		"buildRun %s failed",
		buildRun.Name,
	)
}
