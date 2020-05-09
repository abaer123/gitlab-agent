#!/usr/bin/env bash

set -e -o pipefail

cp "$1" "${BUILD_WORKSPACE_DIRECTORY}/$2"
n=$(basename "$1")
chmod +w "${BUILD_WORKSPACE_DIRECTORY}/$2/$n"
