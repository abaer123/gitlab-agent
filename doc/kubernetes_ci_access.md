# Access to Kubernetes from CI

## Problem to solve

As an Application Operator, I would like certain CI jobs to be able to access my Kubernetes cluster, connected
via GitLab Kubernetes Agent. That way I don't have to open up my cluster to access it from CI.

## Intended users

* [Allison (Application Ops)](https://about.gitlab.com/handbook/marketing/product-marketing/roles-personas/#allison-application-ops)
* [Priyanka (Platform Engineer)](https://about.gitlab.com/handbook/marketing/product-marketing/roles-personas/#priyanka-platform-engineer)

## User experience goal

The user can allow certain CI jobs to access Kubernetes clusters connected via GitLab Kubernetes Agent.

A single CI job can access multiple clusters, that is to access multiple Kubernetes Agents.
This is often required in production environments, where the production environment is composed of multiple clusters
in different regions/availability zones.

## Proposal

In the Kubernetes Agent's configuration file, managed as code in the configuration project, user specifies a list of
projects and groups, CI jobs from which can access this particular agent. CI jobs of the configuration project itself
can access all agents configured via this project (TODO security review).

```yaml
# .gitlab/agents/my-agent/config.yaml
ci_access:
  # This agent is accessible from CI jobs in these projects
  projects:
    - id: group1/group1-1/project1
      default_namespace: namespace-to-use-as-default
      access_as:
        agent: {}
        impersonate:
          name: name-of-identity-to-impersonate
          groups:
            - group1
            - group2
          extra:
            key1: [val1, val2]
        ci_job: {}
        ci_user: {}
  # This agent is accessible from CI jobs in projects in these groups
  groups:
    - id: group2/group2-1
      default_namespace: ...
      access_as: ...
```

When a CI job, that has access to one or more agents, runs, GitLab injects a
[`kubectl`-compatible configuration file](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig)
(using a [variable of type `File`](https://docs.gitlab.com/ee/ci/variables/#custom-cicd-variables)) and sets
[`KUBECONFIG` environment variable](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#the-kubeconfig-environment-variable)
to its location on disk. The file contains a
[context](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#context)
per GitLab Kubernetes Agent that this CI job is allowed to access.

If the project, where the CI job is running, has certificate-based integration configured, then the generated
configuration file contains contexts for both integrations. This allows users to use both integration
simultaneously, for example to migrate from one to the other.

CI job can set context `<context name>` as the current one using `kubectl config set-context <context name>`.
A context can also be explicitly specified in each `kubectl` invocation using `kubectl --context=<context name> <command>`.

After a context is selected, `kubectl` (or any other compatible program) can be used as if working with a cluster directly.

## Implementation

### `kubectl` configuration file

- [`Context name`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L165) is
  constructed according to the following pattern: `<configuration project full path>:<agent name>`.

  Example: `groupX/subgroup1/project1:my-agent`.

- [`Server`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L65) is set to
  `https://kas.gitlabhost.tld:<port>`. There needs to be only one
  [`NamedCluster`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L155)
  element in the config that all contexts refer to. It's
  [`Name`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L157) should be set to
  `gitlab`.

- [`Namespace`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L148) is set to
  the value of `projects[].default_namespace`.

- [`Token`](https://github.com/kubernetes/client-go/blob/v0.20.4/tools/clientcmd/api/v1/types.go#L110) is set to the
  value of `<token type>:<agent id>:<CI_JOB_TOKEN>`, where:
  - `<token type>` is the type of the token that is being provided. For CI integration it's the string `ci`. In the
    future we may have more types of tokens that `gitlab-kas` may accept.
  - `<agent id>` is the id of the agent that can be accessed using this context. This
    value and the context's name are the only unique values across contexts.
  - `<CI_JOB_TOKEN>` is the value of the
    [`CI_JOB_TOKEN`](https://docs.gitlab.com/ee/user/project/new_ci_build_permissions_model.html#job-token) variable.

### Identifiers

All identifiers have one of the following structures:

- `gitlab:<identifier type>`

- `gitlab:<identifier type>:<identifier type-specific information>`. `identifier type-specific information` may contain
  columns (`:`) to separate pieces of information.

### Impersonation

User [impersonation](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation),
when configured, supplies identifying information to the in-cluster access control mechanisms,
such as RBAC and admission controllers, when a request is made. This allows Platform Engineers to precisely set up
permissions based on groups and/or "extra".

Identity that is used to make an actual Kubernetes API request in a cluster is configured using
the `access_as` config section. For any option other than `agent` to work, `agentk`'s
`ServiceAccount` needs to have correct permissions. At most one key is allowed:

- `agent` - make the requests using the agent's identity i.e. using the `ServiceAccount` credentials the
  `agentk` `Pod` is running under. This is the default behavior. This is the only impersonation mode where user
  can use the impersonation functionality from the client. In other modes requests with impersonation headers are
  rejected with 400 because they can not be fulfilled - those headers are already in use by the impersonation mode and
  there is no way to perform "nested impersonation".

- `impersonate` - make the requests using some identity.

- `ci_job` - impersonate the CI job. When the agent makes the request to the actual Kubernetes API, it sets the
  impersonation credentials in the following way:

   - `UserName` is set to `gitlab:ci_job:<job id>`

     Example: `gitlab:ci_job:1074499489`.

   - `Groups` is set to:

     - `gitlab:ci_job` to identify all requests coming from CI jobs.

     - The list of ids of groups the project is in.

     - The project id.

     - The slug of the environment this job belongs to.

     Example: for a CI job in `group1/group1-1/project1` where:

     - Group `group1` has id `23`.

     - Group `group1/group1-1` has id `25`.

     - Project `group1/group1-1/project1` has id `150`.

     - Job running in a `prod` environment.

     group list would be [`gitlab:ci_job`, `gitlab:group:23`, `gitlab:group:25`, `gitlab:project:150`,
     `gitlab:project_env:150:prod`].

   - `Extra` carries extra information about the request:

     - `agent.gitlab.com/id` contains the agent id.

     - `agent.gitlab.com/config_project_id` contains the agent's configuration project id.

     - `agent.gitlab.com/project_id` contains the CI project id.

     - `agent.gitlab.com/ci_pipeline_id` contains the CI pipeline id.

     - `agent.gitlab.com/ci_job_id` contains the CI job id.

     - `agent.gitlab.com/username` contains the username of the user the CI job is running as.

     - `agent.gitlab.com/environment_slug` contains the slug of the environment. Only set if running in an environment.

- `ci_user` - impersonate the user this CI job is running as. Details depend on
  https://gitlab.com/gitlab-org/gitlab/-/issues/243740, tentatively:

  - `UserName` is set to `gitlab:user:<username>`

    Example: `gitlab:user:ash2k`.

  - `Groups` is set to:

    - `gitlab:user` to identify all requests coming from GitLab users.

    - The list of roles the user has in the project where the CI job is running.

    Example: for a Maintainer in project `group1/group1-1/project1` with id `150`
    the list of groups would be [`gitlab:user`, `gitlab:project_role:150:reporter`, `gitlab:project_role:150:developer`,
    `gitlab:project_role:150:maintainer`]

  - `Extra` - see above.

  Full list of groups for a user can be huge, so it was decided to use a list of roles the user has instead.

Group/project ids are used because:

- group/project names can be sensitive information that should not be exposed.

- group/project names can change over time, breaking permissions set in RBAC.

### Authentication

Requests to `https://kas.gitlabhost.tld:<port>` are authenticated using the `CI_JOB_TOKEN` that is passed in each request.

### Authorization

There are two authorization steps, performed in the following order:

1. Coarse-grained authorization: the CI job, identified by the supplied `CI_JOB_TOKEN`, is checked to see if it is
   allowed to access a particular agent, identified by the supplied agent id.
   Note that any agent id can be supplied by manipulating the
   configuration file, but only the agent ids that are allowed to be accessed from that particular CI job are allowed to
   pass this authorization step.

1. Fine-grained authorization: performed by the in-cluster access control mechanisms, configured by the Platform
   Engineer. Information, described in the [Impersonation](#impersonation) section above, can be used to define what is allowed.

### Default configuration

Be default, the agent should work without an agent configuration file as well. The following configuration should be the default

```yaml
# .gitlab/agents/my-agent/config.yaml
ci_access:
  # This agent is accessible from CI jobs in these projects
  projects:
    - id: <self>
      default_namespace: <namespace where the agent is installed>
      access_as:
        agent: {}
  groups:
    - id: <group of the agent>
      default_namespace: <namespace where the agent is installed>
      access_as:
        agent: {}
```

### Notifying GitLab of agent's configuration

According to the [proposal](#proposal), user maintains the list of groups and/or projects
in the agent's configuration file. This can be thought of as `agent id` -> `allowed project id` and
`agent id` -> `allowed group id` indexes. We need **reverse** of these i.e. information about agents, allowed for a
project/group to access. It is needed to:

- Implement the [`/api/v4/job/allowed_agents`](#apiv4joballowed_agents-api) API endpoint, providing the list of
  allowed agents with their configuration.

- To be able to construct the `kubectl` configuration file.

https://gitlab.com/gitlab-org/gitlab/-/issues/323708 tracks the plumbing work to make it possible to build such an index.
Once it is implemented, we need to add new indexes to perform `ci project id` -> `agent id` and
`group id` -> `agent id` lookups.

We might add another level of authorization from the group side, if requested by users. This is tracked by https://gitlab.com/gitlab-org/gitlab/-/issues/330591 and is initially out of scope for the CI tunnel.

### `/api/v4/job/allowed_agents` API

`/api/v4/job/allowed_agents` is a new endpoint that returns the required data:

- Information about the CI job, pipeline, project, user.
- The list of agent ids that this CI job is allowed to access.

Only the needed fields are returned, not everything. Algorithm:

1. Retrieve the list of agents allowed to be accessed from the CI project by querying the
   `ci project id` -> `agent id` index.

1. Gather an ordered (from more nested/inner to less nested/outer) list of groups for the CI project by querying the
   `group id` -> `agent id` index.

   Example: for project `group1/group1-1/project1` the list would be [`group1/group1-1`, `group1`].

1. For each group fetch the list of agents, allowed to be accessed by that group. If an agent id has already been seen
   either on step 1 or this step, discard the found information. Keep the most specific configuration for the agent.

   Example: for project `group1/group1-1/project1` the configuration specificity order is:

   1. Project-level configuration `group1/group1-1/project1`.
   1. Inner-most group configuration `group1/group1-1`.
   1. Outer group configuration `group1`.

1. Collate information from above and return it.

Request:

```text
GET /api/v4/job/allowed_agents
Accept: application/json
Job-Token: <CI_JOB_TOKEN>
```

`Job-Token` header name is consistent with other API endpoints that use `CI_JOB_TOKEN` for authentication.

Response on success:

```text
HTTP/1.1 200 OK
Content-Type: application/json

{
  "allowed_agents": [
    {
      "id": 5, // agent id
      "config_project": {
        "id": 3
      },
      "configuration": { // contains section of the agent's config file as is, with 'id' removed
        "default_namespace": "namespace-to-use-as-default",
        "access_as": {
          "agent: {}
        }
      }
    },
    {
      "id": 3,
      "config_project": {
        "id": 3 // same as above
      },
      "configuration": {
        // "default_namespace": "", // not set
        "access_as": {
          "ci_job: {}
        }
      }
    },
    {
      "id": 10,
      "config_project": {
        "id": 11 // agent from a different project
      },
      "configuration": {
        "access_as": {
          "ci_user: {}
        }
      }
    }
  ],
  "job": {
    "id": 3 // job id
  },
  "pipeline": {
    "id": 6 // pipeline id
  },
  "project": {
    "id": 150, // project id
    "groups": [
      {
        "id": 23 // id of the group this project is in
      },
      {
        "id": 25
      }
    ]
  },
  "environment": {
    "slug": "slug_of_the_environment" // empty if not part of an environment
  },
  "user": { // user who is running the job
    "id": 1,
    "username": "root",
    "roles_in_project": [
      "reporter", "developer", "maintainer"
    ]
  }
}
```

### Request proxying flow

1. `gitlab-kas` gets a request from the CI job with `CI_JOB_TOKEN` and agent id in it.

   - If `CI_JOB_TOKEN` is missing, the request is rejected with HTTP code 401.

   - if agent id is missing or invalid, the request is rejected with HTTP code 400.

1. `gitlab-kas` makes a request to [`/api/v4/job/allowed_agents`](#apiv4joballowed_agents-api) endpoint to get the information about the
   `CI_JOB_TOKEN` it received.

   - It handles the HTTP status codes, returning 401/403 on 401/403 i.e. when `CI_JOB_TOKEN` is invalid.

1. `gitlab-kas` checks if the agent id it got in the request from the CI job is in the list it got from `/api/v4/job/allowed_agents`.
   If it is not, the request is rejected with HTTP code 403.

1. (optional) `gitlab-kas` adds impersonation headers to the request based on the agent's configuration.

1. `gitlab-kas` proxies the request to the destination agent, identified by the agent id from the request.
   See [`gitlab-kas` request routing](kas_request_routing.md) for information on how the request routing works.
