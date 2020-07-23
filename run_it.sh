#!/usr/bin/env bash

# Update the variables below to run the integration tests locally.
# Start GDK before running tests.
# Currently these tests are not self-contained, some stuff is hardcoded. Some stuff needs to be hardcoded in GitLab too.
# It will become better over time.

export GITALY_ADDRESS="unix:/Users/mikhail/src/gitlab-development-kit/praefect.socket"
export GITLAB_ADDRESS="http://127.0.0.1:3000"
export KAS_TOKEN="5cJvh6M9652dsYQeZz7H"
export KUBECONFIG=/Users/mikhail/.kube/config
export KUBECONTEXT=kind-kind
export TEST_LOG_FORMATTER=color

exec make test-it
