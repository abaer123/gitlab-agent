# Frequently Asked Questions

This document collects random bits of knowledge about the project which do not fit anywhere else. Also see [architecture-related FAQ](architecture.md#faq).

- **Q**: Why is the build slow? Why does the build pull so many dependencies?

  **A**: `agentk` uses [`gitops-engine`](https://github.com/argoproj/gitops-engine) to implement [GitOps](gitops.md). `gitops-engine` for better or worse [depends on `k8s.io/kubernetes`](https://github.com/argoproj/gitops-engine/issues/56) package, which is the whole Kubernetes repository. Even though it uses only a few packages, because of  defects in the `go` module resolution code ([one](https://github.com/golang/go/issues/31580), [two](https://github.com/golang/go/issues/33008)), `go.sum` for this project contains all (?) the dependencies of Kubernetes. `go.sum` is used by [Gazelle](https://github.com/bazelbuild/bazel-gazelle) to [generate dependency information](https://github.com/bazelbuild/bazel-gazelle#update-repos) for Bazel. This is why the project has a lot of dependencies to download and hence is slow to build.

- **Q**: Why does the build print so many `go: finding module` messages?

  **A**: See the previous entry. Kubernetes uses some packages from Docker, which depend on the logrus library. `go` modules support does not like invalid case of imports that some packages use and barfs at it. This is annoying but harmless. It may be fixed in newer Kubernetes/Docker versions already.
