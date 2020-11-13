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

package load_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/homeport/build-load/internal/load"
)

var _ = Describe("math functions", func() {
	Context("buildRun result set", func() {
		It("should calculate the result set", func() {
			var results = []BuildRunResult{}

			var factors = []time.Duration{1, 5, 12}
			for _, f := range factors {
				results = append(results, BuildRunResult{
					TotalBuildRunTime:      time.Duration(f * 1 * time.Second),
					BuildRunRampUpDuration: time.Duration(f * 10 * time.Second),
					TaskRunRampUpDuration:  time.Duration(f * 100 * time.Second),
					PodRampUpDuration:      time.Duration(f * 1000 * time.Second),
					InternalProcessingTime: time.Duration(f * 10000 * time.Second),
				})
			}

			Expect(CalculateBuildRunResultSet(results)).To(Equal(BuildRunResultSet{
				NumberOfResults: len(factors),
				Minimum: BuildRunResult{
					TotalBuildRunTime:      time.Duration(1 * time.Second),
					BuildRunRampUpDuration: time.Duration(10 * time.Second),
					TaskRunRampUpDuration:  time.Duration(100 * time.Second),
					PodRampUpDuration:      time.Duration(1000 * time.Second),
					InternalProcessingTime: time.Duration(10000 * time.Second),
				},
				Mean: BuildRunResult{
					TotalBuildRunTime:      time.Duration(6 * time.Second),
					BuildRunRampUpDuration: time.Duration(60 * time.Second),
					TaskRunRampUpDuration:  time.Duration(600 * time.Second),
					PodRampUpDuration:      time.Duration(6000 * time.Second),
					InternalProcessingTime: time.Duration(60000 * time.Second),
				},
				Median: BuildRunResult{
					TotalBuildRunTime:      time.Duration(5 * time.Second),
					BuildRunRampUpDuration: time.Duration(50 * time.Second),
					TaskRunRampUpDuration:  time.Duration(500 * time.Second),
					PodRampUpDuration:      time.Duration(5000 * time.Second),
					InternalProcessingTime: time.Duration(50000 * time.Second),
				},
				Maximum: BuildRunResult{
					TotalBuildRunTime:      time.Duration(12 * time.Second),
					BuildRunRampUpDuration: time.Duration(120 * time.Second),
					TaskRunRampUpDuration:  time.Duration(1200 * time.Second),
					PodRampUpDuration:      time.Duration(12000 * time.Second),
					InternalProcessingTime: time.Duration(120000 * time.Second),
				},
			}))
		})
	})
})
