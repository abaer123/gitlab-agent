# Development guide

## Repository overview

You can watch the video below to understand how this repository is
structured:

[![GitLab Kubernetes Agent repository overview](http://img.youtube.com/vi/j8CyaCWroUY/0.jpg)](http://www.youtube.com/watch?v=j8CyaCWroUY "GitLab Kubernetes Agent repository overview")

## Running the agent locally

The easiest way to test the agent locally is to run `kas` and `agentk` yourself.

First you need to setup two files:

1. `cfg.yaml`.
   For reference, here's an example file:

   ```yaml
   agent:
     listen:
       network: tcp
       address: 127.0.0.1:8150
       websocket: false
   gitlab:
     address: http://localhost:3000
     authentication_secret_file: /Users/tkuah/code/ee-gdk/gitlab/.gitlab_kas_secret
   agent:
     gitops:
       poll_period: "10s"
   ```

1. `token.txt`. This is the token for the agent you [created](https://docs.gitlab.com/ee/user/clusters/agent/#create-an-agent-record-in-gitlab).
   Note the file must **not** have a newline. The simplest way to achieve this is with:

   ```sh
   echo -n "<TOKEN>" > token.txt
   ```

You can then start the binaries with:

```sh
# Need GitLab to start :)
gdk start
# Stop GDK's version of kas
gdk stop gitlab-k8s-agent

# Start kas
bazel run //cmd/kas -- --configuration-file="$(pwd)/cfg.yaml"

# Start agentk
bazel run //cmd/agentk -- --kas-address=grpc://127.0.0.1:8150 --token-file="$(pwd)/token.txt"
```

You can also inspect the [Makefile](Makefile) for more targets.
