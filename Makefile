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

.PHONY: fmt-bazel
fmt-bazel:
	bazel run //:buildozer
	bazel run //:buildifier

.PHONY: internal-regenerate-proto
internal-regenerate-proto:
	bazel run //build:extract_generated_proto

.PHONY: regenerate-proto
regenerate-proto: internal-regenerate-proto fmt update-bazel

.PHONY: internal-regenerate-mocks
internal-regenerate-mocks:
	go generate -x -v \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc/mock_agentrpc" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab/mock_gitlab" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly/mock_gitalypool" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_engine" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_gitaly" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/testing/mock_misc"

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
update-bazel:
	bazel run //:gazelle

.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports -w cmd it internal pkg

.PHONY: test
test: fmt update-bazel test-ci

.PHONY: test-ci
test-ci:
	bazel test \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
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
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		--test_output=all \
		--test_arg=-test.v \
		-- $$(bazel query 'attr(tags, manual, kind(test, //it/...))')

.PHONY: quick-test
quick-test:
	bazel test \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		--build_tests_only \
		-- //...

.PHONY: docker
docker: update-bazel
	bazel build \
		//cmd/agentk:container \
		//cmd/kas:container

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
	# Build all targets in a single invocation for maximum parallelism
	bazel build \
		//cmd/agentk:push_docker_tag \
		//cmd/agentk:push_docker_tag_race \
		//cmd/kas:push_docker_tag \
		//cmd/kas:push_docker_tag_race
	# Actually push built images one by one
	bazel run \
		//cmd/agentk:push_docker_tag
	bazel run \
		//cmd/agentk:push_docker_tag_race
	bazel run \
		//cmd/kas:push_docker_tag
	bazel run \
		//cmd/kas:push_docker_tag_race

# Build and push all docker images tagged with the current commit sha.
# This only works on a linux machine
.PHONY: release-commit-all-ci
release-commit-all-ci:
	# Build all targets in a single invocation for maximum parallelism
	bazel build \
		//cmd/agentk:push_docker_commit \
		//cmd/agentk:push_docker_commit_race \
		//cmd/kas:push_docker_commit \
		//cmd/kas:push_docker_commit_race
	# Actually push built images one by one
	bazel run \
		//cmd/agentk:push_docker_commit
	bazel run \
		//cmd/agentk:push_docker_commit_race
	bazel run \
		//cmd/kas:push_docker_commit
	bazel run \
		//cmd/kas:push_docker_commit_race


# Build and push all docker images tagged "latest".
# This only works on a linux machine
.PHONY: release-latest-all-ci
release-latest-all-ci:
	# Build all targets in a single invocation for maximum parallelism
	bazel build \
		//cmd/agentk:push_docker_latest \
		//cmd/agentk:push_docker_latest_race \
		//cmd/kas:push_docker_latest \
		//cmd/kas:push_docker_latest_race
	# Actually push built images one by one
	bazel run \
		//cmd/agentk:push_docker_latest
	bazel run \
		//cmd/agentk:push_docker_latest_race
	bazel run \
		//cmd/kas:push_docker_latest
	bazel run \
		//cmd/kas:push_docker_latest_race

.PHONY: release-commit-normal
release-commit-normal: update-bazel
	bazel run \
		//cmd/agentk:push_docker_commit
	bazel run \
		//cmd/kas:push_docker_commit

# This only works on a linux machine
.PHONY: release-commit-race
release-commit-race: update-bazel
	bazel run \
		//cmd/agentk:push_docker_commit_race
	bazel run \
		//cmd/kas:push_docker_commit_race

# Set TARGET_DIRECTORY variable to the target directory before running this target
.PHONY: gdk-install
gdk-install:
	bazel run //build:extract_race_binaries_for_gdk -- "$(TARGET_DIRECTORY)"

# Set TARGET_DIRECTORY variable to the target directory before running this target
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG = $(shell git tag --points-at HEAD 2>/dev/null || true)
ifeq ($(GIT_TAG), )
	GIT_TAG = "v0.0.0"
endif
.PHONY: kas
kas:
	go build \
		-tags "${GO_BUILD_TAGS}" \
		-ldflags "-X gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Version=$(GIT_TAG) -X gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Commit=$(GIT_COMMIT)" \
		-o "$(TARGET_DIRECTORY)" ./cmd/kas

# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
.PHONY: show-go-dependency-updates
show-go-dependency-updates:
	go list \
		-tags "${GO_BUILD_TAGS}" \
		-u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all 2> /dev/null
