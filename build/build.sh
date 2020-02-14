#!/bin/bash
set -e

export DOCKER_IMAGE_AND_TAG=${1}
export GOARCH=$(go env GOARCH)
make go-build
make docker/build
