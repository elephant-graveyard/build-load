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

const singleBuildrunTemplate = `<!DOCTYPE html>
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
    type: 'line',
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

const seriesBuildrunTemplate = `<!DOCTYPE html>
<html>

<head>
  <meta charset="utf-8">
  <script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.5.0/Chart.min.js"></script>
  <style>
	canvas{
		-moz-user-select: none;
	}
	.chart-container {
		width: 800px;
		margin-left: 40px;
		margin-right: 40px;
		border: 1px solid rgb(240, 240, 240);
		border-radius: 4px;
		margin: 4px;
	}
	.container {
		display: flex;
		flex-direction: column;
		flex-wrap: wrap;
		justify-content: center;
	}
  </style>
  </head>

<body>
  <div class="container">
	<div class="chart-container">
		<canvas id="myMedianChart"></canvas>
	</div>
	<div class="chart-container">
		<canvas id="myMeanChart"></canvas>
	</div>
  </div>

<script>
  var ctx = document.getElementById('myMedianChart').getContext('2d');
  new Chart(ctx, {
    type: 'line',
    data: {
      labels: {{ .Labels }},
	  datasets: {{ .MedDatasets }},
    },
    options: {
      title: {
        display: true,
        text: {{ .MedText }}
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
  var ctxb = document.getElementById('myMeanChart').getContext('2d');
  new Chart(ctxb, {
    type: 'line',
    data: {
      labels: {{ .Labels }},
	  datasets: {{ .MeanDatasets }},
    },
    options: {
      title: {
        display: true,
        text: {{ .MeanText }}
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
	Fill            bool      `json:"fill"`
	BackgroundColor string    `json:"backgroundColor"`
	BorderColor     string    `json:"borderColor"`
	Data            []float64 `json:"data"`
}

type singleInputs struct {
	Text     string
	LabelX   string
	LabelY   string
	Labels   []string
	Datasets []dataset
}

type seriesInputs struct {
	MedText      string
	MeanText     string
	LabelX       string
	LabelY       string
	Labels       []string
	MedDatasets  []dataset
	MeanDatasets []dataset
}

func prepareDatasets() []dataset {
	return []dataset{
		{
			Label:           BuildrunCompletionTime,
			Fill:            false,
			BackgroundColor: "#6cf9a6",
			BorderColor:     "#6cf9a6",
			Data:            []float64{},
		},
		{
			Label:           BuildrunControlTime,
			Fill:            false,
			BackgroundColor: "#fdc10a",
			BorderColor:     "#fdc10a",
			Data:            []float64{},
		},
		{
			Label:           TaskrunCompletionTime,
			Fill:            false,
			BackgroundColor: "#34a887",
			BorderColor:     "#34a887",
			Data:            []float64{},
		},
		{
			Label:           TaskrunControlTime,
			Fill:            false,
			BackgroundColor: "#ad36a6",
			BorderColor:     "#ad36a6",
			Data:            []float64{},
		},
		{
			Label:           PodCompletionTime,
			Fill:            false,
			BackgroundColor: "#a064a6",
			BorderColor:     "#a064a6",
			Data:            []float64{},
		},
		{
			Label:           PodControlTime,
			Fill:            false,
			BackgroundColor: "#ada469",
			BorderColor:     "#ada469",
			Data:            []float64{},
		},
	}
}

// CreateBuildrunResultsChartJS creates a page with ChartJS to display the results of buildruns
func CreateBuildrunResultsChartJS(data []BuildRunResult, w io.Writer) error {
	tmpl, err := template.New("report").Parse(singleBuildrunTemplate)
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

	return tmpl.Execute(w, singleInputs{
		Text:     "BuildRun times",
		LabelX:   "buildrun",
		LabelY:   "time in seconds",
		Labels:   labels,
		Datasets: datasets,
	})
}

// CreateChartJS creates a page with ChartsJS to render the provided results
func CreateChartJS(data []BuildRunResultSet, w io.Writer) error {
	tmpl, err := template.New("report").Parse(seriesBuildrunTemplate)
	if err != nil {
		return err
	}

	var labels = []string{}
	var medDatasets = prepareDatasets()
	var meanDatasets = prepareDatasets()

	for _, buildRunResultSet := range data {
		labels = append(labels, fmt.Sprintf("%d", buildRunResultSet.NumberOfResults))

		medDatasets[0].Data = append(medDatasets[0].Data, buildRunResultSet.Median.ValueOf(BuildrunCompletionTime).Seconds())
		medDatasets[1].Data = append(medDatasets[1].Data, buildRunResultSet.Median.ValueOf(BuildrunControlTime).Seconds())
		medDatasets[2].Data = append(medDatasets[2].Data, buildRunResultSet.Median.ValueOf(TaskrunCompletionTime).Seconds())
		medDatasets[3].Data = append(medDatasets[3].Data, buildRunResultSet.Median.ValueOf(TaskrunControlTime).Seconds())
		medDatasets[4].Data = append(medDatasets[4].Data, buildRunResultSet.Median.ValueOf(PodCompletionTime).Seconds())
		medDatasets[5].Data = append(medDatasets[5].Data, buildRunResultSet.Median.ValueOf(PodControlTime).Seconds())

		meanDatasets[0].Data = append(meanDatasets[0].Data, buildRunResultSet.Mean.ValueOf(BuildrunCompletionTime).Seconds())
		meanDatasets[1].Data = append(meanDatasets[1].Data, buildRunResultSet.Mean.ValueOf(BuildrunControlTime).Seconds())
		meanDatasets[2].Data = append(meanDatasets[2].Data, buildRunResultSet.Mean.ValueOf(TaskrunCompletionTime).Seconds())
		meanDatasets[3].Data = append(meanDatasets[3].Data, buildRunResultSet.Mean.ValueOf(TaskrunControlTime).Seconds())
		meanDatasets[4].Data = append(meanDatasets[4].Data, buildRunResultSet.Mean.ValueOf(PodCompletionTime).Seconds())
		meanDatasets[5].Data = append(meanDatasets[5].Data, buildRunResultSet.Mean.ValueOf(PodControlTime).Seconds())
	}

	return tmpl.Execute(w, seriesInputs{
		MedText:      "Build run times with different numbers of parallel builds (Median Chart)",
		MeanText:     "Build run times with different numbers of parallel builds (Mean Chart)",
		LabelX:       "number of parallel builds",
		LabelY:       "time in seconds",
		Labels:       labels,
		MedDatasets:  medDatasets,
		MeanDatasets: meanDatasets,
	})
}
