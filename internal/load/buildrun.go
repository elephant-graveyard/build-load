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
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/text"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

type buildRunOptions struct {
	generateServiceAccount bool
	skipDelete             bool
}

// BuildRunOption specifies optional settings for a buildrun
type BuildRunOption func(*buildRunOptions)

// CheckSystemAndConfig sanity checks the cluster using the provided buildrun
// settings to verify whether a buildrun can work and how much pressure it
// would put onto the system
func CheckSystemAndConfig(kubeAccess KubeAccess, buildCfg BuildConfig, parallel int) error {
	// Check whether the configured cluster build strategy is available
	clusterBuildStrategy, err := kubeAccess.BuildClient.BuildV1alpha1().ClusterBuildStrategies().Get(buildCfg.ClusterBuildStrategy, metav1.GetOptions{})
	if err != nil {
		clusterBuildStrategy = nil

		switch terr := err.(type) {
		case *errors.StatusError:
			switch terr.ErrStatus.Code {
			case http.StatusNotFound:
				if list, _ := kubeAccess.BuildClient.BuildV1alpha1().ClusterBuildStrategies().List(metav1.ListOptions{}); list != nil {
					var names = make([]string, len(list.Items))
					for i, entry := range list.Items {
						names[i] = entry.GetName()
					}

					if len(names) > 0 {
						return bunt.Errorf("failed to find ClusterBuildStrategy _%s_, available strategies are: %s",
							buildCfg.ClusterBuildStrategy,
							strings.Join(names, "\n"),
						)
					}

					return bunt.Errorf("failed to find ClusterBuildStrategy _%s_", buildCfg.ClusterBuildStrategy)
				}

			case http.StatusForbidden:
				warn("The current permissions do not allow to check whether build strategy CadetBlue{*%s*} is available.\n", buildCfg.ClusterBuildStrategy)
			}
		}
	}

	// Given that the permissions allow it, check how many buildruns are
	// currently in the system already
	if buildRunsResults, err := kubeAccess.BuildClient.BuildV1alpha1().BuildRuns("").List(metav1.ListOptions{}); err == nil {
		var (
			totalBuildRuns     int
			completedBuildRuns int
		)

		for _, buildRun := range buildRunsResults.Items {
			if buildRun.Status.CompletionTime != nil {
				completedBuildRuns++
			}

			totalBuildRuns++
		}

		if totalBuildRuns > 0 {
			bunt.Printf("There are currently LightSkyBlue{%s} in the system. It might be an idea to go through the list of completed buildruns to remove old and obsolete buildruns.\n",
				text.Plural(totalBuildRuns, "completed buildrun"),
			)

			fmt.Println()
		}

		if totalBuildRuns-completedBuildRuns > 0 {
			bunt.Printf("PaleGoldenrod{_Please note:_} With currently %s, there might be some interference with the test buildruns. Please take the current system utilization into consideration when analysing any performance measurements.\n",
				text.Plural(totalBuildRuns-completedBuildRuns, "active buildrun"),
			)

			fmt.Println()
		}
	}

	if nodesResults, err := kubeAccess.Client.CoreV1().Nodes().List(metav1.ListOptions{}); err == nil {
		var totalCPU int64
		var totalMemory int64
		for _, node := range nodesResults.Items {
			totalCPU += node.Status.Capacity.Cpu().MilliValue()
			totalMemory += node.Status.Capacity.Memory().Value()
		}

		totalNodeResources := corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(totalCPU, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(totalMemory, resource.BinarySI),
		}

		if clusterBuildStrategy != nil {
			resourcesForClusterBuildStrategy := estimateResourceRequests(*clusterBuildStrategy, int64(parallel))

			scaleToString := func(q *resource.Quantity) string {
				var mods = []string{"Byte", "KiB", "MiB", "GiB", "TiB"}

				tmp := float64(q.Value())

				var i = 0
				for i = 0; tmp > 1023.9 && i < len(mods); i++ {
					tmp /= 1024.0
				}

				return fmt.Sprintf("%.1f %s", tmp, mods[i])
			}

			bunt.Printf("Keep in mind, with Moccasin{_%s_}, the estimated resource request will be roughly SlateGray{%v CPU cores} and LightSlateGray{%v system memory}. Available in the cluster are SlateGray{%v CPU cores} and LightSlateGray{%v system memory}.\n\n",
				text.Plural(parallel, "concurrent buildrun"),
				resourcesForClusterBuildStrategy.Cpu(),
				scaleToString(resourcesForClusterBuildStrategy.Memory()),
				totalNodeResources.Cpu(),
				scaleToString(totalNodeResources.Memory()),
			)
		}
	}

	return nil
}

// GenerateServiceAccount sets whether or not a service account is created for the buildrun
func GenerateServiceAccount(value bool) BuildRunOption {
	return func(o *buildRunOptions) {
		o.generateServiceAccount = value
	}
}

// SkipDelete sets whether or not the resources like build, buildrun and output image should be cleaned up
func SkipDelete(value bool) BuildRunOption {
	return func(o *buildRunOptions) {
		o.skipDelete = value
	}
}

// ExecuteSingleBuildRun executes a single buildrun based on the given settings
func ExecuteSingleBuildRun(kubeAccess KubeAccess, namespace string, name string, buildSpec buildv1alpha1.BuildSpec, options ...BuildRunOption) (*BuildRunResult, error) {
	var buildRunOptions = buildRunOptions{}
	for _, option := range options {
		option(&buildRunOptions)
	}

	build, err := applyBuild(kubeAccess, newBuild(namespace, name, buildSpec))
	if err != nil {
		return nil, err
	}

	var graceperiod int64 = int64(0)
	deleteOptions := metav1.DeleteOptions{GracePeriodSeconds: &graceperiod}

	if !buildRunOptions.skipDelete {
		defer func() {
			if err := deleteBuild(kubeAccess, build.Namespace, build.Name, &deleteOptions); err != nil {
				warn("failed to delete build %s, %v\n", name, err)
			}
		}()
	}

	buildRun, err := applyBuildRun(kubeAccess, newBuildRun(name, *build, buildRunOptions.generateServiceAccount))
	if err != nil {
		return nil, err
	}

	if !buildRunOptions.skipDelete {
		defer func() {
			if err := deleteBuildRun(kubeAccess, buildRun.Namespace, buildRun.Name, &deleteOptions); err != nil {
				warn("failed to delete buildrun %s, %v\n", name, err)
			}
		}()
	}

	buildRun, err = waitForBuildRunCompletion(kubeAccess, buildRun)
	if err != nil {
		return nil, err
	}

	if !buildRunOptions.skipDelete {
		defer func() {
			debug("Delete container image %s", buildRun.Status.BuildSpec.Output.ImageURL)
			if err := deleteContainerImage(kubeAccess, buildRun.Namespace, build.Spec.Output.SecretRef, buildRun.Status.BuildSpec.Output.ImageURL); err != nil {
				warn("failed to delete image %s, %v\n", buildRun.Status.BuildSpec.Output.ImageURL, err)
			}
		}()
	}

	var buildRunResult = BuildRunResult{
		Value{
			BuildrunCompletionTime,
			duration(buildRun.CreationTimestamp.Time, buildRun.Status.CompletionTime.Time),
		},
	}

	taskRun, pod := lookUpTaskRunAndPod(kubeAccess, *buildRun)
	if pod != nil {
		if taskRun != nil {
			buildRunResult = append(buildRunResult,
				Value{
					BuildrunControlTime,
					duration(buildRun.Status.StartTime.Time, taskRun.Status.StartTime.Time),
				},
			)

			buildRunResult = append(buildRunResult,
				Value{
					TaskrunCompletionTime,
					duration(taskRun.Status.StartTime.Time, taskRun.Status.CompletionTime.Time),
				},
			)

			buildRunResult = append(buildRunResult,
				Value{
					TaskrunControlTime,
					duration(taskRun.Status.StartTime.Time, pod.Status.StartTime.Time),
				},
			)
		}

		var lastContainerIdx = len(pod.Status.ContainerStatuses) - 1
		buildRunResult = append(buildRunResult,
			Value{
				PodCompletionTime,
				duration(pod.Status.StartTime.Time, pod.Status.ContainerStatuses[lastContainerIdx].State.Terminated.FinishedAt.Time),
			},
		)

		buildRunResult = append(buildRunResult,
			Value{
				PodControlTime,
				duration(buildRun.Status.StartTime.Time, pod.Status.StartTime.Time),
			},
		)
	}

	debug("buildrun _%s/%s_ results: %v",
		namespace,
		name,
		buildRunResult,
	)

	return &buildRunResult, nil
}

// ExecuteParallelBuildRuns executes the same buildrun multiple times in
// parallel
func ExecuteParallelBuildRuns(kubeAccess KubeAccess, namingCfg NamingConfig, buildCfg BuildConfig, parallel int) ([]BuildRunResult, error) {
	var errors = make(chan error, parallel)
	var wg sync.WaitGroup
	wg.Add(parallel)

	var buildRunResults = make([]BuildRunResult, parallel)
	for i := 0; i < parallel; i++ {
		go func(idx int) {
			defer wg.Done()

			namespace, name := createNamespaceAndName(namingCfg, buildCfg, idx)

			buildSpec, err := createBuildSpec(name, buildCfg)
			if err != nil {
				errors <- err
				return
			}

			result, err := ExecuteSingleBuildRun(
				kubeAccess,
				namespace,
				name,
				*buildSpec,
				GenerateServiceAccount(buildCfg.GenerateServiceAccount),
				SkipDelete(buildCfg.SkipDelete),
			)

			if err != nil {
				errors <- err
				return
			}

			buildRunResults[idx] = *result
		}(i)
	}

	wg.Wait()
	close(errors)

	return buildRunResults, wrapErrorChanResults(errors, "failed to execute buildruns")
}

// ExecuteSeriesOfParallelBuildRuns executes a series of parallel buildruns
// increasing the number of parallel buildruns with each interation
func ExecuteSeriesOfParallelBuildRuns(kubeAccess KubeAccess, namingCfg NamingConfig, buildCfg BuildConfig, start int, end int, increment int) ([]BuildRunResultSet, error) {
	var results = []BuildRunResultSet{}

	for parallelBuilds := start; parallelBuilds < end; parallelBuilds += increment {
		buildRunResults, err := ExecuteParallelBuildRuns(kubeAccess, namingCfg, buildCfg, parallelBuilds)
		if err != nil {
			return nil, err
		}

		buildRunResultSet := CalculateBuildRunResultSet(buildRunResults)

		// TODO Make it configure whether this should be printed or not
		fmt.Println(buildRunResultSet)

		results = append(results, buildRunResultSet)
	}

	return results, nil
}

// ExecuteTestPlan executes the given test plan step by step
func ExecuteTestPlan(kubeAccess KubeAccess, testplan TestPlan) error {
	for i, step := range testplan.Steps {
		bunt.Printf("Running test plan step %d/%d: LightSlateGray{%s}, using cluster build strategy _%s_ to build CornflowerBlue{~%s~}\n",
			i+1,
			len(testplan.Steps),
			step.Name,
			step.BuildSpec.StrategyRef.Name,
			step.BuildSpec.Source.URL,
		)

		name := fmt.Sprintf("test-plan-step-%s", step.Name)

		outputImageURL, err := getOutputImageURL(name, step.BuildSpec.Output.ImageURL)
		if err != nil {
			return err
		}

		step.BuildSpec.Output.ImageURL = outputImageURL

		if _, err := ExecuteSingleBuildRun(kubeAccess, testplan.Namespace, name, step.BuildSpec, GenerateServiceAccount(testplan.GenerateServiceAccount)); err != nil {
			return err
		}
	}

	return nil
}

func estimateResourceRequests(clusterBuildStrategy buildv1alpha1.ClusterBuildStrategy, concurrent int64) corev1.ResourceList {
	var (
		maxCPU *resource.Quantity
		maxMem *resource.Quantity
	)

	// TODO Verify that this approach by searching for the biggest resource
	// values is actually what happens with tekton in a real use case.
	for i := range clusterBuildStrategy.Spec.BuildSteps {
		step := clusterBuildStrategy.Spec.BuildSteps[i]

		if maxCPU == nil || step.Resources.Requests.Cpu().AsDec().Cmp(maxCPU.AsDec()) > 0 {
			maxCPU = step.Resources.Requests.Cpu()
		}

		if maxMem == nil || step.Resources.Requests.Memory().AsDec().Cmp(maxMem.AsDec()) > 0 {
			maxMem = step.Resources.Requests.Memory()
		}
	}

	return corev1.ResourceList{
		corev1.ResourceCPU: *resource.NewMilliQuantity(
			maxCPU.MilliValue()*concurrent,
			resource.DecimalSI,
		),
		corev1.ResourceMemory: *resource.NewQuantity(
			maxMem.Value()*concurrent,
			resource.BinarySI,
		),
	}
}

func duration(start, end time.Time) time.Duration {
	if start.After(end) {
		warn("start time %v is after end time %v, return 0 as the duration", start, end)
		return time.Duration(0)
	}

	return end.Sub(start)
}
