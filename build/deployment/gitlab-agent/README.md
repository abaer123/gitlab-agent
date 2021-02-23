# GitLab Kubernetes Agent

## Description

Package for installing GitLab Kubernetes Agent.

## Prerequisites

- [Kustomize](https://kustomize.io/) version 3.8 or newer.
- *Optional, but recommended* - [`kpt`](https://googlecontainertools.github.io/kpt/) version 0.32.0 or newer.

## Configuration

GitLab Kubernetes Agent needs two pieces of configuration to connect to a GitLab instance:

1. URL. The agent can use WebSockets or gRPC protocols to connect to GitLab. Depending
   on how your GitLab instance is configured, you may need to use one or the other.

   - Specify `grpc` scheme (e.g. `grpc://127.0.0.1:8150`) to use gRPC directly. The connection is not encrypted.
   - Encrypted gRPC is not supported yet. See the issue
     [Support TLS for gRPC communication](https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent/-/issues/7).
   - Specify `ws` scheme to use WebSocket connection. The connection is not encrypted.
   - Specify `wss` scheme to use an encrypted WebSocket connection.

1. Token.

## Use the package

These instructions use Kustomize and are applicable regardless of whether or not
you use `kpt`, but `kpt` makes cloning and updating the package more convenient.

1. Get the package, using the method of your choice.

   - *If you are using `kpt`,* read the
     [`kpt pkg get` documentation](https://googlecontainertools.github.io/kpt/guides/consumer/get/)
     for more information on how the following command works:

      ```shell
      kpt pkg get https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent.git/build/deployment/gitlab-agent gitlab-agent
      ```

   - *If you are not using `kpt`,* you can use contents of this directory directly
     by cloning the repository with the following commands:

     ```shell
     git clone https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent.git
     cd gitlab-agent/build/deployment/gitlab-agent
     ```

1. (Optional) [Make a Kustomize overlay](https://kubernetes-sigs.github.io/kustomize/guides/offtheshelf/)
   with any desired customizations.

1. (Optional) Use Kustomize setters to alter recommended configuration knobs:

    ```shell
    # in gitlab-agent directory
    kustomize cfg list-setters .

        NAME                VALUE               SET BY                  DESCRIPTION              COUNT   REQUIRED   IS SET
    agent-version   latest                  package-default   Image tag for agentk container     1       No         No
    kas-address     grpc://127.0.0.1:8150   package-default   kas address. Use                   1       No         No
                                                              grpc://host.docker.internal:8150
                                                              if connecting from within Docker
                                                              e.g. from kind.
    namespace       gitlab-agent            package-default   Namespace to install GitLab        1       No         No
                                                              Kubernetes Agent into

    kustomize cfg set . namespace custom-place
    set 1 fields
    kustomize cfg set . kas-address wss://my-host.example.com:443/-/kubernetes-agent
    set 1 fields
    ```

1. (Optional but recommended) Commit the configuration into a repository and manage it as code.

1. Prior to deployment, write the agent token you obtained via registration in GitLab to `base/secrets/agent.token`.
   Note that this should not be committed to a repository and is excluded by the `.gitignore` in the `base/secrets`
   directory by default.

    ```shell
    echo -n "<agent token>" | base64 > base/secrets/agent.token
    ```

1. Deploy the stock configuration or your customized overlay:

    ```shell
    # in the package directory
    kustomize build cluster | kubectl apply -f -
    # kustomize build my-custom-overlay | kubectl apply -f -
    ```

Later, you can pull in package updates using
[`kpt pkg update`](https://googlecontainertools.github.io/kpt/guides/consumer/update/):

```shell
kpt pkg update gitlab-agent --strategy resource-merge
```
