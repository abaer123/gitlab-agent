# Extending `gitlab-kas` or `agentk` with new functionality

Functionality is grouped into modules. Each module has a server and an agent part, both are optional, depending on the needs of the module. Parts are called "server module" and "agent module" for simplicity.

Each module has a unique name that is used to identify it for API access, if needed.

## Server module

API for module's server part is defined in the `internal/modules/modserver` directory.

Responsibilities:

- Validates and applies defaults to the corresponding part of the configuration. Most of defaulting happens here, not in the agent module.
- Optionally registers gRPC services on the gRPC server that `agentk` talks to.
- Implements the required functionality.

## Agent module

API for module's agent part is defined in `internal/modules/modagent` directory.

Responsibilities:

- Validates and applies defaults to the corresponding part of the configuration. Defaulting only ensures that fields are not `nil` (to avoid having to check that everywhere).
- Optionally registers gRPC services on the gRPC server that `gitlab-kas` talks to. See [`kas` request routing](kas_request_routing.md).
- Implements the required functionality.

## Structure

A module lives under `internal/modules/{module name}`. Each module may contains one or two parts in separate directories, named after the module:

- Server module directory name pattern: `{module name}_server`.
- Agent module directory name pattern: `{module name}_agent`.

Any code, that needs to be shared between server and agent modules, may be placed directly in the module's directory or a separate subdirectory.

Code for server and agent modules must be in separate directories (i.e. Go packages) to avoid adding unnecessary dependencies from one to the other. That way server module's libraries don't leak into agent module package and vice versa. `gitlab-kas` must only depend on server modules and `agentk` must only depend on agent modules.

Modules may share code via separate packages but must not depend on each other directly. `internal/modules/{module A}` can depend on `internal/modules/{module B}/shared_package`.
