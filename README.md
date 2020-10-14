# build-load

Create synthetic load for [shipwright-io/build](https://github.com/shipwright-io/build)

![build-load](.docs/example-output.png?raw=true "build-load example output")

## Examples

### Kaniko buildrun

```sh
build-load \
  buildruns \
  --build-type=kaniko \
  --cluster-build-strategy=kaniko \
  --source-url=https://github.com/EmilyEmily/docker-simple \
  --output-registry-hostname=docker.io \
  --output-registry-namespace=boatyard \
  --output-registry-secret-ref=registry-credentials
```

### Buildpacks buildrun

```sh
build-load \
  buildruns \
  --build-type=buildpack \
  --cluster-build-strategy=buildpacks-v3 \
  --source-url=https://github.com/sclorg/nodejs-ex \
  --output-registry-hostname=docker.io \
  --output-registry-namespace=boatyard \
  --output-registry-secret-ref=registry-credentials
```

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

To promote a new version, create a new release in GitHub. This will trigger a build of the binaries in Travis CI and uploads them to the respective release automatically within a couple of minutes. The best way to create a new release is to use the GitHub CLI.

As a pre-requisite, a [`semver` tool](https://github.com/fsaintjacques/semver-tool) and the [GitHub CLI](https://github.com/cli/cli) are suggested on your system, for example on macOS:

```sh
curl --silent --fail --location https://raw.githubusercontent.com/fsaintjacques/semver-tool/master/src/semver --output /usr/local/bin/semver && chmod a+rx /usr/local/bin/semver
brew install gh
```

Example for creating a new patch release:

```sh
VERSION="v$(semver bump patch "$(git describe --tags --abbrev=0)")"
gh release create "$VERSION" --title "build-load release $VERSION"
```

The CLI will interactively prompt for missing details like the notes.
