BUILD_DIR := build
PROVIDER_NAME := system-v1alpha1
TF_PROVIDER_NAME := terraform-provider-${PROVIDER_NAME}
TERRAFORMRC := "${HOME}/.terraformrc"
TF_RC_DEV_KEY := "nokia-eda/${PROVIDER_NAME}"
TFPLUGINDOCS_VERSION := v0.22.0

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts"
	@rm -rf ${BUILD_DIR}

.PHONY: fmt
fmt: ## Run go fmt and terraform fmt against code.
	go fmt ./...
	terraform fmt -recursive examples

.PHONY: vet
vet: ## Run go vet against code.
	@go mod tidy
	go vet ./...

.PHONY: build-dir
build-dir:
	@mkdir -p ${BUILD_DIR}

.PHONY: build
build: build-dir fmt vet ## Build the terraform provider
	@echo "Building ${TF_PROVIDER_NAME}"
	go build -ldflags="-s -w" -o ${BUILD_DIR}/${TF_PROVIDER_NAME} main.go

.PHONY: tfplugindocs
tfplugindocs: $(LOCALBIN) ## Download tfplugindocs binary into bin
	$(call go-install-tool,$(LOCALBIN)/tfplugindocs,github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs,$(TFPLUGINDOCS_VERSION))

.PHONY: gen-docs
gen-docs: tfplugindocs ## Generate docs using local tfplugindocs binary.
	@echo "Generating documentation"
	@$(LOCALBIN)/tfplugindocs generate --provider-dir . --provider-name ${PROVIDER_NAME}

##@ Deployment

.PHONY: install
install: build ## Install the terraform provider
	@echo "Installing ${TF_PROVIDER_NAME} dev override"
	@export TERRAFORMRC=${TERRAFORMRC} KEY=${TF_RC_DEV_KEY} BUILD_PATH="$$(realpath ${BUILD_DIR})"; \
	[ -f "$${TERRAFORMRC}" ] || { echo "Creating $${TERRAFORMRC}"; echo 'provider_installation {\n  dev_overrides {\n  }\n  direct {}\n}' > "$${TERRAFORMRC}"; }; \
	if grep -q "$${KEY}" "$${TERRAFORMRC}"; then \
		echo "Key $${KEY} already present in $${TERRAFORMRC}"; \
	else \
		awk -v key="\"$${KEY}\"" -v value="\"$${BUILD_PATH}\"" '\
		BEGIN { in_block=0 } \
		/dev_overrides[[:space:]]*{/ { in_block=1 } \
		in_block && /}/ { print "      " key " = " value; in_block=0 } \
		{ print }' "$${TERRAFORMRC}" > "$${TERRAFORMRC}.tmp" && mv "$${TERRAFORMRC}.tmp" "$${TERRAFORMRC}"; \
		echo "Added key $${KEY} to dev_overrides block in $${TERRAFORMRC} pointing to $${BUILD_PATH}"; \
	fi

.PHONY: uninstall
uninstall: ## Uninstall the terraform provider
	@echo "Uninstalling ${TF_PROVIDER_NAME} dev override"
	@rm -rf ${BUILD_DIR}
	@export TERRAFORMRC=${TERRAFORMRC} KEY=${TF_RC_DEV_KEY}; \
	awk -v key="$${KEY}" '$$0 ~ key { next } { print }' "$${TERRAFORMRC}" > "$${TERRAFORMRC}.tmp" && mv "$${TERRAFORMRC}.tmp" "$${TERRAFORMRC}"; \
	echo "Removed key $${KEY} from dev_overrides block in $${TERRAFORMRC};"
