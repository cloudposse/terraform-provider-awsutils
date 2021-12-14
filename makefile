TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=registry.terraform.io
NAMESPACE=cloudposse
NAME=awsutils
BINARY=terraform-provider-${NAME}
VERSION=9999.99.99
OS_ARCH=darwin_amd64
SHELL := /bin/bash
CURRENT_DIR = $(shell pwd)
USER_ID = $(shell id -u)
GROUP_ID = $(shell id -g)

# List of targets the `readme` target should call before generating the readme
export README_DEPS ?= docs/targets.md docs/terraform.md

-include $(shell curl -sSL -o .build-harness "https://git.io/build-harness"; echo .build-harness)

build:
	go build

deps:
	go mod download

generate:
	go generate

docs: tfdocs readme

tfdocs:
	tfplugindocs

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

# Lint terraform code
lint:
	$(SELF) terraform/install terraform/get-modules terraform/get-plugins terraform/lint terraform/validate

docker:
	docker run --rm -it -w ${CURRENT_DIR} \
		-v ${CURRENT_DIR}:${CURRENT_DIR} \
		-v ~/.cache:/.cache \
		-v ~/.terraform.d:/.terraform.d \
		-u ${USER_ID}:${GROUP_ID} \
		golang \
		make $(filter-out $@,$(MAKECMDGOALS))

# Run acceptance tests
testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m