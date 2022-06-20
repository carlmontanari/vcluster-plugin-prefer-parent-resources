.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

lint: ## Run linters
	gofumpt -w .
	goimports -w .
	golines -w .
	golangci-lint run

test: ## Run unit tests
	gotestsum --format testname --hide-summary=skipped -- -coverprofile=cover.out ./hooks/...

cov:  ## Produce html coverage report
	go tool cover -html=cover.out

connect-vcluster: ## connect to vcluster and pop out KUBECONFIG
	vcluster connect my-vcluster -n my-vcluster --kube-config="./kubeconfig.yaml" --update-current=false