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
	"fmt"
	"io"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/wrap"
	"github.com/spf13/cobra"

	"github.com/homeport/build-load/internal/load"
)

var buildRunSeriesCmdSettings struct {
	buildTestsMin       int
	buildTestsMax       int
	buildTestsIncrement int
	namingCfg           load.NamingConfig
	buildCfg            load.BuildConfig

	htmlOutput string
	csvOutput  string
}

var buildRunSeriesCmd = &cobra.Command{
	Use:           "buildruns-series",
	Short:         "Creates a series of buildruns",
	Long:          bunt.Sprintf("*Creates a series of buildruns*\n\nCheck _buildruns_ command help for more details and examples regarding buildrun specific flags."),
	SilenceUsage:  true,
	SilenceErrors: true,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		if buildRunSeriesCmdSettings.buildTestsMin <= 0 ||
			buildRunSeriesCmdSettings.buildTestsMax <= 0 ||
			buildRunSeriesCmdSettings.buildTestsIncrement <= 0 ||
			buildRunSeriesCmdSettings.buildTestsMin > buildRunSeriesCmdSettings.buildTestsMax {
			return wrap.Errorf(
				fmt.Errorf(cmd.UsageString()),
				"input parameters for min, max, and increment are out of bounds",
			)
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		kubeAccess, err := load.NewKubeAccess()
		if err != nil {
			return err
		}

		if err := load.CheckSystemAndConfig(*kubeAccess, buildRunSeriesCmdSettings.buildCfg, buildRunSeriesCmdSettings.buildTestsMax); err != nil {
			return err
		}

		results, err := load.ExecuteSeriesOfParallelBuildRuns(*kubeAccess, buildRunSeriesCmdSettings.namingCfg, buildRunSeriesCmdSettings.buildCfg, buildRunSeriesCmdSettings.buildTestsMin, buildRunSeriesCmdSettings.buildTestsMax, buildRunSeriesCmdSettings.buildTestsIncrement)
		if err != nil {
			return err
		}

		if err := store(buildRunSeriesCmdSettings.htmlOutput, func(w io.Writer) error { return load.CreateChartJS(results, w) }); err != nil {
			return err
		}

		if err := store(buildRunSeriesCmdSettings.csvOutput, func(w io.Writer) error { return load.CreateResultSetCSV(results, w) }); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildRunSeriesCmd)

	buildRunSeriesCmd.Flags().SortFlags = false
	buildRunSeriesCmd.PersistentFlags().SortFlags = false

	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsMin, "build-tests-min", 5, "lowest number of parallel builds to test (must be greater than zero)")
	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsMax, "build-tests-max", 100, "highest number of parallel builds to test (must be greater than zero and min)")
	buildRunSeriesCmd.Flags().IntVar(&buildRunSeriesCmdSettings.buildTestsIncrement, "build-tests-increment", 5, "increment for spinning up the number of parallel tests (must be greater than zero)")

	buildRunSeriesCmd.Flags().StringVar(&buildRunSeriesCmdSettings.htmlOutput, "html", "", "filename of the HTML report")
	buildRunSeriesCmd.Flags().StringVar(&buildRunSeriesCmdSettings.csvOutput, "csv", "", "filename of the CSV report")

	applyNamingFlags(buildRunSeriesCmd, &buildRunSeriesCmdSettings.namingCfg)
	applyBuildRunSettingsFlags(buildRunSeriesCmd, &buildRunSeriesCmdSettings.buildCfg)
}
