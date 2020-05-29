#!/usr/bin/env bash

export GITALY_ADDRESS="unix:/Users/mikhail/src/gitlab-development-kit/praefect.socket"
export GITLAB_ADDRESS="http://127.0.0.1:3000"

make test-it
