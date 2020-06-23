#!/usr/bin/env bash

set -e -o pipefail

target_dir="$1"
file_to_copy="$2"
shift 2

for i in "$@"
do
  name=$(basename "$i")
  if [[ "$name" == "$file_to_copy" ]]
  then
    to="${BUILD_WORKSPACE_DIRECTORY}/$target_dir/$name"
    cp "$i" "$to"
    chmod +w "$to"
    break
  fi
done
