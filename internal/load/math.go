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
	"math"
	"sort"
	"time"
)

// CalculateResultSet creates a result set using a list of
// results to get the minimum, mean, median, and maximum results
func CalculateResultSet(results []Result, entityType string) ResultSet {
	return ResultSet{
		EntityType:      entityType,
		NumberOfResults: len(results),
		Minimum:         min(results),
		Maximum:         max(results),
		Mean:            mean(results),
		Median:          median(results),
	}
}

func emptyResult(reference Result, init time.Duration) (result Result) {
	for _, value := range reference {
		result = append(result, Value{Description: value.Description, Value: init})
	}

	return
}

func min(results []Result) Result {
	tmp := emptyResult(results[0], time.Duration(math.MaxInt64))
	for _, buildRunResult := range results {
		for i, value := range buildRunResult {
			if value.Value < tmp[i].Value {
				tmp[i].Value = value.Value
			}
		}
	}

	return tmp
}

func max(results []Result) Result {
	tmp := emptyResult(results[0], time.Duration(math.MinInt64))
	for _, buildRunResult := range results {
		for i, value := range buildRunResult {
			if value.Value > tmp[i].Value {
				tmp[i].Value = value.Value
			}
		}
	}

	return tmp
}

func mean(results []Result) Result {
	tmp := emptyResult(results[0], time.Duration(0))
	for _, buildRunResult := range results {
		for i, value := range buildRunResult {
			tmp[i].Value += value.Value
		}
	}

	for i := range tmp {
		tmp[i].Value /= time.Duration(len(results))
	}

	return tmp
}

func median(results []Result) Result {
	type values struct {
		values []time.Duration
	}

	listOfValues := make([]values, len(results[0]))
	for _, buildRunResult := range results {
		for i, value := range buildRunResult {
			listOfValues[i].values = append(listOfValues[i].values, value.Value)
		}
	}

	for idx := range listOfValues {
		sort.Slice(listOfValues[idx].values, func(i, j int) bool {
			return listOfValues[idx].values[i] < listOfValues[idx].values[j]
		})
	}

	length := len(results)
	tmp := emptyResult(results[0], time.Duration(0))
	for i := range tmp {
		switch length % 2 {
		case 0:
			l, r := listOfValues[i].values[length/2-1], listOfValues[i].values[length/2]
			tmp[i].Value = (l + r) / time.Duration(2)

		default:
			tmp[i].Value = listOfValues[i].values[(length-1)/2]
		}
	}

	return tmp
}
