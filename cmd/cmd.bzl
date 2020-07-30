"""
Macros for cmd.
"""

load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_push")
load("//build:build.bzl", "copy_absolute")

def define_command_targets(name, binary_embed):
    go_binary(
        name = name,
        embed = binary_embed,
        visibility = ["//visibility:public"],
    )

    go_binary(
        name = "%s_race" % name,
        embed = binary_embed,
        race = "on",
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )

    go_binary(
        name = "%s_linux" % name,
        embed = binary_embed,
        goarch = "amd64",
        goos = "linux",
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )

    go_binary(
        name = "%s_linux_race" % name,
        embed = binary_embed,
        goarch = "amd64",
        goos = "linux",
        race = "on",
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )

    copy_absolute(
        name = "extract_%s" % name,
        label = ":%s" % name,
        file_to_copy = name,
    )

    copy_absolute(
        name = "extract_%s_race" % name,
        label = ":%s_race" % name,
        file_to_copy = "%s_race" % name,
    )

    go_image(
        name = "container",
        base = "@go_image_static//image",
        binary = ":%s_linux" % name,
        tags = ["manual"],
    )

    # Use CC base image here to have libstdc++6 installed because it is
    # needed for race detector to work https://github.com/golang/go/issues/14481
    # Otherwise getting:
    # error while loading shared libraries: libstdc++.so.6: cannot open shared object file: No such file or directory
    go_image(
        name = "container_race",
        base = "@cc_image_base//image",
        binary = ":%s_linux_race" % name,
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_tag",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_tag_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}-race",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_commit",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_COMMIT}",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_commit_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_COMMIT}-race",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_latest",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "latest",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_latest_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "latest-race",
        tags = ["manual"],
    )
