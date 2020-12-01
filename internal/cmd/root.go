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
	"bytes"
	"fmt"
	"os"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/neat"
	"github.com/gonvenience/wrap"
	"github.com/homeport/build-load/internal/load"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var shipwriteBuildURL = bunt.Sprintf("CornflowerBlue{~https://github.com/shipwright-io/build~}")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "build-load",
	Short: fmt.Sprintf("Create synthetic load for %s", shipwriteBuildURL),
	Long:  fmt.Sprintf("Create synthetic load for %s", shipwriteBuildURL),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var headline string = "Error occurred"
		var code int = 1
		var buf bytes.Buffer

		switch terr := err.(type) {
		case wrap.ContextError:
			headline = fmt.Sprintf("Error: %s", terr.Context())
			buf.WriteString(terr.Cause().Error())

		default:
			buf.WriteString(terr.Error())
		}

		neat.Box(
			os.Stderr,
			headline,
			&buf,
			neat.HeadlineColor(bunt.Coral),
			neat.ContentColor(bunt.DimGray),
		)

		os.Exit(code)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&load.Debug, "debug", false, "enable additional output messages")
}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
