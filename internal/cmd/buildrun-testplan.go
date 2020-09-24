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
	"io/ioutil"
	"os"

	"github.com/homeport/build-load/internal/load"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var buildRunTestplanCmdSettings struct {
	testplanPath string
}

var buildRunTestplanCmd = &cobra.Command{
	Use:           "buildruns-testplan",
	Short:         "Creates and executes buildruns specified in a testplan",
	Long:          "Creates and executes buildruns specified in a testplan",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeAccess, err := load.NewKubeAccess()
		if err != nil {
			return err
		}

		testplan, err := loadTestPlan(buildRunTestplanCmdSettings.testplanPath)
		if err != nil {
			return err
		}

		return load.ExecuteTestPlan(*kubeAccess, *testplan)
	},
}

func init() {
	rootCmd.AddCommand(buildRunTestplanCmd)

	buildRunTestplanCmd.Flags().SortFlags = false
	buildRunTestplanCmd.PersistentFlags().SortFlags = false

	buildRunTestplanCmd.Flags().StringVar(&buildRunTestplanCmdSettings.testplanPath, "testplan", "", "testplan configuration file")
}

func loadTestPlan(path string) (*load.TestPlan, error) {
	var data []byte
	var err error

	switch path {
	case "-":
		data, err = ioutil.ReadAll(os.Stdin)

	default:
		data, err = ioutil.ReadFile(path)
	}

	if err != nil {
		return nil, err
	}

	var testplan load.TestPlan
	if err := yaml.Unmarshal(data, &testplan); err != nil {
		return nil, err
	}

	// apply implicit defaults
	for i := range testplan.Steps {
		if len(testplan.Steps[i].BuildRunSettings.Source.Dockerfile) == 0 {
			testplan.Steps[i].BuildRunSettings.Source.Dockerfile = "Dockerfile"
		}

		if len(testplan.Steps[i].BuildRunSettings.Source.ContextDir) == 0 {
			testplan.Steps[i].BuildRunSettings.Source.ContextDir = "/"
		}
	}

	return &testplan, nil
}
