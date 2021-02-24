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

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/text"
)

func (rs ResultSet) String() string {
	bold := func(args ...string) []string {
		var tmp = make([]string, len(args))
		for i, str := range args {
			tmp[i] = bunt.Sprintf("*%s*", str)
		}

		return tmp
	}

	headline := bold("Description", "Minimum", "Mean", "Median", "Maximum")
	tableData := [][]string{headline}

	for _, entry := range rs.Minimum {
		line := make([]string, len(headline))
		line[0] = entry.Description

		tableData = append(tableData, line)
	}

	for i, x := range []Result{rs.Minimum, rs.Mean, rs.Median, rs.Maximum} {
		for j, value := range x {
			tableData[j+1][i+1] = value.Value.String()
		}
	}

	table, err := neat.Table(tableData, neat.AlignCenter(1, 2, 3, 4), neat.CustomSeparator(bunt.Sprintf(" DimGray{│} ")))
	if err != nil {
		panic(err)
	}

	return neat.ContentBox(
		bunt.Sprintf("Results based on %s", text.Plural(rs.NumberOfResults, fmt.Sprintf("parallel %s", rs.EntityType))),
		table,
		neat.HeadlineColor(bunt.Beige),
		neat.NoLineWrap(),
	)
}
