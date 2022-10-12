/*
Copyright © 2022 The Homeport Team

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

package load_test

import (
	"fmt"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/homeport/build-load/internal/load"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/rand"

	shipwrightBuild "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func p[T any](t T) *T { return &t }

var _ = Describe("Kubernetes cluster based tests", func() {
	var kubeAccess *KubeAccess

	var withTemporaryNamespace = func(f func(string)) {
		name := rand.String(8)

		namespace, err := kubeAccess.Client.CoreV1().Namespaces().Create(
			kubeAccess.Context,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: name},
			},
			metav1.CreateOptions{},
		)

		Expect(err).ToNot(HaveOccurred())
		Expect(namespace).ToNot(BeNil())
		defer func() {
			Expect(kubeAccess.
				Client.
				CoreV1().
				Namespaces().
				Delete(kubeAccess.Context, name, metav1.DeleteOptions{}),
			).To(Succeed())
		}()

		f(name)
	}

	var withTemporaryClusterBuildStrategy = func(f func(shipwrightBuild.ClusterBuildStrategy)) {
		name := rand.String(8)

		cbs, err := kubeAccess.BuildClient.
			ShipwrightV1alpha1().
			ClusterBuildStrategies().
			Create(
				kubeAccess.Context,
				&shipwrightBuild.ClusterBuildStrategy{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
					Spec: shipwrightBuild.BuildStrategySpec{
						BuildSteps: []shipwrightBuild.BuildStep{
							{
								Container: corev1.Container{
									Name:    "no-op",
									Image:   "alpine:latest",
									Command: []string{"/bin/true"},
								},
							},
						},
					},
				},
				metav1.CreateOptions{},
			)

		Expect(err).ToNot(HaveOccurred())
		Expect(cbs).ToNot(BeNil())
		defer func() {
			Expect(kubeAccess.
				BuildClient.
				ShipwrightV1alpha1().
				ClusterBuildStrategies().
				Delete(kubeAccess.Context, name, metav1.DeleteOptions{}),
			).To(Succeed())
		}()

		f(*cbs)
	}

	BeforeEach(func() {
		var err error

		kubeAccess, err = NewKubeAccess()
		if err != nil {
			Skip("Skipping Kubernetes cluster based tests, because cluster config could not be obtained: " + err.Error())
		}

		Expect(kubeAccess).ToNot(BeNil())

		uri, err := url.Parse(kubeAccess.RestConfig.Host)
		Expect(err).ToNot(HaveOccurred())

		if uri.Hostname() != "127.0.0.1" && uri.Hostname() != "localhost" {
			Skip("Skipping Kubernetes cluster based tests, because cluster is not hosted on localhost, instead it is " + kubeAccess.RestConfig.Host)
		}
	})

	Context("using builds", func() {
		It("should create builds in a system", func() {
			withTemporaryNamespace(func(namespace string) {
				withTemporaryClusterBuildStrategy(func(cbs shipwrightBuild.ClusterBuildStrategy) {
					results, err := ExecuteBuilds(
						*kubeAccess,
						NamingConfig{
							Namespace: namespace,
							Prefix:    "test",
						},
						BuildConfig{
							ClusterBuildStrategy: cbs.Name,
							SourceURL:            "https://github.com/shipwright-io/sample-go",
							SourceContextDir:     "docker-build",
							SourceDockerfile:     "Dockerfile",
							OutputImageURL:       "registry.registry.svc.cluster.local:32222/test/something",
						},
						42,
					)

					Expect(err).ToNot(HaveOccurred())
					Expect(results).ToNot(BeEmpty())
				})
			})
		})
	})

	Context("using buildruns", func() {
		It("should check system and config to verify a buildrun would work", func() {
			withTemporaryClusterBuildStrategy(func(cbs shipwrightBuild.ClusterBuildStrategy) {
				Expect(CheckSystemAndConfig(
					*kubeAccess,
					BuildConfig{
						ClusterBuildStrategy: cbs.Name,
						SourceURL:            "https://github.com/shipwright-io/sample-go",
						SourceContextDir:     "docker-build",
						SourceDockerfile:     "Dockerfile",
						OutputImageURL:       "registry.registry.svc.cluster.local:32222/test/something",
					},
					1,
				)).To(Succeed())
			})
		})

		It("should execute a single buildrun using temporary strategy and the Go sample", func() {
			withTemporaryNamespace(func(namespace string) {
				withTemporaryClusterBuildStrategy(func(cbs shipwrightBuild.ClusterBuildStrategy) {
					buildRunName := rand.String(8)
					outputImageName := rand.String(8)

					result, err := ExecuteSingleBuildRun(
						*kubeAccess,
						namespace,
						buildRunName,
						shipwrightBuild.BuildSpec{
							Source: shipwrightBuild.Source{
								URL: p("https://github.com/shipwright-io/sample-go"),
							},
							Dockerfile: p("Dockerfile"),
							Strategy: shipwrightBuild.Strategy{
								Kind: p(shipwrightBuild.ClusterBuildStrategyKind),
								Name: cbs.Name,
							},
							Output: shipwrightBuild.Image{
								Image: fmt.Sprintf("registry.registry.svc.cluster.local:32222/test/%s", outputImageName),
							},
						},
						map[string]string{},
					)

					Expect(err).ToNot(HaveOccurred())
					Expect(result).ToNot(BeNil())
				})
			})
		})

		It("should execute parallel buildruns using temporary strategy and the Go sample", func() {
			withTemporaryNamespace(func(namespace string) {
				withTemporaryClusterBuildStrategy(func(cbs shipwrightBuild.ClusterBuildStrategy) {
					results, err := ExecuteParallelBuildRuns(
						*kubeAccess,
						NamingConfig{
							Namespace: namespace,
							Prefix:    "test",
						},
						BuildConfig{
							ClusterBuildStrategy: cbs.Name,
							SourceURL:            "https://github.com/shipwright-io/sample-go",
							OutputImageURL:       "registry.registry.svc.cluster.local:32222/test/prefix",
						},
						4,
					)

					Expect(err).ToNot(HaveOccurred())
					Expect(results).ToNot(BeEmpty())
				})
			})
		})

		It("should execute a series of buildruns using temporary strategy and the Go sample", func() {
			withTemporaryNamespace(func(namespace string) {
				withTemporaryClusterBuildStrategy(func(cbs shipwrightBuild.ClusterBuildStrategy) {
					resultSet, err := ExecuteSeriesOfParallelBuildRuns(
						*kubeAccess,
						NamingConfig{
							Namespace: namespace,
							Prefix:    "test",
						},
						BuildConfig{
							ClusterBuildStrategy: cbs.Name,
							SourceURL:            "https://github.com/shipwright-io/sample-go",
							OutputImageURL:       "registry.registry.svc.cluster.local:32222/test/prefix",
						},
						1, 4, 1,
					)

					Expect(err).ToNot(HaveOccurred())
					Expect(resultSet).ToNot(BeEmpty())
				})
			})
		})
	})

	Context("using testplans", func() {
		It("should execute a test plan", func() {
			const testplanTemplate = `
---
namespace: %s
steps:
- name: one
  buildSpec:
    source:
      url: "https://github.com/shipwright-io/sample-go"
    strategy:
      kind: ClusterBuildStrategy
      name: %s
    output:
      image: registry.registry.svc.cluster.local:32222/test

- name: two
  buildSpec:
    source:
      url: "https://github.com/shipwright-io/sample-go"
    strategy:
      kind: ClusterBuildStrategy
      name: %s
    output:
      image: registry.registry.svc.cluster.local:32222/test
`

			withTemporaryNamespace(func(namespace string) {
				withTemporaryClusterBuildStrategy(func(one shipwrightBuild.ClusterBuildStrategy) {
					withTemporaryClusterBuildStrategy(func(two shipwrightBuild.ClusterBuildStrategy) {
						testplan, err := NewTestPlan(strings.NewReader(fmt.Sprintf(testplanTemplate, namespace, one.Name, two.Name)))
						Expect(err).ToNot(HaveOccurred())
						Expect(testplan).ToNot(BeNil())

						Expect(ExecuteTestPlan(*kubeAccess, *testplan)).To(Succeed())
					})
				})
			})
		})
	})
})
