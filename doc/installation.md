# Attaching an existing cluster to GitLab using the GitLab Kubernetes Agent

The GiLab Kubernetes Agent allows an Infrastructure as Code, GitOps approach to integrating Kubernetes clusters with GitLab. The agent allows strict authorization rules to be set up around GitLab's possibilities. Installing the agent requires access to your cluster.

## Step 1: Set up the agent configuration

The agent configuration is stored in a repository, that the agent pulls automatically to configure itself. The agent configuration project needs minimal configuration to work, for detailed examples and features, see the documentation.

1. [Create a new project using the GitLab Kubernetes Agent Configuration project template](https://gitlab.com/projects/new#create_from_template) - should be simpler as it is now
2. Provide the project slug of your forked project (input box)

## Step 2: Install the agent

### Store the registration token inside the cluster

Installing the agent requires a secret token to be available in your cluster that the agent has access to. The agent will use this token to authenticate with GitLab.

| Registration token | Kubernetes manifest |
| -- | -- |

Token expires: 2020-06-23 19:18:00 (i) - information button says that every token is valid for 24 hours, if unused.
Button: Refresh token

#### Registration token

&lt;token comes here

#### Kubernetes manifest

```
apiVersion: v1
kind: Secret
metadata:
  name: gitlab-kubernetes-agent
type: Opaque
data:
  token: MWYyZDFlMmU2N2Rm
```

### Install the agent

You can install the agent using your favourite kubernetes management tool.

| Kubernetes manifest | Helm | Infrastructure as Code repo | kpt |
| -- | -- | -- | -- |

#### Kubernetes manifest

Run the following command to install the GitLab Kubernetes Agent:

`kubectl apply --filename https://gitlab.com/nagyv-gitlab/kubernetes-mindmap/snippets/1989522/raw`

The above command will create a service account with `cluster-admin` rights, and install the agent into the `gitlab-agent` namespace. If you'd like more fine-grained control, like a specially set up service account, read the detailed installation instructions below.

You can install each piece separately using the following commands

```
kubectl apply --filename https://gitlab.com/nagyv-gitlab/kubernetes-mindmap/snippets/1989522/raw
kubectl apply --filename https://gitlab.com/nagyv-gitlab/kubernetes-mindmap/snippets/1989522/raw
```

#### Helm

Run the following command to install the GitLab Kubernetes Agent using Helm:

```
helm repo add gitlab https://charts.gitlab.io/
helm install gitlab/gitlab-kubernetes agent --version 0.1.0
```

At the moment our Helm based installation lacks customization options. Please, [provide feedback on this issue](#) about what you'd like to be able to configure in a `values.yml`.

#### Infrastructure as Code repo

Feel free to fork https://gitlab.com/.... and add it to your pipeline

#### Kpt
