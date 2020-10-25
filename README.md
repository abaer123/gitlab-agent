# gitlab-agent

GitLab Kubernetes Agent is an active in-cluster component for solving any GitLab<->Kubernetes integration tasks.

It's implemented as two communicating pieces - GitLab Kubernetes Agent (`agentk`) that is running in the cluster and GitLab Kubernetes Agent Server (`gitlab-kas`) that is running on the GitLab side. Please see the [architecture](doc/architecture.md) document and other documents in the [doc](doc) directory for more information. [User-facing documentation](https://docs.gitlab.com/ee/user/clusters/agent/) is also available.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
