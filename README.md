# gitlab-agent

GitLab Kubernetes Agent is an active in-cluster component for solving any GitLab<->Kubernetes integration tasks.

**This is a work in progress, it's not used anywhere yet.**

Please see the [architecture](doc/architecture.md) document and other documents in the [doc](doc) directory for more information.

## Use cases and ideas

Below are some ideas that can be built using the agent.

* “Real-time” and resilient web hooks. Polling git repos scales poorly and so webhooks were invented. They remove polling, easing the load on infrastructure, and reduce the "event happened->it got noticed in an external system" latency. However, "webhooks" analog cannot work if cluster is behind a firewall. So an agent, runnning in the cluster, can connect to GitLab and receive a message when a change happens. Like web hooks, but the actual connection is initiated from the client, not from the server. Then the agent could:

  * Emulate a webhook inside of the cluster

  * Update a Kubernetes object with a new state. It can be a GitLab-specific object with some concrete schema about a git repository. Then we can have third-parties integrate with us via this object-based API. It can also be some integration-specific object.

* “Real-time” data access. Agent can stream requested data back to GitLab. See https://gitlab.com/gitlab-org/gitlab/-/issues/212810.

* Feature/component discovery. GitLab may need a third-party component to be installed in a cluster for a particular feature to work. Agent can do that component discovery. E.g. we need Prometheus for metrics and we probably can find it in the cluster (is this a bad example? it illustrates the idea though).

* Prometheus PromQL API proxying. Configure where Prometheus is available in the cluster, and allow GitLab to issue PromQL queries to the in-cluster Prometheus.

* Better [GitOps](https://www.gitops.tech/) support. A repository can be used as a IaC repo. On successful CI run on the main repo, a commit is merged into that IaC repo. Commit describes the new desired state of infrastructure in a particular cluster (or clusters). An agent in a corresponding cluster(s) picks up the update and applies it to the objects in the cluster. We can work with Argo-cd/Flux here to try to reuse existing code and integrate with the community-built tools.

* “Infrastructure drift detection”. Monitor and alert on unexpected changes in Kubernetes objects that are managed in the IaC repo. Should support various ways to describe infrastructure (kustomize/helm/plain yaml/etc). 

* Preview changes to IaC specs against the current state of the corresponding cluster right in the MR. 

* “Live diff”. Building on top of the previous feature. In repo browser when a directory with IaC specs is opened, show a live comparison of what is in the repo and what is in the corresponding cluster. 

* Kubernetes has audit logs. We could build a page to view them and perhaps correlate with other GitLab events? 

* See how we can support https://github.com/kubernetes-sigs/application.

  * In repo browser detect resource specs with the defined annotations and show the relevant meta information bits
  * Have a panel showing live list of installed applications based on the annotations from the specification

* Emulate Kubernetes API and proxy it into the actual cluster via the agents (to overcome the firewall). Do we even need this?

## Open questions and things to consider

### GitLab.com + `agentk`

We have CloudFlare CDN in front of GitLab.com. The connections that `agentk` establishes are long-running by design. It may or may not be an issue. See https://gitlab.com/groups/gitlab-com/gl-infra/-/epics/228

### High availability and scalability

#### GitLab Kubernetes Agent (`agentk`)

`agentk` is the agent on the Kubernetes side.

Multiple `Pod`s per deployment would be needed to have a highly available deployment. This might be trivial but might require doing [leader election](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection?tab=doc), depending on the functionality the agent will provide.

#### GitLab Kubernetes Agent Server (`kas`)

`kas` is the server-side counterpart of `agentk`.

The difficulty of having multiple copies of the program is that only one of the copies has an active connection from a particular Kubernetes cluster. So to serve a request from GitLab targeted at that cluster some sort of request routing would be needed. There are options (off the top of my head):

- [Consistent hashing](https://en.wikipedia.org/wiki/Consistent_hashing) could be used to:
  - Minimize disruptions if a copy of `kas` goes missing
  - Route traffic to the correct copy

  We use [`nginx` as our ingress controller](https://docs.gitlab.com/charts/charts/nginx/index.html) and it does [support consistent hashing](https://www.nginx.com/resources/wiki/modules/consistent_hash/).

- We could have `nginx` ask one (any) of `kas` where the right copy (the one that has the connection) running. `kas` can do a lookup in Redis where each cluster connection is registered and return the address to `nginx` via [X-Accel-Redirect](https://www.nginx.com/resources/wiki/start/topics/examples/x-accel/#x-accel-redirect) header.

- We could teach `kas` to gossip with its copies so that they tell each other where a connection for each cluster is. Each copy will know about all the connections and either proxy the request or use `X-Accel-Redirect` to redirect the traffic. This is [Cassandra's gossip](https://docs.datastax.com/en/cassandra-oss/3.x/cassandra/architecture/archGossipAbout.html) + Cassandra's coordinator node-based request routing ideas but it's much easier to build on Kubernetes because we can just use the API to find other copies of the program.

### `agentk` topologies

In a cluster `agentk` can be deployed:

- One or more cluster-wide deployment that works across all namespaces. Useful to manage cluster-wide state using GitOps. As long as deployments are configured to avoid any conflicts, this can work and be beneficial to allow separate deployments for separate tasks.
- One or more per-namespace deployments, each concerned only with what is happening in a particular namespace. Note that namespace where `agentk` is deployed, and the namespace it's managing might be different namespaces.
- Both of the above at the same time.

Because of the above, each `agentk` copy must have its own identity for GitLab to be able to tell one from the other e.g. for troubleshooting reasons. Each copy should get a URL of `kas` and fetch the configuration from it. In that configuration the agent should see if it's per-namespace or cluster-wide. Configuration is stored in a Git repo on GitLab to promote IaC approach.

Each `agentk` copy also gets its own `ServiceAccount` with minimum required permissions.

### Permissions within the cluster

Currently customers are rightly concerned with us asking cluster-admin access. For GitOps and similar functionality something still has to have permissions to CRUD Kubernetes objects. The solution here is to give cluster operator (our customer) exclusive control of the permissions. Then they can allow the agent do only what they want it to be able to do. Where RBAC is not flexible enough (e.g. namespaces - don't want to allow CRUD for arbitrary namespaces but only some, based on some logic), we can provide an admission webhook that enforces some rules for the agent's `ServiceAccount` in addition to RBAC.

### Environments

How to map GitLab's [environments](https://gitlab.com/help/ci/environments) onto clusters/agents/namespaces? This link states the following:

> It's important to know that:
>
> * Environments are like tags for your CI jobs, describing where code gets deployed.

We can follow this model and mark each agent as belonging to one or more environments. It's a many to many relationship:
- Multiple agents can be part of an environment. Example: X prod clusters with some number of agents each
- An agent can be part of multiple environments. Example: a cluster-wide agent where the cluster is used for both production and non-production deployments

Note that cluster-environment is a many to many relationship too:

- A cluster may be part of multiple environments
- An environment can include several clusters
