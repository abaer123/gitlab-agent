#!/usr/bin/env bash

set -e -o pipefail

# This is a helper script that allows copying files, generated by a bazel rule, into a directory, specified
# by a workspace root relative path.

if (( $# != 3 )); then
  echo 'Not enough or too many command line arguments' >&2
  exit 1
fi

source_files="$1"
file_to_copy="$2"
workspace_relative_target_directory="$3"

# Don't want to double quote because $(locations //label) in build.bzl expands into a single argument which
# contains space-separated file names.
# shellcheck disable=SC2068
for file in $source_files
do
  name=$(basename "$file")
  if [[ "$name" == "$file_to_copy" ]]
  then
    to="$BUILD_WORKSPACE_DIRECTORY/$workspace_relative_target_directory/$name"
    cp "$BUILD_WORKSPACE_DIRECTORY/$file" "$to"
    chmod +w "$to"
    break
  fi
done