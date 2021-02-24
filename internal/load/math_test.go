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
	Context("result set", func() {
		It("should calculate the result set with an odd input list length", func() {
			var results = []Result{}

			var factors = []time.Duration{1, 5, 12}
			for _, f := range factors {
				results = append(results, Result{
					Value{MockLabel1, time.Duration(f * 1 * time.Second)},
					Value{MockLabel2, time.Duration(f * 10 * time.Second)},
					Value{MockLabel3, time.Duration(f * 100 * time.Second)},
					Value{MockLabel4, time.Duration(f * 1000 * time.Second)},
					Value{MockLabel5, time.Duration(f * 10000 * time.Second)},
				})
			}

			Expect(CalculateResultSet(results, "thing")).To(Equal(ResultSet{
				EntityType:      "thing",
				NumberOfResults: len(factors),
				Minimum: Result{
					Value{MockLabel1, time.Duration(1 * time.Second)},
					Value{MockLabel2, time.Duration(10 * time.Second)},
					Value{MockLabel3, time.Duration(100 * time.Second)},
					Value{MockLabel4, time.Duration(1000 * time.Second)},
					Value{MockLabel5, time.Duration(10000 * time.Second)},
				},
				Mean: Result{
					Value{MockLabel1, time.Duration(6 * time.Second)},
					Value{MockLabel2, time.Duration(60 * time.Second)},
					Value{MockLabel3, time.Duration(600 * time.Second)},
					Value{MockLabel4, time.Duration(6000 * time.Second)},
					Value{MockLabel5, time.Duration(60000 * time.Second)},
				},
				Median: Result{
					Value{MockLabel1, time.Duration(5 * time.Second)},
					Value{MockLabel2, time.Duration(50 * time.Second)},
					Value{MockLabel3, time.Duration(500 * time.Second)},
					Value{MockLabel4, time.Duration(5000 * time.Second)},
					Value{MockLabel5, time.Duration(50000 * time.Second)},
				},
				Maximum: Result{
					Value{MockLabel1, time.Duration(12 * time.Second)},
					Value{MockLabel2, time.Duration(120 * time.Second)},
					Value{MockLabel3, time.Duration(1200 * time.Second)},
					Value{MockLabel4, time.Duration(12000 * time.Second)},
					Value{MockLabel5, time.Duration(120000 * time.Second)},
				},
			}))
		})

		It("should calculate the result set with an even input list length", func() {
			var results = []Result{}

			var factors = []time.Duration{1, 2, 4, 9}
			for _, f := range factors {
				results = append(results, Result{
					Value{MockLabel1, time.Duration(f * 1 * time.Second)},
					Value{MockLabel2, time.Duration(f * 10 * time.Second)},
					Value{MockLabel3, time.Duration(f * 100 * time.Second)},
					Value{MockLabel4, time.Duration(f * 1000 * time.Second)},
					Value{MockLabel5, time.Duration(f * 10000 * time.Second)},
				})
			}

			Expect(CalculateResultSet(results, "thing")).To(Equal(ResultSet{
				EntityType:      "thing",
				NumberOfResults: len(factors),
				Minimum: Result{
					Value{MockLabel1, time.Duration(1 * time.Second)},
					Value{MockLabel2, time.Duration(10 * time.Second)},
					Value{MockLabel3, time.Duration(100 * time.Second)},
					Value{MockLabel4, time.Duration(1000 * time.Second)},
					Value{MockLabel5, time.Duration(10000 * time.Second)},
				},
				Mean: Result{
					Value{MockLabel1, time.Duration(4 * time.Second)},
					Value{MockLabel2, time.Duration(40 * time.Second)},
					Value{MockLabel3, time.Duration(400 * time.Second)},
					Value{MockLabel4, time.Duration(4000 * time.Second)},
					Value{MockLabel5, time.Duration(40000 * time.Second)},
				},
				Median: Result{
					Value{MockLabel1, time.Duration(3 * time.Second)},
					Value{MockLabel2, time.Duration(30 * time.Second)},
					Value{MockLabel3, time.Duration(300 * time.Second)},
					Value{MockLabel4, time.Duration(3000 * time.Second)},
					Value{MockLabel5, time.Duration(30000 * time.Second)},
				},
				Maximum: Result{
					Value{MockLabel1, time.Duration(9 * time.Second)},
					Value{MockLabel2, time.Duration(90 * time.Second)},
					Value{MockLabel3, time.Duration(900 * time.Second)},
					Value{MockLabel4, time.Duration(9000 * time.Second)},
					Value{MockLabel5, time.Duration(90000 * time.Second)},
				},
			}))
		})

		It("should create the min, mean, median, and max values independent of the respective buildrun", func() {
			var x = func(a, b, c, d, e uint64) Result {
				return Result{
					Value{MockLabel1, time.Duration(a) * time.Second},
					Value{MockLabel2, time.Duration(b) * time.Second},
					Value{MockLabel3, time.Duration(c) * time.Second},
					Value{MockLabel4, time.Duration(d) * time.Second},
					Value{MockLabel5, time.Duration(e) * time.Second},
				}
			}

			var results = []Result{
				x(1, 12, 1, 12, 1),
				x(5, 5, 5, 5, 5),
				x(12, 1, 12, 1, 12),
			}

			Expect(CalculateResultSet(results, "thing")).To(Equal(ResultSet{
				EntityType:      "thing",
				NumberOfResults: len(results),
				Minimum: Result{
					Value{MockLabel1, time.Duration(1 * time.Second)},
					Value{MockLabel2, time.Duration(1 * time.Second)},
					Value{MockLabel3, time.Duration(1 * time.Second)},
					Value{MockLabel4, time.Duration(1 * time.Second)},
					Value{MockLabel5, time.Duration(1 * time.Second)},
				},
				Mean: Result{
					Value{MockLabel1, time.Duration(6 * time.Second)},
					Value{MockLabel2, time.Duration(6 * time.Second)},
					Value{MockLabel3, time.Duration(6 * time.Second)},
					Value{MockLabel4, time.Duration(6 * time.Second)},
					Value{MockLabel5, time.Duration(6 * time.Second)},
				},
				Median: Result{
					Value{MockLabel1, time.Duration(5 * time.Second)},
					Value{MockLabel2, time.Duration(5 * time.Second)},
					Value{MockLabel3, time.Duration(5 * time.Second)},
					Value{MockLabel4, time.Duration(5 * time.Second)},
					Value{MockLabel5, time.Duration(5 * time.Second)},
				},
				Maximum: Result{
					Value{MockLabel1, time.Duration(12 * time.Second)},
					Value{MockLabel2, time.Duration(12 * time.Second)},
					Value{MockLabel3, time.Duration(12 * time.Second)},
					Value{MockLabel4, time.Duration(12 * time.Second)},
					Value{MockLabel5, time.Duration(12 * time.Second)},
				},
			}))
		})
	})
})
