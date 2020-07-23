# Identity and authentication

This document uses the word `agent` to mean GitLab Kubernetes Agent on the conceptual level. The program that is the implementation of it is actually called `agentk`. See the [architecture page](architecture.md).

## Agent identity and name

Each agent has an identity that is unique within a GitLab installation. Each agent has an immutable name that is unique within the project the agent is attached to. Agent names follow the DNS label standard as defined in [RFC 1123](https://tools.ietf.org/html/rfc1123). This means the name must:

- contain at most 63 characters.
- contain only lowercase alphanumeric characters or `-`.
- start with an alphanumeric character.
- end with an alphanumeric character.

Kubernetes uses the [same naming restriction](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names) for some names.

The regex for names is: `/\A[a-z0-9]([-a-z0-9]*[a-z0-9])?\z/`.

## Multiple agents in a cluster

A Kubernetes cluster may have 0 or more agents running in it. Each of these agents likely has a different configuration. Some may have features A and B turned on and some - B and C. This flexibility is desirable to allow potentially different groups of people to use different features of the agent in the same cluster. For example, [Priyanka (Platform Engineer)](https://about.gitlab.com/handbook/marketing/product-marketing/roles-personas/#priyanka-platform-engineer) may want to use cluster-wide features of the agent while [Sasha (Software Developer)](https://about.gitlab.com/handbook/marketing/product-marketing/roles-personas/#sasha-software-developer) uses the agent that has access to a particular namespace only.

Each agent is likely running using a distinct Kubernetes identity - [`ServiceAccount`](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/). Each `ServiceAccount` may have a distinct set of permissions attached to it. This allows the agent administrator to minimize the permissions for each particular agent depending on the configured features to follow the [principle of least privilege](https://en.wikipedia.org/wiki/Principle_of_least_privilege).

## Authentication

When adding a new agent the user gets a bearer access token for it from the UI wizard. The agent uses this token to authenticate with GitLab. It is a secret and must be treated as such e.g. stored in a `Secret` in Kubernetes. The token is a random string and does not encode any information in it.

Each agent may have 0 or more tokens in GitLab's database. Ability to have several valid tokens helps facilitate token rotation without having to re-register an agent. Each token record in the database has:

- Agent identity it belongs to
- Token value. Encrypted at rest.
- Creation time.
- Who created it.
- Revocation flag to mark token as revoked.
- Revocation time.
- Who revoked it.
- A text field to store any comments the administrator may want to make about the token for future self.

Tokens are immutable. Only the following fields can be updated:
- Revocation flag. Can only be updated to `true` once. Immutable after that.
- Revocation time. Set automatically to the current time when revocation flag is set. Immutable after that.
- Comments field. Can be updated any number of times, including after the token has been revoked.

The agent sends its token along with each request to GitLab to authenticate itself. For each request GitLab checks if the token is valid - exists in the database and has not been revoked. This information may be cached for some time to reduce load on the database.

Tokens can be managed by users with `maintainer` and higher level of permissions.

## Authorization

GitLab will provide the following information as part of the response for a given Agent access token:

- Agent config git repository (Note: we don't have per-folder authorization)
- Agent name
