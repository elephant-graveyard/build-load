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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/homeport/build-load/internal/load"
)

var _ = Describe("create CSV reports", func() {
	var mockResults = func(n int) []Result {
		var result = make([]Result, n)
		for i := 0; i < n; i++ {
			result[i] = Result{
				Value{MockLabel1, time.Duration(i+1) * 1 * time.Second},
				Value{MockLabel2, time.Duration(i+1) * 10 * time.Second},
				Value{MockLabel3, time.Duration(i+1) * 100 * time.Second},
				Value{MockLabel4, time.Duration(i+1) * 1000 * time.Second},
				Value{MockLabel5, time.Duration(i+1) * 10000 * time.Second},
			}
		}

		return result
	}

	Context("having a list of results", func() {
		It("should create a CSV file based on the content in the results", func() {
			var buildrunResults = mockResults(5)

			var buf bytes.Buffer
			err := CreateResultsCSV(buildrunResults, &buf)
			Expect(err).ToNot(HaveOccurred())

			Expect(buf.String()).To(Equal(`buildrun, mock #1, mock #2, mock #3, mock #4, mock #5
1       , 1000   , 10000  , 100000 , 1000000, 10000000
2       , 2000   , 20000  , 200000 , 2000000, 20000000
3       , 3000   , 30000  , 300000 , 3000000, 30000000
4       , 4000   , 40000  , 400000 , 4000000, 40000000
5       , 5000   , 50000  , 500000 , 5000000, 50000000
`))
		})
	})

	Context("having a result set", func() {
		It("should create a CSV file based on the content in the result set", func() {
			var buildRunResultSets = []ResultSet{
				CalculateResultSet(mockResults(5), "thing"),
				CalculateResultSet(mockResults(10), "thing"),
				CalculateResultSet(mockResults(15), "thing"),
				CalculateResultSet(mockResults(20), "thing"),
				CalculateResultSet(mockResults(25), "thing"),
			}

			var buf bytes.Buffer
			err := CreateResultSetCSV(buildRunResultSets, &buf)
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
