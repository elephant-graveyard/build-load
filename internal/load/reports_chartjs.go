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
	"io"
	"strconv"
)

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
        text: {{ .Text }}
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
            labelString: {{ .LabelX }}
          },
          gridLines: {
            display: false,
          }
        }],
        yAxes: [{
          scaleLabel: {
            display: true,
            labelString: {{ .LabelY }}
          },
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

type dataset struct {
	Label           string    `json:"label"`
	BackgroundColor string    `json:"backgroundColor"`
	Data            []float64 `json:"data"`
}

type inputs struct {
	Text     string
	LabelX   string
	LabelY   string
	Labels   []string
	Datasets []dataset
}

func prepareDatasets() []dataset {
	return []dataset{
		{
			Label:           BuildrunCompletionTime,
			BackgroundColor: "#6cf9a6",
			Data:            []float64{},
		},
		{
			Label:           BuildrunControlTime,
			BackgroundColor: "#fdc10a",
			Data:            []float64{},
		},
		{
			Label:           TaskrunCompletionTime,
			BackgroundColor: "#34a887",
			Data:            []float64{},
		},
		{
			Label:           TaskrunControlTime,
			BackgroundColor: "#ad36a6",
			Data:            []float64{},
		},
		{
			Label:           PodCompletionTime,
			BackgroundColor: "#a064a6",
			Data:            []float64{},
		},
		{
			Label:           PodControlTime,
			BackgroundColor: "#ada469",
			Data:            []float64{},
		},
	}
}

// CreateBuildrunResultsChartJS creates a page with ChartJS to display the results of buildruns
func CreateBuildrunResultsChartJS(data []BuildRunResult, w io.Writer) error {
	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return err
	}

	var labels = []string{}
	var datasets = prepareDatasets()

	for i, buildRunResult := range data {
		labels = append(labels, strconv.Itoa(i+1))

		for j, value := range buildRunResult {
			datasets[j].Data = append(datasets[j].Data, value.Value.Seconds())
		}
	}

	return tmpl.Execute(w, inputs{
		Text:     "BuildRun times",
		LabelX:   "buildrun",
		LabelY:   "time in seconds",
		Labels:   labels,
		Datasets: datasets,
	})
}

// CreateChartJS creates a page with ChartsJS to render the provided results
func CreateChartJS(data []BuildRunResultSet, w io.Writer) error {
	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return err
	}

	var labels = []string{}
	var datasets = prepareDatasets()

	for _, buildRunResultSet := range data {
		labels = append(labels, fmt.Sprintf("%d", buildRunResultSet.NumberOfResults))

		datasets[0].Data = append(datasets[0].Data, buildRunResultSet.Median.ValueOf(BuildrunCompletionTime).Seconds())
		datasets[1].Data = append(datasets[1].Data, buildRunResultSet.Median.ValueOf(BuildrunControlTime).Seconds())
		datasets[2].Data = append(datasets[2].Data, buildRunResultSet.Median.ValueOf(TaskrunCompletionTime).Seconds())
		datasets[3].Data = append(datasets[3].Data, buildRunResultSet.Median.ValueOf(TaskrunControlTime).Seconds())
		datasets[4].Data = append(datasets[4].Data, buildRunResultSet.Median.ValueOf(PodCompletionTime).Seconds())
		datasets[5].Data = append(datasets[5].Data, buildRunResultSet.Median.ValueOf(PodControlTime).Seconds())
	}

	return tmpl.Execute(w, inputs{
		Text:     "Build run times with different numbers of parallel builds",
		LabelX:   "number of parallel builds",
		LabelY:   "time in seconds",
		Labels:   labels,
		Datasets: datasets,
	})
}
