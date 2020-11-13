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
	"html/template"
	"os"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/text"
)

func (brs BuildRunResultSet) String() string {
	bold := func(args ...string) []string {
		var tmp = make([]string, len(args))
		for i, str := range args {
			tmp[i] = bunt.Sprintf("*%s*", str)
		}

		return tmp
	}

	headline := bold("Description", "Minimum", "Mean", "Median", "Maximum")
	tableData := [][]string{headline}

	for _, entry := range brs.Minimum {
		line := make([]string, len(headline))
		line[0] = entry.Description

		tableData = append(tableData, line)
	}

	for i, x := range []BuildRunResult{brs.Minimum, brs.Mean, brs.Median, brs.Maximum} {
		for j, value := range x {
			tableData[j+1][i+1] = value.Value.String()
		}
	}

	table, err := neat.Table(tableData, neat.VertialBarSeparator())
	if err != nil {
		panic(err)
	}

	return neat.ContentBox(
		bunt.Sprintf("Results based on %s", text.Plural(brs.NumberOfResults, "parallel buildrun")),
		table,
		neat.HeadlineColor(bunt.Beige),
		neat.NoLineWrap(),
	)
}

// CreateChartJS creates a page with ChartsJS to render the provided results
func CreateChartJS(filename string, data []BuildRunResultSet) error {
	const reportTemplate = `<!DOCTYPE html>
	<html>

	<head>
	    <meta charset="utf-8">
	    <script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.5.0/Chart.min.js"></script>
	</head>

	<body>
	    <div class="chart-container" style="position: relative; width:90vw;">
	        <canvas id="myChart"></canvas>
	    </div>

	<script>
	    var ctx = document.getElementById('myChart').getContext('2d');
	    var myChart = new Chart(ctx, {
	        type: 'bar',
	        data: {
	            labels: {{ .Labels }},
	            datasets: {{ .Datasets }},
	        },
	        options: {
	            title: {
	                display: true,
	                text: 'Build run times with different numbers of parallel builds'
	            },
	            tooltips: {
	                displayColors: true,
	                callbacks: {
	                    mode: 'x',
	                },
	            },
	            scales: {
	                xAxes: [{
	                    scaleLabel: {
	                        display: true,
	                        labelString: 'number of parallel builds'
	                    },
	                    stacked: true,
	                    gridLines: {
	                        display: false,
	                    }
	                }],
	                yAxes: [{
	                    scaleLabel: {
	                        display: true,
	                        labelString: 'time in seconds'
	                    },
	                    stacked: true,
	                    ticks: {
	                        beginAtZero: true,
	                    },
	                    type: 'linear',
	                }]
	            },
	            responsive: true,
	            maintainAspectRatio: true,
	            legend: { position: 'bottom' },
	        }
	    });
	</script>
	</body>
	</html>
	`

	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return err
	}

	type dataset struct {
		Label           string    `json:"label"`
		BackgroundColor string    `json:"backgroundColor"`
		Data            []float64 `json:"data"`
	}

	var (
		labels   = []string{}
		datasets = []dataset{
			{
				Label:           InternalProcessingTime,
				BackgroundColor: "#6cf9a6",
				Data:            []float64{},
			},
			{
				Label:           PodRampUpDuration,
				BackgroundColor: "#fdc10a",
				Data:            []float64{},
			},
			{
				Label:           TaskRunRampUpDuration,
				BackgroundColor: "#34a887",
				Data:            []float64{},
			},
			{
				Label:           BuildRunRampUpDuration,
				BackgroundColor: "#ad36a6",
				Data:            []float64{},
			},
		}
	)

	for _, buildRunResultSet := range data {
		labels = append(labels, fmt.Sprintf("%d", buildRunResultSet.NumberOfResults))

		datasets[0].Data = append(datasets[0].Data, buildRunResultSet.Median.ValueOf(InternalProcessingTime).Seconds())
		datasets[1].Data = append(datasets[1].Data, buildRunResultSet.Median.ValueOf(PodRampUpDuration).Seconds())
		datasets[2].Data = append(datasets[2].Data, buildRunResultSet.Median.ValueOf(TaskRunRampUpDuration).Seconds())
		datasets[3].Data = append(datasets[3].Data, buildRunResultSet.Median.ValueOf(BuildRunRampUpDuration).Seconds())
	}

	output, err := os.Create(filename)
	if err != nil {
		return err
	}

	type inputs struct {
		Labels   []string
		Datasets []dataset
	}

	return tmpl.Execute(output, inputs{
		Labels:   labels,
		Datasets: datasets,
	})
}
