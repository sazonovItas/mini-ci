# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: run
run: workflower
	./bin/workflower

.PHONY: workflower
workflower:
	go build -o bin/workflower ./cmd/workflower

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## RUN go vet against code
	go vet ./...

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
		mkdir -p $(LOCALBIN)

## Tool Binaries
SQLC ?= $(LOCALBIN)/sqlc

## Tool Versions
SQLC_VERSION ?= v1.30.0

.PHONY: sqlc-gen
sqlc-gen: sqlc
	$(SQLC) generate

.PHONY: sqlc
sqlc: $(SQLC)
$(SQLC): $(LOCALBIN)
	$(call go-install-tool,$(SQLC),github.com/sqlc-dev/sqlc/cmd/sqlc,$(SQLC_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $$(realpath $(1)-$(3)) $(1)
endef
