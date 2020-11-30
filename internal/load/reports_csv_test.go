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
	"bytes"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/homeport/build-load/internal/load"
)

var _ = Describe("create CSV reports", func() {
	var mockBuildRunResults = func(n int) []BuildRunResult {
		var result = make([]BuildRunResult, n)
		for i := 0; i < n; i++ {
			result[i] = BuildRunResult{
				Value{MockLabel1, time.Duration(i+1) * 1 * time.Second},
				Value{MockLabel2, time.Duration(i+1) * 10 * time.Second},
				Value{MockLabel3, time.Duration(i+1) * 100 * time.Second},
				Value{MockLabel4, time.Duration(i+1) * 1000 * time.Second},
				Value{MockLabel5, time.Duration(i+1) * 10000 * time.Second},
			}
		}

		return result
	}

	Context("having a buildrun result set", func() {
		It("should create a CSV file based on the content in the buildrun result set", func() {
			var buildRunResultSets = []BuildRunResultSet{
				CalculateBuildRunResultSet(mockBuildRunResults(5)),
				CalculateBuildRunResultSet(mockBuildRunResults(10)),
				CalculateBuildRunResultSet(mockBuildRunResults(15)),
				CalculateBuildRunResultSet(mockBuildRunResults(20)),
				CalculateBuildRunResultSet(mockBuildRunResults(25)),
			}

			var buf bytes.Buffer
			err := CreateBuildRunResultSetCSV(buildRunResultSets, &buf)
			Expect(err).ToNot(HaveOccurred())

			Expect(buf.String()).To(Equal(`number of results, mock #1, mock #2, mock #3, mock #4 , mock #5
5                , 3000   , 30000  , 300000 , 3000000 , 30000000
10               , 5500   , 55000  , 550000 , 5500000 , 55000000
15               , 8000   , 80000  , 800000 , 8000000 , 80000000
20               , 10500  , 105000 , 1050000, 10500000, 105000000
25               , 13000  , 130000 , 1300000, 13000000, 130000000
`))
		})
	})
})
