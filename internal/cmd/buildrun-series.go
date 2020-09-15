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

package cmd

import (
	"github.com/gonvenience/bunt"
	"github.com/homeport/build-load/internal/load"
	"github.com/spf13/cobra"
)

var buildRunSeriesCmdSettings struct {
	buildTestsMin       int
	buildTestsMax       int
	buildTestsIncrement int
	config              load.BuildRunSettings
}

var buildRunSeriesCmd = &cobra.Command{
	Use:           "buildruns-series",
	Short:         "Creates a series of buildruns",
	Long:          bunt.Sprintf("*Creates a series of buildruns*\n\nCheck _buildruns_ command help for more details and examples regarding buildrun specific flags."),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeAccess, err := load.NewKubeAccess()
		if err != nil {
			return err
		}

		if err := load.CheckSystemAndConfig(*kubeAccess, buildRunSeriesCmdSettings.config, buildRunSeriesCmdSettings.buildTestsMax); err != nil {
			return err
		}

		results, err := load.ExecuteSeriesOfParallelBuildRuns(*kubeAccess, buildRunSeriesCmdSettings.config, buildRunSeriesCmdSettings.buildTestsMin, buildRunSeriesCmdSettings.buildTestsMax, buildRunSeriesCmdSettings.buildTestsIncrement)
		if err != nil {
			return err
		}

		return load.CreateChartJS("report.html", results)
	},
}

func init() {
	rootCmd.AddCommand(buildRunSeriesCmd)

	buildRunSeriesCmd.Flags().SortFlags = false
	buildRunSeriesCmd.PersistentFlags().SortFlags = false

	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsMin, "build-tests-min", 5, "lowest number of parallel builds to test")
	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsMax, "build-tests-max", 100, "highest number of parallel builds to test")
	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsIncrement, "build-tests-increment", 5, "increment for spinning up the number of parallel tests")
	applyBuildRunSettingsFlags(buildRunSeriesCmd, &buildRunSeriesCmdSettings.config)
}
