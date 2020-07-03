# Identity and authentication

This document uses the word `agent` to mean GitLab Kubernetes Agent on the conceptual level. The program that is the implementation of it is actually called `agentk`. See the [architecture page](architecture.md).

## Cluster identity

Each cluster has an identity that is unique within a GitLab installation.

## Agent identity and name

Each agent has an identity that is unique within a GitLab installation. Each agent has a name that is unique within the project it is attached to. Agent's name is used to find agent's configuration in the project's [configuration repository](configuration_repository.md).

Each agent belongs to a single Kubernetes cluster. A Kubernetes cluster may have 0 or more agents registered for it.

## Authentication

When adding a new agent the user gets a bearer access token for it from the UI wizard. The agent uses this token to authenticate with GitLab. It is a secret and must be treated as such e.g. stored in a `Secret` in Kubernetes. The token is a random string and does not encode any information in it.

Each agent may have 0 or more tokens in GitLab's database. Ability to have several valid tokens helps facilitate token rotation without having to re-register an agent. Each token record in the database has:

- Agent identity it belongs to
- Creation time
- Who created it
- Revocation flag to mark tokens as revoked
- Revocation time
- A text field to store any comments the administrator may want to make about the token for future self

For each request from an agent GitLab checks if the token is valid - exists in the database and has not been revoked. This information may be cached for some time to reduce load on the database.

Tokens can be managed by users with `maintainer` and higher level of permissions.

## Authorization

GitLab will provide the following information as part of the response for a given Agent access token:

- Agent config git repository (Note: we don't have per-folder authorization)
- Agent name
- Manifest projects: TBD on how kgb and agentk deploys manifest
