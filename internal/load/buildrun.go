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
	"strings"
	"sync"
	"time"

	buildv1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/text"
	"k8s.io/apimachinery/pkg/api/resource"
)

// CheckSystemAndConfig sanity checks the cluster using the provided buildrun
// settings to verify whether a buildrun can work and how much pressure it
// would put onto the system
func CheckSystemAndConfig(kubeAccess KubeAccess, config BuildRunSettings, parallel int) error {
	// Check whether the configured cluster build strategy is available
	cbsRaw, err := kubeAccess.DynClient.Resource(ClusterBuildStrategy).Get(config.ClusterBuildStrategy, metav1.GetOptions{})
	if err != nil {
		if list, _ := kubeAccess.DynClient.Resource(ClusterBuildStrategy).List(metav1.ListOptions{}); list != nil {
			names := make([]string, len(list.Items))
			for i, entry := range list.Items {
				names[i] = entry.GetName()
			}

			return fmt.Errorf("failed to find ClusterBuildStrategy %s, available strategies are: %s",
				config.ClusterBuildStrategy,
				strings.Join(names, ", "),
			)
		}

		bunt.Printf("DarkOrange{*Warning:*} The current permissions do not allow to check whether build strategy CadetBlue{*%s*} is available.\n\n", config.ClusterBuildStrategy)
	}

	// Given that the permissions allow it, check how many buildruns are
	// currently in the system already
	if buildRunsResults, err := kubeAccess.DynClient.Resource(BuildRunResource).List(metav1.ListOptions{}); err == nil {
		var (
			totalBuildRuns     int
			completedBuildRuns int
		)

		for _, item := range buildRunsResults.Items {
			buildRun, err := asBuildRun(item)
			if err != nil {
				return err
			}

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
			bunt.Printf("PaleGoldenrod{_Please note:_} With currently %s, there might be some interference with the test buildruns. Please take the current system utilisation into consideration when analysing any performance measurements.\n",
				text.Plural(totalBuildRuns-completedBuildRuns, "active buildrun"),
			)

			fmt.Println()
		}
	}

	if nodesResults, err := kubeAccess.Client.CoreV1().Nodes().List(metav1.ListOptions{}); err != nil {
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

		//
		if cbsRaw != nil {
			clusterBuildStrategy, err := asClusterBuildStrategy(*cbsRaw)
			if err != nil {
				return err
			}

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

// ExecuteSingleBuildRun executes a single buildrun based on the given settings
func ExecuteSingleBuildRun(kubeAccess KubeAccess, name string, config BuildRunSettings) (*BuildRunResult, error) {
	_, err := applyBuild(kubeAccess.DynClient, config.Namespace, newBuild(name, config))
	if err != nil {
		return nil, err
	}

	defer deleteResource(kubeAccess, BuildResource, config.Namespace, name)

	_, err = applyBuildRun(kubeAccess.DynClient, config.Namespace, newBuildRun(name, name))
	if err != nil {
		return nil, err
	}

	defer deleteResource(kubeAccess, BuildRunResource, config.Namespace, name)

	buildRun, err := waitForBuildRunCompletion(kubeAccess, config.Namespace, name)
	if err != nil {
		return nil, err
	}

	defer deleteContainerImage(kubeAccess, buildRun.Namespace, config.Output.SecretRef, buildRun.Status.BuildSpec.Output.ImageURL)

	taskRun, err := lookUpTaskRun(kubeAccess.DynClient, config.Namespace, *buildRun)
	if err != nil {
		return nil, err
	}

	pod, err := lookUpPod(kubeAccess.Client, config.Namespace, *taskRun)
	if err != nil {
		return nil, err
	}

	var totalTektonStepsTime time.Duration
	for _, step := range taskRun.Status.Steps {
		totalTektonStepsTime += step.Terminated.FinishedAt.Time.Sub(step.Terminated.StartedAt.Time)
	}

	lastInitPodIdx := len(pod.Status.InitContainerStatuses) - 1
	lastInitPod := pod.Status.InitContainerStatuses[lastInitPodIdx]

	totalBuildRunTime := buildRun.Status.CompletionTime.Time.Sub(buildRun.ObjectMeta.CreationTimestamp.Time)
	buildRunRampUpDuration := taskRun.CreationTimestamp.Time.Sub(buildRun.CreationTimestamp.Time)
	taskRunRampUpDuration := pod.CreationTimestamp.Time.Sub(taskRun.CreationTimestamp.Time)
	podRampUpDuration := lastInitPod.State.Terminated.FinishedAt.Sub(pod.CreationTimestamp.Time)
	internalProcessingTime := totalBuildRunTime - buildRunRampUpDuration - taskRunRampUpDuration - podRampUpDuration - totalTektonStepsTime

	return &BuildRunResult{
		TotalBuildRunTime:      totalBuildRunTime,
		BuildRunRampUpDuration: buildRunRampUpDuration,
		TaskRunRampUpDuration:  taskRunRampUpDuration,
		PodRampUpDuration:      podRampUpDuration,
		InternalProcessingTime: internalProcessingTime,
	}, nil
}

// ExecuteParallelBuildRuns executes the same buildrun multiple times in
// parallel
func ExecuteParallelBuildRuns(kubeAccess KubeAccess, config BuildRunSettings, parallel int) ([]BuildRunResult, error) {
	var errors = make(chan error, parallel)
	var wg sync.WaitGroup
	wg.Add(parallel)

	var buildRunResults = make([]BuildRunResult, parallel)
	for i := 0; i < parallel; i++ {
		go func(idx int) {
			defer wg.Done()

			name := fmt.Sprintf("%s-%s-%s-%d",
				config.Prefix,
				config.Name,
				config.BuildType,
				idx,
			)

			result, err := ExecuteSingleBuildRun(kubeAccess, name, config)
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
func ExecuteSeriesOfParallelBuildRuns(kubeAccess KubeAccess, config BuildRunSettings, start int, end int, increment int) ([]BuildRunResultSet, error) {
	var results = []BuildRunResultSet{}

	for parallelBuilds := start; parallelBuilds <= end; parallelBuilds += increment {
		buildRunResults, err := ExecuteParallelBuildRuns(kubeAccess, config, parallelBuilds)
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
		bunt.Printf("Running test plan step %d/%d: LightSlateGray{%s}, build type *%s* using cluster build strategy _%s_ to build CornflowerBlue{~%s~}\n",
			i+1,
			len(testplan.Steps),
			step.Name,
			step.BuildRunSettings.BuildType,
			step.BuildRunSettings.ClusterBuildStrategy,
			step.BuildRunSettings.Source.URL,
		)

		if _, err := ExecuteSingleBuildRun(kubeAccess, step.Name, step.BuildRunSettings); err != nil {
			return err
		}
	}

	return nil
}

func estimateResourceRequests(clusterBuildStrategy buildv1.ClusterBuildStrategy, concurrent int64) corev1.ResourceList {
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
