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

# CICD BUILD HARNESS
####################
GITHUB_USER := $(shell echo $(GITHUB_USER) | sed 's/@/%40/g')

.PHONY: default
default:: init;

.PHONY: init\:
init::
	@mkdir -p variables
ifndef GITHUB_USER
	$(info GITHUB_USER not defined)
	exit -1
endif
	$(info Using GITHUB_USER=$(GITHUB_USER))
ifndef GITHUB_TOKEN
	$(info GITHUB_TOKEN not defined)
	exit -1
endif

-include $(shell curl -fso .build-harness -H "Authorization: token ${GITHUB_TOKEN}" -H "Accept: application/vnd.github.v3.raw" "https://raw.github.ibm.com/ICP-DevOps/build-harness/master/templates/Makefile.build-harness"; echo .build-harness)
####################

.PHONY: dependencies
dependencies:
	go mod tidy
	go mod vendor

.PHONY: lint test
lint:
	@echo "Linting disabled."

test:
	@echo "Testing disabled."


.PHONY: go-build
go-build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-a -tags netgo -o ./$(APP)_$(GOARCH) \
		./cmd/watcher

.PHONY: docker-image
docker-image:
	$(eval DOCKER_BUILD_OPTS := '--build-arg "VCS_REF=$(GIT_COMMIT)" \
           --build-arg "VCS_URL=$(GIT_REMOTE_URL)" \
           --build-arg "IMAGE_NAME=$(DOCKER_IMAGE)" \
           --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" \
		   --build-arg "SUMMARY=$(SUMMARY)" \
		   --build-arg "GOARCH=$(GOARCH)"')
	@make DOCKER_BUILD_OPTS=$(DOCKER_BUILD_OPTS) docker:build
	@make DOCKER_URI=$(DOCKER_URI)-$(GIT_COMMIT) docker:tag

.PHONY: docker-push
# Push the docker image
docker-push:
ifneq ($(RETAG),)
	@make docker:tag
	@make docker:push
	@echo "Retagged image as $(DOCKER_URI) and pushed to $(DOCKER_REGISTRY)"
else
	@make DOCKER_URI=$(DOCKER_URI)-$(GIT_COMMIT) docker:push
endif

.PHONY: va-scan
va-scan:
ifeq ($(RETAG),)
	@make VASCAN_DOCKER_URI=$(DOCKER_URI)-$(GIT_COMMIT) vascan:image
endif


.PHONY: docker-push-rhel
docker-push-rhel:
ifneq ($(RETAG),)
	@make DOCKER_URI=$(DOCKER_URI)-rhel docker:tag
	@make DOCKER_URI=$(DOCKER_URI)-rhel docker:push
	@echo "Retagged image as $(DOCKER_URI)-rhel and pushed to $(DOCKER_REGISTRY)"
else
	@make DOCKER_URI=$(DOCKER_URI)-$(GIT_COMMIT)-rhel docker:tag
	@make DOCKER_URI=$(DOCKER_URI)-$(GIT_COMMIT)-rhel docker:push
endif
