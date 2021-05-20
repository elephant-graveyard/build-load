/*
Copyright © 2021 The Homeport Team

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
	"fmt"
	"sync"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func buildError(build buildv1alpha1.Build) error {
	if build.Status.Registered == corev1.ConditionTrue {
		return nil
	}

	return fmt.Errorf("build failed to register. Reason=%s. Message=%s", build.Status.Reason, build.Status.Message)
}

func registerSingleBuild(kubeAccess KubeAccess, namespace string, name string, buildSpec buildv1alpha1.BuildSpec, buildAnnotations map[string]string, options ...BuildRunOption) (*Result, error) {
	var buildRunOptions = buildRunOptions{}
	for _, option := range options {
		option(&buildRunOptions)
	}

	build, err := applyBuild(kubeAccess, newBuild(namespace, name, buildSpec, buildAnnotations))
	if err != nil {
		return nil, err
	}

	if !buildRunOptions.skipDelete {
		defer func() {
			if err := deleteBuild(kubeAccess, build.Namespace, build.Name, defaultDeleteOptions); err != nil {
				warn("failed to delete build %s, %v\n", name, err)
			}
		}()
	}

	build, err = waitForBuildRegistered(kubeAccess, build)
	if err != nil {
		return nil, err
	}

	var buildRegisteredTime time.Time

	for _, mf := range build.ManagedFields {
		if mf.Operation == metav1.ManagedFieldsOperationUpdate && mf.Manager == "shipwright-build-controller" {
			buildRegisteredTime = mf.Time.Time
		}
	}

	if buildRegisteredTime.IsZero() {
		return nil, fmt.Errorf("did not find update time for build %s", build.Name)
	}

	var buildResult = Result{
		Value{
			BuildRegistrationTime,
			duration(build.CreationTimestamp.Time, buildRegisteredTime),
		},
	}

	debug("build _%s/%s_ results: %v",
		namespace,
		name,
		buildResult,
	)

	return &buildResult, nil
}

func waitForBuildRegistered(kubeAccess KubeAccess, build *buildv1alpha1.Build) (*buildv1alpha1.Build, error) {
	var (
		timeout   = defaultBuildRunWaitTimeout
		interval  = 5 * time.Second
		namespace = build.Namespace
		name      = build.Name
	)

	debug("Polling every %v to wait for registration of build %s", interval, build.Name)
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		build, err = kubeAccess.BuildClient.BuildV1alpha1().Builds(namespace).Get(kubeAccess.Context, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch build.Status.Registered {
		case corev1.ConditionTrue:
			return true, nil

		case corev1.ConditionFalse:
			return false, buildError(*build)
		}

		return false, nil
	})

	return build, err
}

// ExecuteBuilds creates a number of builds and waits for them to be registered
func ExecuteBuilds(kubeAccess KubeAccess, namingCfg NamingConfig, buildCfg BuildConfig, count int) ([]Result, error) {

	var errors = make(chan error, count)
	var wg sync.WaitGroup
	wg.Add(count)

	var buildResults = make([]Result, count)
	for i := 0; i < count; i++ {
		go func(idx int) {
			defer wg.Done()

			namespace, name := createNamespaceAndName(namingCfg, buildCfg, idx)

			buildSpec, err := createBuildSpec(name, buildCfg)
			if err != nil {
				errors <- err
				return
			}

			buildAnnotations := createBuildAnnotations(buildCfg)

			result, err := registerSingleBuild(
				kubeAccess,
				namespace,
				name,
				*buildSpec,
				buildAnnotations,
				GenerateServiceAccount(buildCfg.GenerateServiceAccount),
				SkipDelete(buildCfg.SkipDelete),
			)

			if err != nil {
				errors <- err
				return
			}

			buildResults[idx] = *result
		}(i)
	}

	wg.Wait()
	close(errors)

	return buildResults, wrapErrorChanResults(errors, "failed to execute buildruns")
}
