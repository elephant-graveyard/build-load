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

version := $(shell git describe --tags --abbrev=0 2>/dev/null || (git rev-parse HEAD | cut -c-8))
sources := $(wildcard cmd/*/*.go internal/*/*.go)
goos := $(shell uname | tr '[:upper:]' '[:lower:]')
goarch := $(shell uname -m | sed 's/x86_64/amd64/')

.PHONY: all
all: clean verify build

.PHONY: clean
clean:
	@GO111MODULE=on go clean -cache $(shell go list ./...)
	@rm -rf binaries

.PHONY: verify
verify:
	@GO111MODULE=on go mod download
	@GO111MODULE=on go mod verify

.PHONY: build
build: binaries/build-load-linux-amd64 binaries/build-load-darwin-amd64
	@/bin/sh -c "echo '\n\033[1mSHA sum of compiled binaries:\033[0m'"
	@shasum -a256 binaries/build-load-linux-amd64 binaries/build-load-darwin-amd64
	@echo

.PHONY: install
install: $(sources)
	@GO111MODULE=on CGO_ENABLED=0 GOOS=$(goos) GOARCH=$(goarch) go build \
		-tags netgo \
		-ldflags='-s -w -extldflags "-static" -X github.com/homeport/build-load/internal/cmd.version=$(version)' \
		-o /usr/local/bin/build-load \
		cmd/build-load/main.go

binaries/build-load-linux-amd64: $(sources)
	@GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-tags netgo \
		-ldflags='-s -w -extldflags "-static" -X github.com/homeport/build-load/internal/cmd.version=$(version)' \
		-o binaries/build-load-linux-amd64 \
		cmd/build-load/main.go

binaries/build-load-darwin-amd64: $(sources)
	@GO111MODULE=on CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		-tags netgo \
		-ldflags='-s -w -extldflags "-static" -X github.com/homeport/build-load/internal/cmd.version=$(version)' \
		-o binaries/build-load-darwin-amd64 \
		cmd/build-load/main.go
