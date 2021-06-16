workspace(name = "gitlab_k8s_agent")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# When updating rules_go make sure to update org_golang_x_tools dependency below by copying it from
# https://github.com/bazelbuild/rules_go/blob/master/go/private/repositories.bzl
# Also update to the same version/commit in go.mod.
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "54d978734b7cbdcaa5cc4eece4ca4053d8b49972ae04a9fdb0ef05cfe812e524",
    strip_prefix = "rules_docker-0adf8b2ff23e8d7a14562be0f5707cd4dbb32998",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/0adf8b2ff23e8d7a14562be0f5707cd4dbb32998.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "c28eef4d30ba1a195c6837acf6c75a4034981f5b4002dda3c5aa6e48ce023cf1",
    strip_prefix = "buildtools-4.0.1",
    urls = ["https://github.com/bazelbuild/buildtools/archive/4.0.1.tar.gz"],
)

http_archive(
    name = "com_github_ash2k_bazel_tools",
    sha256 = "adc624afa929abf56ffa7be34177e74ffa60e87d7820d781ca2128b3f238d91d",
    strip_prefix = "bazel-tools-cbe7710fca61d5cc585af4ea29b0e1423e1ac17d",
    urls = ["https://github.com/ash2k/bazel-tools/archive/cbe7710fca61d5cc585af4ea29b0e1423e1ac17d.tar.gz"],
)

http_archive(
    name = "rules_proto",
    sha256 = "9fc210a34f0f9e7cc31598d109b5d069ef44911a82f507d5a88716db171615a8",
    strip_prefix = "rules_proto-f7a30f6f80006b591fa7c437fe5a951eb10bcbcf",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/f7a30f6f80006b591fa7c437fe5a951eb10bcbcf.tar.gz",
    ],
)

http_archive(
    name = "rules_proto_grpc",
    sha256 = "fa7a59e0d1527ac69be652407b457ba1cb40700752a3ee6cc2dd25d9cb28bb1a",
    strip_prefix = "rules_proto_grpc-3.1.0",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/archive/3.1.0.tar.gz"],
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
    sha256 = "038f1caa773a7e35b3663865ffb003169c6a71dc995e39bf4815792f385d837d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_pkg/releases/download/0.4.0/rules_pkg-0.4.0.tar.gz",
        "https://github.com/bazelbuild/rules_pkg/releases/download/0.4.0/rules_pkg-0.4.0.tar.gz",
    ],
)

git_repository(
    name = "bazelruby_rules_ruby",
    commit = "b738b9885f3fba30cbbb6fc05f23ae7a78a4ca84",
    remote = "https://github.com/bazelruby/rules_ruby.git",
    shallow_since = "1619209813 -0700",
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
    sha256 = "1efbc55de3ca1211fe4c0afc559f2edbded30ed3095d94dd602311bf604b3fd7",
    urls = ["http://ftp.debian.org/debian/pool/main/g/git/git_2.30.2-1_amd64.deb"],
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

# It's here to add build tags
go_repository(
    name = "com_gitlab_gitlab_org_labkit",
    build_file_proto_mode = "disable_global",
    # The same list of go build tags must be in four places:
    # - Makefile
    # - Workspace
    # - .bazelrc
    # - .golangci.yml
    build_tags = [
        "tracer_static",
        "tracer_static_jaeger",
    ],  # keep
    importpath = "gitlab.com/gitlab-org/labkit",
    sum = "h1:mh4g+c/esQSWCVQJ197aftrIbXlhXCtiLC8RgSvyCx0=",
    version = "v1.4.1",
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
    sum = "h1:4CF52PCseTFt4bE+Yk3dIpdVi7XWuPVMhPtm4FaIJPM=",
    version = "v0.6.1",
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
    sha256 = "60a5cee8304b4d9130344f156a10ba648e315b5fca4b84939b765b26ce217dee",
    strip_prefix = "tools-0.1.0",
    # v0.1.0, as of 2021-03-17
    urls = [
        "https://mirror.bazel.build/github.com/golang/tools/archive/v0.1.0.zip",
        "https://github.com/golang/tools/archive/v0.1.0.zip",
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
    version = "1.16.3",
)

gazelle_dependencies()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")

# Latest images as of 2020-02-10

# debug-nonroot-amd64 from https://console.cloud.google.com/gcr/images/distroless/GLOBAL/base-debian10
container_pull(
    name = "go_debug_image_base",
    digest = "sha256:565adbedbfed0048c30dd819fdf0cc0d8fd9c2ca466a32b6a9367ff156058a8e",
    registry = "gcr.io",
    repository = "distroless/base-debian10",
)

# nonroot-amd64 from https://console.cloud.google.com/gcr/images/distroless/GLOBAL/static-debian10
container_pull(
    name = "go_image_static",
    digest = "sha256:8f3b47c7984464f417f9d5f5e232ac3fae6453e84f053724fef457c4ba67ceaf",
    registry = "gcr.io",
    repository = "distroless/static-debian10",
)

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")
load("@com_github_ash2k_bazel_tools//buildozer:deps.bzl", "buildozer_dependencies")
load("@com_github_ash2k_bazel_tools//multirun:deps.bzl", "multirun_dependencies")
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

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")

grpc_deps()

pgv_third_party()

rules_pkg_dependencies()
