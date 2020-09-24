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

const (
	// BuildTypeKaniko is the name of the kaniko build strategy
	BuildTypeKaniko = "kaniko"

	// BuildTypeBuildpacks is the name of the buildpacks build strategy
	BuildTypeBuildpacks = "buildpacks"
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
      --build-type=kaniko \
      --cluster-build-strategy=kaniko \
      --source-url=https://github.com/EmilyEmily/docker-simple \
      --output-registry-hostname=docker.io \
      --output-registry-namespace=boatyard \
      --output-registry-secret-ref=registry-credentials}

  _Run a Buildpacks V3 based build with ten builds in parallel:_
    LightSteelBlue{build-load \
      buildruns \
      --build-type=buildpack \
      --cluster-build-strategy=buildpacks-v3 \
      --source-url=https://github.com/sclorg/nodejs-ex \
      --output-registry-hostname=docker.io \
      --output-registry-namespace=boatyard \
      --output-registry-secret-ref=registry-credentials
      --parallel=10}
`)

func applyBuildRunSettingsFlags(cmd *cobra.Command, config *load.BuildRunSettings) {
	pf := cmd.PersistentFlags()

	pf.StringVar(&config.Namespace, "namespace", "default", "namespace to test in")
	pf.StringVar(&config.BuildType, "build-type", "", "build type to be tested")
	pf.StringVar(&config.ClusterBuildStrategy, "cluster-build-strategy", "", "specify which cluster build strategy to be tested")

	pf.StringVar(&config.Name, "name", "test", "name to used for kube resources")
	pf.StringVar(&config.Prefix, "prefix", "load", "prefix for kube resource names")

	pf.StringVar(&config.Source.URL, "source-url", "", "specify source URL to build from")
	pf.StringVar(&config.Source.ContextDir, "source-context", "/", "specify directory to be used in the source repository")
	pf.StringVar(&config.Source.SecretRef, "source-secret", "", "specify secret to be used to access the source")

	pf.StringVar(&config.Source.Dockerfile, "dockerfile", "Dockerfile", "specify name of the docker file for kaniko builds")

	pf.StringVar(&config.Output.RegistryHostname, "output-registry-hostname", "", "output registry hostname")
	pf.StringVar(&config.Output.RegistryNamespace, "output-registry-namespace", "", "output registry namespace")
	pf.StringVar(&config.Output.SecretRef, "output-registry-secret-ref", "", "secret that contains the access credentials for the output registry")

	cobra.MarkFlagRequired(pf, "build-type")
	cobra.MarkFlagRequired(pf, "source-url")
	cobra.MarkFlagRequired(pf, "output-registry-hostname")
	cobra.MarkFlagRequired(pf, "output-registry-namespace")
	cobra.MarkFlagRequired(pf, "output-registry-secret-ref")
}
