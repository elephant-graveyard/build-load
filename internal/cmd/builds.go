/*
Copyright © 2021 The Homeport Team

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
	"github.com/gonvenience/wrap"
	"github.com/homeport/build-load/internal/load"
	"github.com/spf13/cobra"
)

var buildsCmdSettings struct {
	count     int
	namingCfg load.NamingConfig
	buildCfg  load.BuildConfig

	htmlOutput string
	csvOutput  string
}

var buildsCmd = &cobra.Command{
	Use:           "builds",
	Short:         "Creates a series of builds",
	Long:          bunt.Sprintf("*Creates a series of builds*\n\nWaits for them to be registered."),
	SilenceUsage:  true,
	SilenceErrors: true,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		if buildsCmdSettings.count <= 0 {
			return wrap.Errorf(
				fmt.Errorf(cmd.UsageString()),
				"input parameter for count is out bounds",
			)
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		kubeAccess, err := load.NewKubeAccess()
		if err != nil {
			return err
		}

		buildResults, err := load.ExecuteBuilds(*kubeAccess, buildsCmdSettings.namingCfg, buildsCmdSettings.buildCfg, buildsCmdSettings.count)
		if err != nil {
			return err
		}

		fmt.Print(load.CalculateResultSet(buildResults, "build"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildsCmd)

	buildsCmd.Flags().SortFlags = false
	buildsCmd.PersistentFlags().SortFlags = false

	buildsCmd.Flags().IntVar(&buildsCmdSettings.count, "count", 5, "Number of builds")

	applyNamingFlags(buildsCmd, &buildsCmdSettings.namingCfg)
	applyBuildRunSettingsFlags(buildsCmd, &buildsCmdSettings.buildCfg)
}
