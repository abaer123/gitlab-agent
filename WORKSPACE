workspace(name = "gitlab_k8s_agent")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# When updating rules_go make sure to update org_golang_x_tools dependency below by copying it from
# https://github.com/bazelbuild/rules_go/blob/master/go/private/repositories.bzl
# Also update to the same version/commit in go.mod.
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "0c10738a488239750dbf35336be13252bad7c74348f867d30c3c3e0001906096",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.2/rules_go-v0.23.2.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.2/rules_go-v0.23.2.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "3efbd23e195727a67f87b2a04fb4388cc7a11a0c0c2cf33eec225fb8ffbb27ea",
    strip_prefix = "rules_docker-0.14.2",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.14.2/rules_docker-v0.14.2.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "a0e79f5876a1552ae8000882e4189941688f359a80b2bc1d7e3a51cab6257ba1",
    strip_prefix = "buildtools-3.0.0",
    urls = ["https://github.com/bazelbuild/buildtools/archive/3.0.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "0e556f6d537df818dce1caadc825936f43a555773040e35c8b131067a85f11cd",
    strip_prefix = "bazel-tools-936325de16966d259eee3f309f8578b761cfc874",
    urls = ["https://github.com/atlassian/bazel-tools/archive/936325de16966d259eee3f309f8578b761cfc874.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "6117a0f96af1d264747ea3f3f29b7b176831ed8acfd428e04f17c48534c83147",
    strip_prefix = "rules_proto-8b81c3ccfdd0e915e46ffa888d3cdb6116db6fa5",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/8b81c3ccfdd0e915e46ffa888d3cdb6116db6fa5.tar.gz",
    ],
)

http_archive(
    name = "rules_proto_grpc",
    sha256 = "5f0f2fc0199810c65a2de148a52ba0aff14d631d4e8202f41aff6a9d590a471b",
    strip_prefix = "rules_proto_grpc-1.0.2",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/archive/1.0.2.tar.gz"],
)

http_archive(
    name = "bazel_skylib",
    sha256 = "97e70364e9249702246c0e9444bccdc4b847bed1eb03c5a3ece4f83dfe6abc44",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

# Workaround for https://github.com/argoproj/gitops-engine/issues/56
go_repository(
    name = "io_k8s_kubernetes",
    # Kubernetes uses "BUILD" files, we use "BUILD.bazel" to ignore them.
    build_file_name = "BUILD.bazel",
    build_file_proto_mode = "disable_global",
    importpath = "k8s.io/kubernetes",
    replace = "k8s.io/kubernetes",
    sum = "h1:VKkgXDo0PzBeoVABOWc4/ozDOzoouHbNxljqXC8TRIg=",
    version = "v1.17.8",
)

# Copied from rules_go to keep patches in place
http_archive(
    name = "org_golang_x_tools",
    patch_args = ["-p1"],
    patches = [
        # deletegopls removes the gopls subdirectory. It contains a nested
        # module with additional dependencies. It's not needed by rules_go.
        "@io_bazel_rules_go//third_party:org_golang_x_tools-deletegopls.patch",
        # gazelle args: -repo_root . -go_prefix golang.org/x/tools
        "@io_bazel_rules_go//third_party:org_golang_x_tools-gazelle.patch",
        # extras adds go_tool_library rules for packages under
        # go/analysis/passes and their dependencies. These are needed by
        # nogo.
        "@io_bazel_rules_go//third_party:org_golang_x_tools-extras.patch",
    ],
    sha256 = "b05c5b5b9091a35ecb433227ea30aa75cb6b9d9409b308bc75d0975d4a291912",
    strip_prefix = "tools-2bc93b1c0c88b2406b967fcd19a623d1ff9ea0cd",
    # master, as of 2020-05-12
    urls = [
        "https://mirror.bazel.build/github.com/golang/tools/archive/2bc93b1c0c88b2406b967fcd19a623d1ff9ea0cd.zip",
        "https://github.com/golang/tools/archive/2bc93b1c0c88b2406b967fcd19a623d1ff9ea0cd.zip",
    ],
)

load("//build:repositories.bzl", "go_repositories")

# gazelle:repository_macro build/repositories.bzl%go_repositories
go_repositories()

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

go_rules_dependencies()

go_register_toolchains()

gazelle_dependencies()

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@com_github_atlassian_bazel_tools//buildozer:deps.bzl", "buildozer_dependencies")
load("@com_github_atlassian_bazel_tools//multirun:deps.bzl", "multirun_dependencies")
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)
load(
    "@io_bazel_rules_docker//go:image.bzl",
    go_image_repositories = "repositories",
)
load(
    "@io_bazel_rules_docker//cc:image.bzl",
    cc_image_repositories = "repositories",
)
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")
load("@rules_proto_grpc//:repositories.bzl", "rules_proto_grpc_toolchains")
load("@rules_proto_grpc//go:repositories.bzl", rules_proto_grpc_go_repos = "go_repos")

container_repositories()

go_image_repositories()

cc_image_repositories()

buildifier_dependencies()

buildozer_dependencies()

multirun_dependencies()

rules_proto_dependencies()

rules_proto_toolchains()

rules_proto_grpc_toolchains()

rules_proto_grpc_go_repos()
