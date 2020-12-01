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
	"io"
	"strconv"

	"github.com/gonvenience/neat"
)

// CreateBuildRunResultsCSV creates a comma separated values (CSV) content based on the buildruns
func CreateBuildRunResultsCSV(data []BuildRunResult, w io.Writer) error {
	var table = [][]string{}
	for i, buildRunResult := range data {
		// add header based on first entry
		if i == 0 {
			var row = []string{"buildrun"}
			for _, value := range buildRunResult {
				row = append(row, value.Description)
			}

			table = append(table, row)
		}

		var row = []string{strconv.Itoa(i + 1)}
		for _, value := range buildRunResult {
			row = append(row, strconv.Itoa(int(value.Value.Milliseconds())))
		}

		table = append(table, row)
	}

	out, err := neat.Table(table, neat.CustomSeparator(", "))
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, out)
	return err
}

// CreateBuildRunResultSetCSV creates a comma separated values (CSV) content based on the buildrun result sets
func CreateBuildRunResultSetCSV(data []BuildRunResultSet, w io.Writer) error {
	var table = [][]string{}
	for i, buildRunResultSet := range data {
		// add header based on first set entry
		if i == 0 {
			var row = []string{"number of results"}
			for _, value := range buildRunResultSet.Median {
				row = append(row, value.Description)
			}

			table = append(table, row)
		}

		var row = []string{strconv.Itoa(buildRunResultSet.NumberOfResults)}
		for _, value := range buildRunResultSet.Median {
			row = append(row, strconv.Itoa(int(value.Value.Milliseconds())))
		}

		table = append(table, row)
	}

	out, err := neat.Table(table, neat.CustomSeparator(", "))
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, out)
	return err
}
