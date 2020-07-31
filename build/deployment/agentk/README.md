agentk
==================================================

# Description

Package for installing `agentk`, the in-cluster component of GitLab Kubernetes Agent.

# Use the package

1. Get the package using [`kpt`](https://googlecontainertools.github.io/kpt/). See [`kpt pkg get` docs](https://googlecontainertools.github.io/kpt/guides/consumer/get/) for more information on how the following command works:

    ```shell
    kpt pkg get https://gitlab.com/gitlab-org/cluster-integration/gitlab-agent.git/build/deployment/agentk agentk
    ```

    It's possible to use this package without `kpt` - clone or fork this repository and use this directory with Kustomize, as described below. `kpt` merely makes cloning and updating the package more convenient.

1. (Optional) [Make a Kustomize overlay](https://kubernetes-sigs.github.io/kustomize/guides/offtheshelf/) with any desired customizations.

1. (Optional) Use Kustomize setters to alter recommended configuration knobs:

    ```shell
    # in agentk directory
    kustomize cfg list-setters .
     NAME              VALUE               SET BY                  DESCRIPTION             COUNT
    kas-address   tcp://127.0.0.1:5005                     kas address. Use                  1
                                                         tcp://host.docker.internal:5005
                                                         if connecting from within
                                                         Docker e.g. from kind.
    namespace     agentk                 package-default   Namespace to install agentk       1
                                                         into

    kustomize cfg set . namespace custom-place
    set 1 fields
    kustomize cfg set . kas-address wss://my-host.example.com:443
    set 1 fields
    ```

1. (Optional but recommended) Commit the configuration into a repository and manage it as code.

1. Put the agent token into the file where Kustomize [`SecretGenerator`](https://kubernetes-sigs.github.io/kustomize/guides/plugins/builtins/#_secretgenerator_) plugin is expecting it:

    ```shell
    # in agentk directory
    echo -n '<Agent token>' > base/token.txt
    ```

1. Deploy the stock configuration or your customized overlay:

    ```shell
    # in agentk directory
    kustomize build cluster | kubectl apply -f -
    # kustomize build my-custom-overlay | kubectl apply -f -
    ```

1. (Later) Pull in updates for the `agentk` package using [`kpt pkg update`](https://googlecontainertools.github.io/kpt/guides/consumer/update/):

    ```shell
    kpt pkg update agentk --strategy resource-merge
    ```
