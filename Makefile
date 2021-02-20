# The same list of go build tags must be in three places:
# - Makefile
# - Workspace
# - .bazelrc
GO_BUILD_TAGS := tracer_static,tracer_static_jaeger

# Install using your package manager, as recommended by
# https://golangci-lint.run/usage/install/#local-installation
.PHONY: lint
lint:
	golangci-lint run

.PHONY: buildozer
buildozer:
	bazel run //:buildozer

.PHONY: buildifier
buildifier:
	bazel run //:buildifier

.PHONY: fmt-bazel
fmt-bazel: gazelle buildozer buildifier

.PHONY: gazelle
gazelle:
	bazel run //:gazelle

.PHONY: internal-regenerate-proto
internal-regenerate-proto:
	bazel run //build:extract_generated_proto

.PHONY: regenerate-proto
regenerate-proto: internal-regenerate-proto fmt update-bazel

.PHONY: internal-regenerate-mocks
internal-regenerate-mocks:
	go generate -x -v \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/gitops/agent" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel_rpc" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_reverse_tunnel_tracker" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_agent_tracker" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_usage_metrics" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_internalgitaly" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modserver" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_redis" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitaly" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_rpc" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_errtracker" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/cilium_alert/agent"

.PHONY: regenerate-mocks
regenerate-mocks: internal-regenerate-mocks fmt update-bazel

.PHONY: update-repos
update-repos:
	go mod tidy
	./build/update_repos.sh
	bazel run \
		//:gazelle -- \
		update-repos \
		-from_file=go.mod \
		-prune=true \
		-build_file_proto_mode=disable_global \
		-to_macro=build/repositories.bzl%go_repositories

.PHONY: update-bazel
update-bazel: gazelle

.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports -w cmd it internal pkg

.PHONY: test
test: fmt update-bazel test-ci

.PHONY: test-ci
test-ci:
	bazel test \
		-- //...
	bazel build $$(bazel query 'attr(tags, manual, kind(test, //it/...))')

.PHONY: test-it
test-it: fmt update-bazel
	bazel test \
		--test_env=GITALY_ADDRESS=$(GITALY_ADDRESS) \
		--test_env=GITLAB_ADDRESS=$(GITLAB_ADDRESS) \
		--test_env=AGENTK_TOKEN=$(AGENTK_TOKEN) \
		--test_env=KAS_GITLAB_AUTH_SECRET=$(KAS_GITLAB_AUTH_SECRET) \
		--test_env=KUBECONFIG=$(KUBECONFIG) \
		--test_env=KUBECONTEXT=$(KUBECONTEXT) \
		--test_env=TEST_LOG_FORMATTER=$(TEST_LOG_FORMATTER) \
		--test_output=all \
		--test_arg=-test.v \
		-- $$(bazel query 'attr(tags, manual, kind(test, //it/...))')

.PHONY: quick-test
quick-test:
	bazel test \
		--build_tests_only \
		-- //...

.PHONY: docker
docker: update-bazel
	bazel build \
		//cmd/agentk:container \
		//cmd/kas:container \
		//cmd/cli:container \

# This only works from a linux machine
.PHONY: docker-race
docker-race: update-bazel
	bazel build \
		//cmd/agentk:container_race \
		//cmd/kas:container_race

# Export docker image into local Docker
.PHONY: docker-export
docker-export: update-bazel
	bazel run \
		//cmd/agentk:container \
		-- \
		--norun
	bazel run \
		//cmd/kas:container \
		-- \
		--norun
	bazel run \
		//cmd/cli:container \
		-- \
		--norun

# Export docker image into local Docker
# This only works on a linux machine
.PHONY: docker-export-race
docker-export-race: update-bazel
	bazel run \
		//cmd/agentk:container_race \
		-- \
		--norun
	bazel run \
		//cmd/kas:container_race \
		-- \
		--norun

# Build and push all docker images tagged with the tag on the current commit.
# This only works on a linux machine
.PHONY: release-tag-all-ci
release-tag-all-ci:
	bazel run //cmd:release-tag-all

# Build and push all docker images tagged with the current commit sha.
# This only works on a linux machine
.PHONY: release-commit-all-ci
release-commit-all-ci:
	bazel run //cmd:release-commit-all

# Build and push all docker images tagged "latest".
# This only works on a linux machine
.PHONY: release-latest-all-ci
release-latest-all-ci:
	bazel run //cmd:release-latest-all

.PHONY: release-commit-normal
release-commit-normal: update-bazel
	bazel run //cmd:release-commit-normal

# This only works on a linux machine
.PHONY: release-commit-race
release-commit-race: update-bazel
	bazel run //cmd:release-commit-race

# Set TARGET_DIRECTORY variable to the target directory before running this target
.PHONY: gdk-install
gdk-install:
	bazel run //build:extract_race_binaries_for_gdk -- "$(TARGET_DIRECTORY)"

# Set TARGET_DIRECTORY variable to the target directory before running this target
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG = $(shell git tag --points-at HEAD 2>/dev/null || true)
BUILD_TIME = $(shell date -u +%Y%m%d.%H%M%S)
ifeq ($(GIT_TAG), )
	GIT_TAG = "v0.0.0"
endif
.PHONY: kas
kas:
	go build \
		-tags "${GO_BUILD_TAGS}" \
		-ldflags "-X gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Version=$(GIT_TAG) -X gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Commit=$(GIT_COMMIT) -X gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.BuildTime=$(BUILD_TIME)" \
		-o "$(TARGET_DIRECTORY)" ./cmd/kas

# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
.PHONY: show-go-dependency-updates
show-go-dependency-updates:
	go list \
		-tags "${GO_BUILD_TAGS}" \
		-u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null
