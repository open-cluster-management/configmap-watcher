# Copyright Contributors to the Open Cluster Management project

#!/bin/bash
set -e

export DOCKER_IMAGE_AND_TAG=${1}
# make docker/run
make go/gosec-install
make go-coverage
