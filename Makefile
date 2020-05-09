.PHONY: fmt-bazel
fmt-bazel:
	bazel run //:buildozer
	bazel run //:buildifier

.PHONY: regenerate-grpc-internal
regenerate-grpc-internal:
	bazel run //proto/agentrpc:extract_agent_grpc

.PHONY: regenerate-grpc
regenerate-grpc: regenerate-grpc-internal update-bazel

.PHONY: update-repos
update-repos:
	bazel run \
		//:gazelle -- \
		update-repos \
		-from_file=go.mod \
		-to_macro=build/repositories.bzl%go_repositories

.PHONY: update-bazel
update-bazel:
	bazel run //:gazelle

.PHONY: fmt
fmt:
	go run golang.org/x/tools/cmd/goimports

.PHONY: test
test: fmt update-bazel test-ci

.PHONY: test-ci
test-ci:
	bazel test \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		-- //...

.PHONY: quick-test
quick-test:
	bazel test \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		--build_tests_only \
		-- //...
