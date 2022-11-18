.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

ifdef PLUGIN_REPOSITORY
REPO := $(PLUGIN_REPOSITORY)/
else
REPO := "X"
endif


lint: ## Run linters
	gofumpt -w .
	goimports -w .
	golines -w .
	golangci-lint run

test: ## Run unit tests
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=cover.out ./prefer-parent-resources/...

test-race: ## Run unit tests with race flag
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=cover.out ./... -race

cov:  ## Produce html coverage report
	go tool cover -html=cover.out

connect-vcluster: ## connect to vcluster and pop out KUBECONFIG
	vcluster connect my-vcluster -n my-vcluster --kube-config="./kubeconfig.yaml" --update-current=false &

build-image: ## build docker image, set PLUGIN_REPOSITORY envvar to point to pushable repo if not developing docker-desktop or similar
	docker build . -t $(REPO)prefer-parent-resources-hooks
	sed -i ".bak" -r "s|(image:).*|\1 "${REPO}"prefer-parent-resources-hooks|" plugin.yaml
	rm plugin.yaml.bak