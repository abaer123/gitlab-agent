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
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentrpc/mock_agentrpc" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/mock_engine" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/tools/testing/mock_gitaly" \
		"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/gitlab/mock_gitlab"

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
	go run golang.org/x/tools/cmd/goimports -w cmd it pkg

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
		--test_env=KGB_TOKEN=$(KGB_TOKEN) \
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
		//cmd/kgb:container

# This only works from a linux machine
.PHONY: docker-race
docker-race: update-bazel
	bazel build \
		//cmd/agentk:container_race \
		//cmd/kgb:container_race

# Export docker image into local Docker
.PHONY: docker-export
docker-export: update-bazel
	bazel run \
		//cmd/agentk:container \
		-- \
		--norun
	bazel run \
		//cmd/kgb:container \
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
		//cmd/kgb:container_race \
		-- \
		--norun

# Build and push all docker images tagged with the tag on the current commit.
# This only works on a linux machine
.PHONY: release-tag-all
release-tag-all: update-bazel
	# Build all targets in a single invocation for maximum parallelism
	bazel build \
		//cmd/agentk:push_docker_tag \
		//cmd/agentk:push_docker_tag_race \
		//cmd/kgb:push_docker_tag \
		//cmd/kgb:push_docker_tag_race
	# Actually push built images one by one
	bazel run \
		//cmd/agentk:push_docker_tag
	bazel run \
		//cmd/agentk:push_docker_tag_race
	bazel run \
		//cmd/kgb:push_docker_tag
	bazel run \
		//cmd/kgb:push_docker_tag_race

# Build and push all docker images tagged with the current commit sha.
# This only works on a linux machine
.PHONY: release-commit-all
release-commit-all: update-bazel
	# Build all targets in a single invocation for maximum parallelism
	bazel build \
		//cmd/agentk:push_docker_commit \
		//cmd/agentk:push_docker_commit_race \
		//cmd/kgb:push_docker_commit \
		//cmd/kgb:push_docker_commit_race
	# Actually push built images one by one
	bazel run \
		//cmd/agentk:push_docker_commit
	bazel run \
		//cmd/agentk:push_docker_commit_race
	bazel run \
		//cmd/kgb:push_docker_commit
	bazel run \
		//cmd/kgb:push_docker_commit_race

.PHONY: release-commit-normal
release-commit-normal: update-bazel
	bazel run \
		//cmd/agentk:push_docker_commit
	bazel run \
		//cmd/kgb:push_docker_commit

# This only works on a linux machine
.PHONY: release-commit-race
release-commit-race: update-bazel
	bazel run \
		//cmd/agentk:push_docker_commit_race
	bazel run \
		//cmd/kgb:push_docker_commit_race

# Set TARGET_DIRECTORY variable to the target directory before running this target
.PHONY: gdk-install
gdk-install:
	bazel run //build:extract_race_binaries_for_gdk -- "$(TARGET_DIRECTORY)"
