ifdef GITALY_ADDRESS
	GITALY_ADDRESS_ARG := --test_env=GITALY_ADDRESS=$(GITALY_ADDRESS)
else
	GITALY_ADDRESS_ARG :=
endif
ifdef GITLAB_ADDRESS
	GITLAB_ADDRESS_ARG := --test_env=GITLAB_ADDRESS=$(GITLAB_ADDRESS)
else
	GITLAB_ADDRESS_ARG :=
endif

.PHONY: fmt-bazel
fmt-bazel:
	bazel run //:buildozer
	bazel run //:buildifier

.PHONY: regenerate-grpc-internal
regenerate-grpc-internal:
	bazel run //pkg/agentrpc:extract_agent_grpc

.PHONY: regenerate-grpc
regenerate-grpc: regenerate-grpc-internal fmt update-bazel

.PHONY: update-repos
update-repos:
	go mod tidy
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
	bazel test $(GITALY_ADDRESS_ARG) $(GITLAB_ADDRESS_ARG) \
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
