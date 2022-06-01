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

package load_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	MockLabel1 = "mock #1"
	MockLabel2 = "mock #2"
	MockLabel3 = "mock #3"
	MockLabel4 = "mock #4"
	MockLabel5 = "mock #5"
)

func TestLoad(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Load Suite")
}
