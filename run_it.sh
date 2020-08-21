#!/usr/bin/env bash

# Update the variables below to run the integration tests locally.
# Start GDK before running tests.
# Currently these tests are not self-contained, some stuff is hardcoded.
# It will become better over time.

export GITLAB_ADDRESS="http://127.0.0.1:3000"
export AGENTK_TOKEN="5cJvh6M9652dsYQeZz7H"
export KAS_GITLAB_AUTH_SECRET='/Users/mikhail/src/gitlab-development-kit/gitlab/.gitlab_kas_secret'
export KUBECONFIG="$HOME/.kube/config"
export KUBECONTEXT=kind-kind
export TEST_LOG_FORMATTER=color

exec make test-it
