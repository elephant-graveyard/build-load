# Copyright © 2020 The Homeport Team
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

version := $(shell git describe --tags 2>/dev/null || (git rev-parse HEAD | cut -c-8))
sources := $(wildcard cmd/*/*.go internal/*/*.go)
goos := $(shell uname | tr '[:upper:]' '[:lower:]')
goarch := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

.PHONY: all
all: verify build

.PHONY: clean
clean:
	@go clean -cache $(shell go list ./...)
	@rm -rf dist

.PHONY: verify
verify:
	@go mod download
	@go mod verify

.PHONY: test
test: $(sources)
	@ginkgo \
		-r \
		-p \
		--randomize-all \
		--randomize-suites \
		--fail-on-pending \
		--nodes=4 \
		--compilers=2 \
		--slow-spec-threshold=2m \
		--race \
		--trace \
		--cover

.PHONY: build
build: $(sources)
	@go build ./...

.PHONY: install
install: $(sources)
	@CGO_ENABLED=0 GOOS=$(goos) GOARCH=$(goarch) go build \
		-trimpath \
		-ldflags='-s -w -extldflags "-static" -X github.com/homeport/build-load/internal/cmd.version=$(version)' \
		-o /usr/local/bin/build-load \
		cmd/build-load/main.go

.PHONY: install-user
install-user: $(sources)
	@CGO_ENABLED=0 GOOS=$(goos) GOARCH=$(goarch) go build \
		-trimpath \
		-ldflags='-s -w -extldflags "-static" -X github.com/homeport/build-load/internal/cmd.version=$(version)' \
		-o "${HOME}/bin/build-load" \
		cmd/build-load/main.go
