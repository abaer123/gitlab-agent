# Extending `gitlab-kas` or `agentk` with new functionality

Functionality is grouped into modules. Each module has a server and an agent part, both are optional, depending on the needs of the module. Parts are called "server module" and "agent module" for simplicity.

Each module has a unique name that is used to identify it for API access, if needed.

## Structure

A module lives under `internal/module/{module name}`. Each module may contains one or two parts in separate directories: `server` and `agent`. Any code, that needs to be shared between server and agent modules, may be placed directly in the module's directory or a separate subdirectory.

Code for server and agent modules must be in separate directories (i.e. Go packages) to avoid adding unnecessary dependencies from one to the other. That way server module's libraries don't leak into agent module package and vice versa. `gitlab-kas` must only depend on server modules and `agentk` must only depend on agent modules.

Modules may share code via separate packages but must not depend on each other directly. `internal/module/{module A}` (and any subdirectories) can depend on `internal/module/{module B}/some_shared_package`.

## Server module

API for module's server part is defined in the `internal/module/modserver` directory.

### Responsibilities

- Validates and applies defaults to the corresponding part of the `gitlab-kas` configuration.
- Optionally registers gRPC services on the gRPC server that `agentk` talks to.
- Implements the required functionality.

## Agent module

API for module's agent part is defined in `internal/module/modagent` directory.

### Responsibilities

- Validates and applies defaults to the corresponding part of the `agentk` configuration.
- Optionally registers gRPC services on the gRPC server that `gitlab-kas` talks to. See [`kas` request routing](kas_request_routing.md).
- Implements the required functionality.

### Making requests to GitLab

To make requests to GitLab an agent module may use `MakeGitLabRequest()` method on the `modagent.API` object. Using a dedicated method rather than making a REST request directly is beneficial for the following reasons:

- Adds an indirection point that allows `agentk` to inject any necessary interceptors for tracing, monitoring, rate limiting, etc. In general this ensures all requests are made in a uniform manner.
- Modules don't have to deal with authentication and agent token handling, which is a secret. Reducing the exposure of the token within the program improves security because the chance of token leaking is reduced.
- All traffic originating from all agents uses a single entrypoint - `gitlab-kas`. Benefits:

  - It makes it possible to expose only `gitlab-kas` domain and not the rest of GitLab in a case where GitLab is deployed as a self-managed instance with the Kubernetes cluster being in a cloud.
  - `gitlab-kas` performs rate-limiting, monitoring, etc across the board for all GitLab access originating from all the agents.
