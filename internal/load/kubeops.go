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
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/wrap"
	"github.com/lucasb-eyer/go-colorful"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/ptr"
)

var (
	defaultBuildRunWaitTimeout = time.Duration(5 * time.Minute)
	defaultDeleteOptions       = &metav1.DeleteOptions{
		GracePeriodSeconds: ptr.Int64(0),
	}
)

func newBuild(namespace string, name string, buildSpec buildv1alpha1.BuildSpec, annotations map[string]string) buildv1alpha1.Build {
	return buildv1alpha1.Build{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Build",
			APIVersion: "build.dev/v1alpha1",
		},

		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Name:        name,
			Namespace:   namespace,
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
	if err := deleteBuild(kubeAccess, build.Namespace, build.Name, defaultDeleteOptions); err != nil {
		return nil, err
	}

	debug("Create build %s", build.Name)
	return kubeAccess.BuildClient.
		ShipwrightV1alpha1().
		Builds(build.Namespace).
		Create(kubeAccess.Context, &build, metav1.CreateOptions{})
}

func applyBuildRun(kubeAccess KubeAccess, buildRun buildv1alpha1.BuildRun) (*buildv1alpha1.BuildRun, error) {
	if err := deleteBuildRun(kubeAccess, buildRun.Namespace, buildRun.Name, defaultDeleteOptions); err != nil {
		return nil, err
	}

	debug("Create buildrun %s", buildRun.Name)
	return kubeAccess.BuildClient.
		ShipwrightV1alpha1().
		BuildRuns(buildRun.Namespace).
		Create(kubeAccess.Context, &buildRun, metav1.CreateOptions{})
}

func deleteBuild(kubeAccess KubeAccess, namespace string, name string, deleteOptions *metav1.DeleteOptions) error {
	_, err := kubeAccess.BuildClient.ShipwrightV1alpha1().Builds(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	debug("Delete build %s", name)
	if err := kubeAccess.BuildClient.ShipwrightV1alpha1().Builds(namespace).Delete(kubeAccess.Context, name, *deleteOptions); err != nil {
		return fmt.Errorf("failed to delete build %s: %w", name, err)
	}

	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = kubeAccess.BuildClient.ShipwrightV1alpha1().Builds(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
		return errors.IsNotFound(err), nil
	})
}

func deleteBuildRun(kubeAccess KubeAccess, namespace string, name string, deleteOptions *metav1.DeleteOptions) error {
	buildRun, err := kubeAccess.BuildClient.ShipwrightV1alpha1().BuildRuns(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil
	}

	_, pod := lookUpTaskRunAndPod(kubeAccess, *buildRun)

	debug("Delete buildrun %s", name)
	if err := kubeAccess.BuildClient.ShipwrightV1alpha1().BuildRuns(namespace).Delete(kubeAccess.Context, name, *deleteOptions); err != nil {
		return fmt.Errorf("failed to delete buildrun %s: %w", name, err)
	}

	err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = kubeAccess.BuildClient.ShipwrightV1alpha1().BuildRuns(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
		return errors.IsNotFound(err), nil
	})

	if err != nil {
		return err
	}

	if pod != nil {
		err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
			_, err = kubeAccess.Client.CoreV1().Pods(namespace).Get(kubeAccess.Context, pod.Name, metav1.GetOptions{})
			return errors.IsNotFound(err), nil
		})
	}

	return err
}

func lookUpTimeout(kubeAccess KubeAccess, buildRun *buildv1alpha1.BuildRun) time.Duration {
	if buildRun.Spec.Timeout != nil {
		debug("Using BuildRun specified timeout of %v", buildRun.Spec.Timeout.Duration)
		return buildRun.Spec.Timeout.Duration
	}

	if buildRun.Spec.BuildRef != nil {
		build, err := kubeAccess.BuildClient.ShipwrightV1alpha1().Builds(buildRun.Namespace).Get(kubeAccess.Context, buildRun.Spec.BuildRef.Name, metav1.GetOptions{})
		if err == nil {
			if build.Spec.Timeout != nil {
				debug("Using Build specified timeout of %v", build.Spec.Timeout.Duration)
				return build.Spec.Timeout.Duration
			}
		}
	}

	debug("Using default fallback timeout of %v", defaultBuildRunWaitTimeout)
	return defaultBuildRunWaitTimeout
}

func waitForBuildRunCompletion(kubeAccess KubeAccess, buildRun *buildv1alpha1.BuildRun) (*buildv1alpha1.BuildRun, error) {
	var (
		timeout   = lookUpTimeout(kubeAccess, buildRun)
		interval  = 5 * time.Second
		namespace = buildRun.Namespace
		name      = buildRun.Name
	)

	var conditionFunc = func() (done bool, err error) {
		buildRun, err = kubeAccess.BuildClient.ShipwrightV1alpha1().BuildRuns(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		var condition = buildRun.Status.GetCondition(buildv1alpha1.Succeeded)
		if condition == nil {
			return false, nil
		}

		switch condition.Status {
		case corev1.ConditionTrue:
			if buildRun.Status.CompletionTime != nil {
				return true, nil
			}

		case corev1.ConditionFalse:
			return false, fmt.Errorf(condition.Message)
		}

		return false, nil
	}

	debug("Polling every %v to wait for completion of buildrun %s within %v", interval, buildRun.Name, timeout)
	if err := wait.PollImmediate(interval, timeout, conditionFunc); err != nil {
		return buildRun, fmt.Errorf("%s\n\n%w", err.Error(), buildRunError(kubeAccess, *buildRun))
	}

	return buildRun, nil
}

func lookUpTaskRunAndPod(kubeAccess KubeAccess, buildRun buildv1alpha1.BuildRun) (taskRun *pipelinev1alpha1.TaskRun, taskRunPod *corev1.Pod) {
	if buildRun.Status.LatestTaskRunRef != nil {
		tmp, err := kubeAccess.TektonClient.
			TektonV1alpha1().
			TaskRuns(buildRun.Namespace).
			Get(context.TODO(), *buildRun.Status.LatestTaskRunRef, metav1.GetOptions{})

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
			Get(kubeAccess.Context, taskRun.Status.PodName, metav1.GetOptions{})

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
		List(kubeAccess.Context, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("buildrun.build.dev/name=%s", buildRun.Name)},
		)

	if err == nil && len(listResp.Items) == 1 {
		taskRunPod = &listResp.Items[0]
	}

	return taskRun, taskRunPod
}

func lookUpDockerCredentialsFromSecret(kubeAccess KubeAccess, namespace string, secretRef *corev1.LocalObjectReference) (string, string, error) {
	secret, err := kubeAccess.Client.CoreV1().Secrets(namespace).Get(kubeAccess.Context, secretRef.Name, metav1.GetOptions{})
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
	var condition = buildRun.Status.GetCondition(buildv1alpha1.Succeeded)

	if condition == nil {
		return nil
	}

	if condition.Status == corev1.ConditionTrue {
		return nil
	}

	if _, taskRunPod := lookUpTaskRunAndPod(kubeAccess, buildRun); taskRunPod != nil {
		var colorise = func(s string) string {
			var h = fnv.New32()
			h.Write([]byte(s))
			tmp := h.Sum32()

			var color = colorful.Color{
				R: float64((tmp >> 16) & 0xFF),
				G: float64((tmp >> 8) & 0xFF),
				B: float64((tmp >> 0) & 0xFF),
			}

			color = color.BlendHsv(bunt.DimGray, 0.42)

			return bunt.Style(
				fmt.Sprintf("[%s]", s),
				bunt.Foreground(color),
				bunt.Bold(),
			)
		}

		var containerLogs = func() string {
			var buf bytes.Buffer

			for _, container := range append(taskRunPod.Spec.InitContainers, taskRunPod.Spec.Containers...) {
				containerName := colorise(container.Name)

				reader, err := kubeAccess.Client.
					CoreV1().
					RESTClient().
					Get().
					Namespace(taskRunPod.Namespace).
					Name(taskRunPod.Name).
					Resource("pods").
					SubResource("log").
					Param("container", container.Name).
					Stream(kubeAccess.Context)

				if err == nil {
					defer reader.Close()

					var scanner = bufio.NewScanner(reader)
					for scanner.Scan() {
						fmt.Fprintf(&buf, "%s %s\n", containerName, scanner.Text())
					}
				}
			}

			return buf.String()
		}

		var buf bytes.Buffer

		status, _ := neat.ToYAMLString(buildRun.Status)
		bunt.Fprintf(&buf, "*BuildRun Status*\n%s\n\n", status)

		if logOutput := containerLogs(); logOutput != "" {
			bunt.Fprintf(&buf, "*Pod container logs*\n%s\n\n", logOutput)
		}

		return fmt.Errorf(buf.String())
	}

	// default error with not much more details other than the status reason
	return wrap.Errorf(
		fmt.Errorf(condition.Reason),
		"buildRun %s failed",
		buildRun.Name,
	)
}
