# Configuration repository

GitLab Kubernetes integration supports hosting configuration for multiple GitLab Kubernetes Agents in a single repository. These agents may be running in the same or in multiple clusters, with potentially more than one agent per cluster.

## Why configuration repository?

Agent is bootstrapped with two pieces of information:

- GitLab installation URL
- Authentication token

There are two alternative approaches to provide the rest of configuration:

- As a `ConfigMap` or as part of some other Kubernetes object (e.g. environment variables in a `Deployment`)
- In a Git repository

We have chosen the Git repository approach because:

- Infrastructure as Code is a best practice and the user would have put the Kubernetes object with configuration under version control anyway
- Automatically pulling and applying configuration from the repository saves a hassle for the user

## Layout

Minimal repository layout looks like this:

```plaintext
|- .gitlab
    |- agents
       |- my_agent_1
          |- config.yaml
```

`my_agent_1` is the name (identity) of the agent. See [Agent identity and name](identity_and_auth.md#agent-identity-and-name) to find out more about names.

## `config.yaml` syntax

### `include` directive (not implemented)

Agents likely have different configuration, but some of it may be identical. `config.yaml` supports inclusion syntax similar to `.gitlab-ci.yml` [`include` directive](https://docs.gitlab.com/ee/ci/yaml/#include). Only `include: 'some_file_name.yml'` syntax is supported at the moment.

Example repository layout:

```plaintext
|- .gitlab
    |- base_for_agents
    |  |- config.yaml
    |- agents
       |- my_agent_1
       |  |- config.yaml
       |- production-agent
          |- config.yaml
```

`config.yaml` for both agents can include the `../../base_for_agents/config.yaml` file in such layout.

### `gitops` section

#### `manifest_projects` section

`manifest_projects` is a list of manifest projects, each of which is a Git repository with Kubernetes resource definitions in YAML or JSON format. Project can be specified using the `id` field.

```yaml
gitops:
  # Manifest projects are watched by the agent. Whenever a project changes, GitLab deploys the changes using the agent.
  manifest_projects:
    # No authentication mechanisms are currently supported.
  - id: gitlab-org/cluster-integration/gitlab-agent
    # Holds the only api groups and kinds of resources that gitops will monitor.
    # Inclusion rules are evaluated first, then exclusion rules. If there is still no match,
    # resource is monitored.
    resource_inclusions:
    - api_groups:
      - apps
      kinds:
      - '*'
    - api_groups:
      - ''
      kinds:
      - 'ConfigMap'
    # Holds the api groups and kinds of resources to exclude from gitops watch.
    # Inclusion rules are evaluated first, then exclusion rules. If there is still no match,
    # resource is monitored.
    resource_exclusions:
    - api_groups:
      - '*'
      kinds:
      - '*'
    # Namespace to use if not set explicitly in object manifest.
    default_namespace: my-ns
    # Paths inside of the repository to scan for manifest files. Directories with names starting with a dot are ignored.
    paths:
      # Read all .yaml files from team1/app1 directory.
      # See https://github.com/bmatcuk/doublestar#about and
      # https://pkg.go.dev/github.com/bmatcuk/doublestar/v2#Match for globbing rules.
    - glob: '/team1/app1/*.yaml'
      # Read all .yaml files from team2/apps and all subdirectories
    - glob: '/team2/apps/**/*.yaml'
      # If 'paths' is not specified or is an empty list, the configuration below is used
    - glob: '/**/*.{yaml,yml,json}'
```

By default, all resource kinds are monitored. Use `resource_exclusions` section to specify exclusion patterns to narrow down the list of monitored resources. This allows to reduce the needed permissions for the GitOps feature. To invert the matching behavior, exclude all groups/kinds and use `resource_inclusions` to specify the desired resource patterns. See the example configuration above for this pattern.
