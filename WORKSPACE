workspace(name = "gitlab_k8s_agent")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# When updating rules_go make sure to update org_golang_x_tools dependency below by copying it from
# https://github.com/bazelbuild/rules_go/blob/master/go/private/repositories.bzl
# Also update to the same version/commit in go.mod.
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "2697f6bc7c529ee5e6a2d9799870b9ec9eaeb3ee7d70ed50b87a2c2c97e13d9e",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.8/rules_go-v0.23.8.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.8/rules_go-v0.23.8.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d50aa00c2eb4f4a9b4514f45ee9ff7bcf744214b36cc0f1b4affec7b0a0bb9c3",
    strip_prefix = "bazel-gazelle-49a5b63911d89f6bcc5a8c771ea2a3c63c821996",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/archive/49a5b63911d89f6bcc5a8c771ea2a3c63c821996.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "4521794f0fba2e20f3bf15846ab5e01d5332e587e9ce81629c7f96c793bb7036",
    strip_prefix = "rules_docker-0.14.4",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.14.4/rules_docker-v0.14.4.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "315dad13406928011b467ca7f2748a59ae817477f9129e1edaae75deb73e9b78",
    strip_prefix = "buildtools-3.4.0",
    urls = ["https://github.com/bazelbuild/buildtools/archive/3.4.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "47f05775787fe8f7cde87cbf2eeb98df0b15b6c1969d7e651f4ee42bbd3ebb7a",
    strip_prefix = "bazel-tools-dc5e715035b6b17f24f1d40a7eac08f8f2ac8a11",
    urls = ["https://github.com/atlassian/bazel-tools/archive/dc5e715035b6b17f24f1d40a7eac08f8f2ac8a11.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "dedb72afb9476b2f75da2f661a00d6ad27dfab5d97c0460cf3265894adfaf467",
    strip_prefix = "rules_proto-486aaf1808a15b87f1b6778be6d30a17a87e491a",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/486aaf1808a15b87f1b6778be6d30a17a87e491a.tar.gz",
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
    sum = "h1:2rkR3ffvd5YVyPYU4LAUDCKoKQZtjuuj8ga15mbv96o=",
    version = "v1.18.6",
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
