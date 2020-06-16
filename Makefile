.PHONY: fmt-bazel
fmt-bazel:
	bazel run //:buildozer
	bazel run //:buildifier

.PHONY: regenerate-proto-internal
regenerate-proto-internal:
	bazel run //pkg/agentcfg:extract_proto
	bazel run //pkg/agentrpc:extract_agent_grpc

.PHONY: regenerate-proto
regenerate-proto: regenerate-proto-internal fmt update-bazel

.PHONY: update-repos
update-repos:
	go mod tidy
	./build/update-repos.sh
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

.PHONY: release
release: update-bazel
	bazel run \
		//cmd/agentk:push_docker
	bazel run \
		//cmd/kgb:push_docker

# This only works on a linux machine
.PHONY: release-race
release-race: update-bazel
	bazel run \
		//cmd/agentk:push_docker_race
	bazel run \
		//cmd/kgb:push_docker_race
