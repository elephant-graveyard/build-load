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
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/gonvenience/bunt"
	"github.com/spf13/cobra"

	"github.com/homeport/build-load/internal/load"
)

var buildRunSettingsDescription = bunt.Sprintf(`
Prerequisites:
  _Create a Kubernetes secret with the access credentials of a container registry, for example Docker Hub:_
    LightSteelBlue{kubectl --namespace <namespace> create secret docker-registry \
      <secret-name>
      --docker-server=https://index.docker.io/v1/ \
      --docker-username=<docker-hub-username> \
      --docker-password=<docker-hub-password>}

Examples:
  _Run a Kaniko based build with no concurrency:_
    LightSteelBlue{build-load \
      buildruns \
      --cluster-build-strategy=kaniko \
      --source-url=https://github.com/EmilyEmily/docker-simple \
      --output-image-url=docker.io/boatyard \
      --output-secret-ref=registry-credentials}

  _Run a Buildpacks V3 based build with ten builds in parallel:_
    LightSteelBlue{build-load \
      buildruns \
      --cluster-build-strategy=buildpacks-v3 \
      --source-url=https://github.com/sclorg/nodejs-ex \
      --output-image-url=docker.io/boatyard \
      --output-secret-ref=registry-credentials \
      --parallel=10}
`)

func applyNamingFlags(cmd *cobra.Command, namingCfg *load.NamingConfig) {
	pf := cmd.PersistentFlags()

	pf.StringVar(&namingCfg.Namespace, "namespace", "default", "namespace to test in")
	pf.StringVar(&namingCfg.Prefix, "prefix", "test", "prefix for kube resource names")
}

func applyBuildRunSettingsFlags(cmd *cobra.Command, buildCfg *load.BuildConfig) {
	pf := cmd.PersistentFlags()

	pf.StringVar(&buildCfg.ClusterBuildStrategy, "cluster-build-strategy", "", "specify which cluster build strategy to be tested")

	pf.BoolVar(&buildCfg.GenerateServiceAccount, "generate-service-account", true, "generate service account for each build")

	pf.StringVar(&buildCfg.SourceURL, "source-url", "", "specify source URL to build from")
	pf.StringVar(&buildCfg.SourceContextDir, "source-context", "/", "specify directory to be used in the source repository")
	pf.StringVar(&buildCfg.SourceRevision, "source-revision", "master", "specify the branch, tag, or commit to be used")
	pf.StringVar(&buildCfg.SourceSecretRef, "source-secret", "", "specify secret to be used to access the source")
	pf.StringVar(&buildCfg.SourceDockerfile, "dockerfile", "Dockerfile", "specify name of the docker file for kaniko builds")
	pf.BoolVar(&buildCfg.SkipVerifySourceRepository, "skip-verify-repository", false, "skip the verification of the source repository")

	pf.StringVar(&buildCfg.OutputImageURL, "output-image-url", "", "output image URL")
	pf.StringVar(&buildCfg.OutputSecretRef, "output-secret-ref", "", "secret that contains the access credentials for the output registry")

	pf.DurationVar(&buildCfg.Timeout, "timeout", time.Duration(0), "defines the maximum runtime of a build run")

	pf.BoolVar(&buildCfg.SkipDelete, "skip-delete", false, "skip the clean-up of resources, which means no deletion of build, buildrun, and output image")

	cobra.MarkFlagRequired(pf, "source-url")
	cobra.MarkFlagRequired(pf, "output-image-url")
}

func store(filename string, f func(w io.Writer) error) error {
	if len(filename) == 0 {
		return nil
	}

	var buf bytes.Buffer
	f(&buf)
	return ioutil.WriteFile(filename, buf.Bytes(), os.FileMode(0644))
}
