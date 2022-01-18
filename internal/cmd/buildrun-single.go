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
	"github.com/spf13/cobra"

	"github.com/homeport/build-load/internal/load"
)

var buildRunOnceCmdSettings struct {
	parallel  int
	namingCfg load.NamingConfig
	buildCfg  load.BuildConfig

	htmlOutput string
	csvOutput  string
}

var buildRunOnceCmd = &cobra.Command{
	Use:           "buildruns",
	Short:         "Creates a single buildrun",
	Long:          bunt.Sprintf("*Creates a single buildrun*\n%s", buildRunSettingsDescription),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeAccess, err := load.NewKubeAccess()
		if err != nil {
			return err
		}

		if err := load.CheckSystemAndConfig(*kubeAccess, buildRunOnceCmdSettings.buildCfg, buildRunOnceCmdSettings.parallel); err != nil {
			return err
		}

		buildRunResults, err := load.ExecuteParallelBuildRuns(*kubeAccess, buildRunOnceCmdSettings.namingCfg, buildRunOnceCmdSettings.buildCfg, buildRunOnceCmdSettings.parallel)
		if err != nil {
			return err
		}

		if err := store(buildRunOnceCmdSettings.htmlOutput, func(w io.Writer) error { return load.CreateBuildrunResultsChartJS(buildRunResults, w) }); err != nil {
			return err
		}

		if err := store(buildRunOnceCmdSettings.csvOutput, func(w io.Writer) error { return load.CreateResultsCSV(buildRunResults, w) }); err != nil {
			return err
		}

		fmt.Print(load.CalculateResultSet(buildRunResults, "buildrun"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildRunOnceCmd)

	buildRunOnceCmd.Flags().SortFlags = false
	buildRunOnceCmd.PersistentFlags().SortFlags = false

	buildRunOnceCmd.Flags().IntVar(&buildRunOnceCmdSettings.parallel, "parallel", 1, "number of parallel buildruns")

	buildRunOnceCmd.Flags().StringVar(&buildRunOnceCmdSettings.htmlOutput, "html", "", "filename of the HTML report")
	buildRunOnceCmd.Flags().StringVar(&buildRunOnceCmdSettings.csvOutput, "csv", "", "filename of the CSV report")

	applyNamingFlags(buildRunOnceCmd, &buildRunOnceCmdSettings.namingCfg)
	applyBuildRunSettingsFlags(buildRunOnceCmd, &buildRunOnceCmdSettings.buildCfg)
}
