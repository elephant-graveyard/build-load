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

For now, you have to build it from source:

```sh
git clone https://github.com/homeport/build-load.git
cd build-load
make install
```

It will compile the binary into `/usr/local/bin`.

Alternatively, run `make build` and pick the respective binary for your operating system from the `binaries` directory.
