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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/homeport/build-load/internal/load"
)

var buildRunTestplanCmdSettings struct {
	namespace              string
	generateServiceAccount bool
	testplanPath           string
}

var testplanCmdLong = `Run buildruns configured as steps in a testplan YAML file.

Example:
---
namespace: test-namespace
steps:
- name: kaniko
  buildSpec:
    source:
      url: https://github.com/EmilyEmily/docker-simple
      contextDir: /
    strategy:
      name: kaniko
      kind: ClusterBuildStrategy
    dockerfile: Dockerfile
    output:
      image: docker.io/boatyard
      credentials:
        name: reg-cred

- name: buildpacks
  buildSpec:
    source:
      url: https://github.com/sclorg/nodejs-ex
      contextDir: /
    strategy:
      name: buildpacks-v3
      kind: ClusterBuildStrategy
    output:
      image: docker.io/boatyard
      credentials:
        name: reg-cred

`

var buildRunTestplanCmd = &cobra.Command{
	Use:           "buildruns-testplan",
	Short:         "Creates and executes buildruns specified in a testplan",
	Long:          testplanCmdLong,
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

		// Override testplan namespace if command line flag namespace is used
		if len(buildRunTestplanCmdSettings.namespace) > 0 {
			testplan.Namespace = buildRunTestplanCmdSettings.namespace
		}

		return load.ExecuteTestPlan(*kubeAccess, *testplan)
	},
}

func init() {
	rootCmd.AddCommand(buildRunTestplanCmd)

	buildRunTestplanCmd.Flags().SortFlags = false
	buildRunTestplanCmd.PersistentFlags().SortFlags = false

	buildRunTestplanCmd.Flags().StringVar(&buildRunTestplanCmdSettings.namespace, "namespace", "", "namespace to run tests in (takes precedence over namespace in testplan YAML)")
	buildRunTestplanCmd.Flags().BoolVar(&buildRunTestplanCmdSettings.generateServiceAccount, "generate-service-account", true, "generate service account for build")
	buildRunTestplanCmd.Flags().StringVar(&buildRunTestplanCmdSettings.testplanPath, "testplan", "", "testplan configuration file")

	_ = cobra.MarkFlagRequired(buildRunTestplanCmd.Flags(), "testplan")
}

func loadTestPlan(path string) (*load.TestPlan, error) {
	switch path {
	case "-":
		return load.NewTestPlan(os.Stdin)

	default:
		file, err := os.Open(filepath.Clean(path))
		if err != nil {
			return nil, err
		}

		return load.NewTestPlan(file)
	}
}
