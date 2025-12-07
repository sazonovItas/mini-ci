BASE_IMAGE ?= worker:base
BASE_DOCKERFILE ?= Dockerfile.base

GO_VERSION ?= 1.25.0

.PHONY: build-base
build-base:
	@docker build --build-arg GO_VERSION=${GO_VERSION} \
		-t ${BASE_IMAGE} -f ${BASE_DOCKERFILE} .
