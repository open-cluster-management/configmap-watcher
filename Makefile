include Configfile
# Copyright 2019 The Jetstack cert-manager contributors.
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

# GITHUB_USER containing '@' char must be escaped with '%40'
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')
GITHUB_TOKEN ?=

USE_VENDORIZED_BUILD_HARNESS ?=

ifndef USE_VENDORIZED_BUILD_HARNESS
-include $(shell curl -s -H 'Authorization: token ${GITHUB_TOKEN}' -H 'Accept: application/vnd.github.v4.raw' -L https://api.github.com/repos/open-cluster-management/build-harness-extensions/contents/templates/Makefile.build-harness-bootstrap -o .build-harness-bootstrap; echo .build-harness-bootstrap)
else
-include vbh/.build-harness-bootstrap
endif

.PHONY: default
default::
	@echo "Build Harness Bootstrapped"

.PHONY: dependencies
dependencies:
	go mod tidy
	go mod vendor

.PHONY: lint test
lint:
	@echo "Linting disabled."

test:
	@echo "Testing disabled."

.PHONY: go-coverage
go-coverage:
	$(shell go test -coverprofile=coverage.out -json ./...\
		$$(go list ./... | \
			grep -v '/vendor/' \
		) > report.json)
	gosec --quiet -fmt sonarqube -out gosec.json -no-fail ./...
	sonar-scanner --debug || echo "Sonar scanning is not available at this time"

.PHONY: go-build
go-build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-a -tags netgo -o ./$(APP) \
		./cmd/watcher

