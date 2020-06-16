#!/usr/bin/env bash

set -e -o pipefail

cat << 'EOF' > build/repositories.bzl
def go_repositories():
   pass
EOF
