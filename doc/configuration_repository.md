# Layout of configuration repository

GitLab Kubernetes integration supports hosting configuration for multiple GitLab Kubernetes Agents in a single repository. These agents may be running in the same or in multiple clusters, with potentially more than one agent per cluster.

Minimal repository layout looks like this:

```
|- agents
   |- my_agent_1
      |- config.yaml
```

`my_agent_1` is the name (identity) of the agent. It's unique in this project. Names are immutable strings, provided by the user. Agent names can only contain `a-z0-9-_` characters and be up to 64 characters long.

## `config.yaml` syntax

### `include`

Agents likely have different configuration, but some of it may be identical. `config.yaml` supports inclusion syntax similar to `.gitlab-ci.yml` [`include` directive](https://docs.gitlab.com/ee/ci/yaml/#include). Only `include: 'some_file_name.yml'` syntax is supported at the moment.

Example repository layout:

```
|- base
|  |- config.yaml
|- agents
   |- my_agent_1
   |  |- config.yaml
   |- production-agent
      |- config.yaml
```

`config.yaml` for both agents can include the `../../base/config.yaml` file in such layout.
