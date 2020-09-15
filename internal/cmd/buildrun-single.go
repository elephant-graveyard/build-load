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

	"github.com/gonvenience/bunt"
	"github.com/homeport/build-load/internal/load"
	"github.com/spf13/cobra"
)

var buildRunOnceCmdSettings struct {
	parallel int
	config   load.BuildRunSettings
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

		if err := load.CheckSystemAndConfig(*kubeAccess, buildRunOnceCmdSettings.config, buildRunOnceCmdSettings.parallel); err != nil {
			return err
		}

		buildRunResults, err := load.ExecuteParallelBuildRuns(*kubeAccess, buildRunOnceCmdSettings.config, buildRunOnceCmdSettings.parallel)
		if err != nil {
			return err
		}

		fmt.Print(load.CalculateBuildRunResultSet(buildRunResults))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildRunOnceCmd)

	buildRunOnceCmd.Flags().SortFlags = false
	buildRunOnceCmd.PersistentFlags().SortFlags = false

	buildRunOnceCmd.Flags().IntVar(&buildRunOnceCmdSettings.parallel, "parallel", 1, "number of parallel buildruns")
	applyBuildRunSettingsFlags(buildRunOnceCmd, &buildRunOnceCmdSettings.config)
}
