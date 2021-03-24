#!/usr/bin/env bash

# This command is used by bazel as the workspace_status_command
# to implement build stamping with git information.

set -o errexit
set -o nounset
set -o pipefail

GIT_COMMIT=$(git rev-parse --short HEAD)
GIT_TAG=$(git tag --points-at HEAD 2>/dev/null || true)
GIT_TAG="${GIT_TAG:="latest"}"

CONTAINER_REPOSITORY_PATH="${CI_PROJECT_PATH:-"gitlab-org/cluster-integration/gitlab-agent"}"
BUILD_TIME=$(date -u +%Y%m%d.%H%M%S)
# Prefix with STABLE_ so that these values are saved to stable-status.txt
# instead of volatile-status.txt.
# Stamped rules will be retriggered by changes to stable-status.txt, but not by
# changes to volatile-status.txt.
# See https://docs.bazel.build/versions/master/user-manual.html#flag--workspace_status_command
cat <<EOF
STABLE_BUILD_GIT_COMMIT ${GIT_COMMIT-}
STABLE_BUILD_GIT_TAG ${GIT_TAG-}
STABLE_CONTAINER_REPOSITORY_PATH ${CONTAINER_REPOSITORY_PATH-}
BUILD_TIME ${BUILD_TIME-}
EOF
