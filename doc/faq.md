# Frequently Asked Questions

This document collects random bits of knowledge about the project which do not fit anywhere else. Also, see [architecture-related FAQ](architecture.md#faq).

- **Q**: Why is the build slow? Why does the build pull so many dependencies?

  **A**: `agentk` uses [`gitops-engine`](https://github.com/argoproj/gitops-engine) to implement [GitOps](gitops.md).
         `gitops-engine` for better or worse
         [depends on `k8s.io/kubernetes`](https://github.com/argoproj/gitops-engine/issues/56) package, which is the
         whole Kubernetes repository. We also depend on Cilium, which is [quite big too](https://github.com/cilium/cilium/issues/15823).
         This is why the project has a lot of dependencies to download and hence is slow
         to build.

- **Q**: Why `agentk` is written in Go?

  **A**: Kubernetes is written in Go as is the whole ecosystem around it. One of the reasons for it is to be able to use bits of Kubernetes that it exports as libraries. Some very important libraries don't have analogs in other languages e.g. [informers](https://github.com/kubernetes/client-go/blob/ccd5becdffb7fd8006e31341baaaacd14db2dcb7/tools/cache/shared_informer.go#L34-L183).

- **Q**: Why `kas` is written in Go (and not Ruby)?

  **A**:

  - Same as the question above, but to a lesser extent - to use libraries from Kubernetes.
  - Go is perfect for handling long-running connections, Ruby on Rails is not as good.

- **Q**: Why is `kas` not part of the Ruby monolith?

  **A**: Because it's not written in Ruby.

- **Q**: Why use [`zap`](https://github.com/uber-go/zap) instead of [`Logrus`](https://github.com/Sirupsen/logrus) as the logging library?

  **A**: There are several reasons:

  - `Logrus` is in maintenance mode. Why introduce a library that is on life support into a fresh codebase when there are much better alternatives, recommended by its authors?
  - `zap` has a better API that is more convenient to use.
  - `zap` allows to define type-safe log field helpers. All field helpers can be placed into a single file to:
    - help see if there are name clashes.
    - ensure log fields have consistent types. See the next Q.
  - Please see https://gitlab.com/gitlab-org/labkit/-/issues/24#note_407438805 for other benefits.
  - Handbook [recommendation](https://docs.gitlab.com/ee/development/go_guide/#logging) to use `Logrus` [may change in the future](https://gitlab.com/gitlab-org/labkit/-/issues/24) to `zap`.

- **Q**: Why is it important for each log field to be of a type, that is consistent across all logs coming from a single program?

  **A**: In large-scale deployments it's hard to analyze logs because they are scattered across many servers/Pods/VMs. Hence, logs are collected into some central storage, typically Elasticsearch or Splunk, where they can be queried, etc. Each log producer usually gets an individual index, e.g. to avoid mixing logs. Both of these systems expect documents, not raw text lines and hence it's typical to require programs to output structured logs, [like we do](https://docs.gitlab.com/ee/development/go_guide/#structured-json-logging), to simplify parsing. For documents, schema is defined explicitly or implicitly. Each field [has a type](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-types.html). It's an error to try to inject a document that does not conform to the schema. When that happens, logs are usually discarded or put somewhere else, where structure is not enforced, making them unavailable for analysis. This is not great and should be avoided. Also, naive log ingestion pipelines can get stuck re-trying to process logs with incorrectly typed fields ad infinitum.
