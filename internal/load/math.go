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
	"sort"
	"time"
)

// CalculateBuildRunResultSet creates a buildrun result set using a list of
// buildrun results to get the minimum, mean, median, and maximum results
func CalculateBuildRunResultSet(results []BuildRunResult) BuildRunResultSet {
	// sort results based on the total buildrun time
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalBuildRunTime < results[j].TotalBuildRunTime
	})

	var (
		min *BuildRunResult
		max *BuildRunResult
	)

	for i := range results {
		buildRunResult := results[i]

		if min == nil || min.TotalBuildRunTime > buildRunResult.TotalBuildRunTime {
			min = &buildRunResult
		}

		if max == nil || max.TotalBuildRunTime < buildRunResult.TotalBuildRunTime {
			max = &buildRunResult
		}
	}

	return BuildRunResultSet{
		NumberOfResults: len(results),
		Minimum:         *min,
		Maximum:         *max,
		Mean:            averageBuildRunResult(results),
		Median:          medianBuildRunResult(results),
	}
}

func averageBuildRunResult(results []BuildRunResult) BuildRunResult {
	var length = len(results)
	if length == 0 {
		panic("no results available")
	}

	var (
		sumTotalBuildRunTime      time.Duration
		sumBuildRunRampUpDuration time.Duration
		sumTaskRunRampUpDuration  time.Duration
		sumPodRampUpDuration      time.Duration
		sumInternalProcessingTime time.Duration
	)

	for i := range results {
		sumTotalBuildRunTime += results[i].TotalBuildRunTime
		sumBuildRunRampUpDuration += results[i].BuildRunRampUpDuration
		sumTaskRunRampUpDuration += results[i].TaskRunRampUpDuration
		sumPodRampUpDuration += results[i].PodRampUpDuration
		sumInternalProcessingTime += results[i].InternalProcessingTime
	}

	return BuildRunResult{
		TotalBuildRunTime:      sumTotalBuildRunTime / time.Duration(length),
		BuildRunRampUpDuration: sumBuildRunRampUpDuration / time.Duration(length),
		TaskRunRampUpDuration:  sumTaskRunRampUpDuration / time.Duration(length),
		PodRampUpDuration:      sumPodRampUpDuration / time.Duration(length),
		InternalProcessingTime: sumInternalProcessingTime / time.Duration(length),
	}
}

func medianBuildRunResult(results []BuildRunResult) BuildRunResult {
	var length = len(results)
	if length == 0 {
		panic("no results available")
	}

	switch length % 2 {
	case 0:
		return averageBuildRunResult(results[length/2-1 : length/2])

	default:
		return results[(length-1)/2]
	}
}
