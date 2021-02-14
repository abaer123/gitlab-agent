workspace(name = "gitlab_k8s_agent")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

# When updating rules_go make sure to update org_golang_x_tools dependency below by copying it from
# https://github.com/bazelbuild/rules_go/blob/master/go/private/repositories.bzl
# Also update to the same version/commit in go.mod.
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7904dbecbaffd068651916dce77ff3437679f9d20e1a7956bff43826e7645fcc",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.25.1/rules_go-v0.25.1.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "c15ef66698f5d2122a3e875c327d9ecd34a231a9dc4753b9500e70518464cc21",
    strip_prefix = "rules_docker-7da0de3d094aae5601c45ae0855b64fb2771cd72",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/7da0de3d094aae5601c45ae0855b64fb2771cd72.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "a02ba93b96a8151b5d8d3466580f6c1f7e77212c4eb181cba53eb2cae7752a23",
    strip_prefix = "buildtools-3.5.0",
    urls = ["https://github.com/bazelbuild/buildtools/archive/3.5.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "c0a8dd5f94912fc782220467c9e22b4404985082f85408a3390dbe44205d4180",
    strip_prefix = "bazel-tools-5c3b9306e703c6669a6ce064dd6dde69f69cba35",
    urls = ["https://github.com/atlassian/bazel-tools/archive/5c3b9306e703c6669a6ce064dd6dde69f69cba35.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "d8992e6eeec276d49f1d4e63cfa05bbed6d4a26cfe6ca63c972827a0d141ea3b",
    strip_prefix = "rules_proto-cfdc2fa31879c0aebe31ce7702b1a9c8a4be02d2",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/cfdc2fa31879c0aebe31ce7702b1a9c8a4be02d2.tar.gz",
    ],
)

http_archive(
    name = "rules_proto_grpc",
    sha256 = "d771584bbff98698e7cb3cb31c132ee206a972569f4dc8b65acbdd934d156b33",
    strip_prefix = "rules_proto_grpc-2.0.0",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/archive/2.0.0.tar.gz"],
)

http_archive(
    name = "bazel_skylib",
    sha256 = "1c531376ac7e5a180e0237938a2536de0c54d93f5c278634818e0efc952dd56c",
    urls = [
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.3/bazel-skylib-1.0.3.tar.gz",
    ],
)

http_archive(
    name = "rules_pkg",
    sha256 = "6b5969a7acd7b60c02f816773b06fcf32fbe8ba0c7919ccdc2df4f8fb923804a",
    url = "https://github.com/bazelbuild/rules_pkg/releases/download/0.3.0/rules_pkg-0.3.0.tar.gz",
)

http_archive(
    name = "tool_kpt",
    build_file_content = 'exports_files(["kpt"])',
    sha256 = "e423802ab65e77c0d79d22effcd81ea726153f5347f42fb09f84b275ca5bb67f",
    urls = ["https://github.com/GoogleContainerTools/kpt/releases/download/v0.37.1/kpt_linux_amd64-0.37.1.tar.gz"],
)

http_archive(
    name = "tool_kustomize",
    build_file_content = 'exports_files(["kustomize"])',
    sha256 = "bab4ab8881718c29ba174bdf677fd89986ad25c40eb363fec9e78c1aff2ff0ea",
    urls = ["https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.10.0/kustomize_v3.10.0_linux_amd64.tar.gz"],
)

http_file(
    name = "tool_git",
    downloaded_file_path = "git.deb",
    sha256 = "32c6107a72665be54354585da69a464807a5c3d64654826307255da2c64bf930",
    urls = ["http://ftp.debian.org/debian/pool/main/g/git/git_2.30.0-1_amd64.deb"],
)

http_file(
    name = "tool_libpcre2",
    downloaded_file_path = "libpcre2.deb",
    sha256 = "18fa901205ed21c833ff669daae26f675803147f4cc64ddc95fc9cddd7f654c8",
    urls = ["http://ftp.debian.org/debian/pool/main/p/pcre2/libpcre2-8-0_10.32-5_amd64.deb"],
)

http_file(
    name = "tool_zlib1g",
    downloaded_file_path = "zlib1g.deb",
    sha256 = "61bc9085aadd3007433ce6f560a08446a3d3ceb0b5e061db3fc62c42fbfe3eff",
    urls = ["http://ftp.debian.org/debian/pool/main/z/zlib/zlib1g_1.2.11.dfsg-1_amd64.deb"],
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
    sum = "h1:SHJ+Ua3v2nzKB6JEGJilKv/JIiVJFo+dCs707Z3H0nk=",
    version = "v1.20.1",
)

# It's here to add build tags
go_repository(
    name = "com_gitlab_gitlab_org_labkit",
    build_file_proto_mode = "disable_global",
    # The same list of go build tags must be in three places:
    # - Makefile
    # - Workspace
    # - .bazelrc
    build_tags = [
        "tracer_static",
        "tracer_static_jaeger",
    ],  # keep
    importpath = "gitlab.com/gitlab-org/labkit",
    sum = "h1:PDP4id5YEvw6juWrGE88LcTtEridtRAOyvNvUOtcc9o=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_envoyproxy_protoc_gen_validate",
    build_file_proto_mode = "disable_global",
    build_naming_convention = "go_default_library",
    importpath = "github.com/envoyproxy/protoc-gen-validate",
    patch_args = ["-p1"],
    # patch addresses https://github.com/bazelbuild/bazel-gazelle/issues/941
    # patch created by manually editing the build file and running `diff -urN protoc-gen-validate protoc-gen-validate-copy`
    patches = [
        "@gitlab_k8s_agent//build:validate_dependency.patch",
    ],
    sum = "h1:NVV8ILDiHugAAoCtGP7ERYoaumUffVMPQB2CV9C7jNs=",
    version = "v0.4.2-0.20201217164128-7df253a68e6b",
)

# Copied from rules_go to keep patches in place
# https://github.com/bazelbuild/rules_go/blob/master/go/private/repositories.bzl
http_archive(
    name = "org_golang_x_tools",
    patch_args = ["-p1"],
    patches = [
        # deletegopls removes the gopls subdirectory. It contains a nested
        # module with additional dependencies. It's not needed by rules_go.
        "@io_bazel_rules_go//third_party:org_golang_x_tools-deletegopls.patch",
        # gazelle args: -repo_root . -go_prefix golang.org/x/tools -go_naming_convention import_alias
        "@io_bazel_rules_go//third_party:org_golang_x_tools-gazelle.patch",
    ],
    sha256 = "fe3987ccdff6a0e7e5a8353d4d1d2ca3ada5a72ea69462ba7b9b7343b5a25e06",
    strip_prefix = "tools-a1b87a1c0de44760bd00894ef736a8c36548068f",
    # master, as of 2020-12-01
    urls = [
        "https://mirror.bazel.build/github.com/golang/tools/archive/a1b87a1c0de44760bd00894ef736a8c36548068f.zip",
        "https://github.com/golang/tools/archive/a1b87a1c0de44760bd00894ef736a8c36548068f.zip",
    ],
)

# Here to set build_file_proto_mode=default. repositories.bzl sets it to disable_global which is not what we want.
go_repository(
    name = "com_github_lyft_protoc_gen_star",
    build_file_proto_mode = "default",
    importpath = "github.com/lyft/protoc-gen-star",
    sum = "h1:sImehRT+p7lW9n6R7MQc5hVgzWGEkDVZU4AsBQ4Isu8=",
    version = "v0.5.1",
)

load("//build:repositories.bzl", "go_repositories")

# gazelle:repository_macro build/repositories.bzl%go_repositories
go_repositories()

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

go_rules_dependencies()

go_register_toolchains(
    version = "1.15.8",
)

gazelle_dependencies()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

# Latest images as of 2020-02-10

# debug-nonroot-amd64 from https://console.cloud.google.com/gcr/images/distroless/GLOBAL/base-debian10
container_pull(
    name = "go_debug_image_base",
    digest = "sha256:772350b3f38595992fca3f62e6f43b3af932a04e8f2082822d45c211ba730927",
    registry = "gcr.io",
    repository = "distroless/base-debian10",
)

# nonroot-amd64 from https://console.cloud.google.com/gcr/images/distroless/GLOBAL/static-debian10
container_pull(
    name = "go_image_static",
    digest = "sha256:6d7e84ea3d305c4300e62187d5a0f102bab0c758b7fb637ad0911d10e2fcc741",
    registry = "gcr.io",
    repository = "distroless/static-debian10",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@com_github_atlassian_bazel_tools//buildozer:deps.bzl", "buildozer_dependencies")
load("@com_github_atlassian_bazel_tools//multirun:deps.bzl", "multirun_dependencies")
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//go:image.bzl",
    go_image_repositories = "repositories",
)
load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")
load("@rules_proto_grpc//:repositories.bzl", "rules_proto_grpc_toolchains")
load("@rules_proto_grpc//go:repositories.bzl", rules_proto_grpc_go_repos = "go_repos")
load("@com_github_envoyproxy_protoc_gen_validate//:dependencies.bzl", pgv_third_party = "go_third_party")
load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

go_image_repositories()

buildifier_dependencies()

buildozer_dependencies()

multirun_dependencies()

rules_proto_dependencies()

rules_proto_toolchains()

rules_proto_grpc_toolchains()

rules_proto_grpc_go_repos()

pgv_third_party()

rules_pkg_dependencies()
