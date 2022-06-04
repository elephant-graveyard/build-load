# build-load [![License](https://img.shields.io/github/license/homeport/build-load.svg)](https://github.com/homeport/build-load/blob/main/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/homeport/build-load)](https://goreportcard.com/report/github.com/homeport/build-load) [![Tests](https://github.com/homeport/build-load/actions/workflows/tests.yml/badge.svg)](https://github.com/homeport/build-load/actions/workflows/tests.yml) [![Codecov](https://img.shields.io/codecov/c/github/homeport/build-load/main.svg)](https://codecov.io/gh/homeport/build-load) [![Go Reference](https://pkg.go.dev/badge/github.com/homeport/build-load.svg)](https://pkg.go.dev/github.com/homeport/build-load) [![Release](https://img.shields.io/github/release/homeport/build-load.svg)](https://github.com/homeport/build-load/releases/latest)

Create synthetic load for [shipwright-io/build](https://github.com/shipwright-io/build)

![build-load](.docs/example-output.png?raw=true "build-load example output")

## Examples

### Build Runs

#### Kaniko buildrun

```sh
build-load \
  buildruns \
  --namespace=test-namespace \
  --cluster-build-strategy=kaniko \
  --source-url=https://github.com/EmilyEmily/docker-simple \
  --output-image-url=docker.io/boatyard \
  --output-secret-ref=registry-credentials
```

#### Buildpacks buildrun

```sh
build-load \
  buildruns \
  --namespace=test-namespace \
  --cluster-build-strategy=buildpacks-v3 \
  --source-url=https://github.com/sclorg/nodejs-ex \
  --output-image-url=docker.io/boatyard \
  --output-secret-ref=registry-credentials
```

### Test Plan

#### Use Test Plan YAML

```yaml
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
```

Run the test plan using:

```sh
build-load \
  buildruns-testplan \
  --testplan testplan.yml
```

The test plan can also be piped into the program using `-` as the filename and a here-doc YAML.

## Setup

### Download via Homebrew

```sh
brew install homeport/tap/build-load
```

### Download via Curl to Pipe

The download script will work for Linux and macOS systems.

```sh
curl -fsL https://git.io/JTYKj | bash
```

### Build from Source

It will compile the binary into `/usr/local/bin`.

```sh
git clone https://github.com/homeport/build-load.git
cd build-load
make install
```

Alternatively, run `make build` and pick the respective binary for your operating system from the `binaries` directory.

## Development

### Release Process

To promote a new version, create a new release in GitHub. This will trigger a build of the binaries using GitHub Actions and uploads them to the respective release automatically within a couple of minutes. As part of this process, the Homebrew tap is updated, too. The best way to create a new release is to use the GitHub CLI.

As a pre-requisite, a [`semver` tool](https://github.com/fsaintjacques/semver-tool) and the [GitHub CLI](https://github.com/cli/cli) are suggested on your system, for example on macOS:

- `semver` tool

  ```sh
  curl --silent --fail --location https://raw.githubusercontent.com/fsaintjacques/semver-tool/master/src/semver --output /usr/local/bin/semver && chmod a+rx /usr/local/bin/semver
  ```

- GitHub CLI

  ```sh
  brew install gh
  ```

Example for creating a new patch release:

```sh
VERSION="v$(semver bump patch "$(git describe --tags --abbrev=0)")"
gh release create "$VERSION"
```

The CLI will interactively prompt for more details. You can leave everything empty, because the automation will set a title and release notes automatically.
