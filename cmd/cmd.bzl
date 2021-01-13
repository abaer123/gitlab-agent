"""
Macros for cmd.
"""

load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_image", "container_push")
load("//build:build.bzl", "copy_absolute")

def define_command_targets(name, binary_embed):
    x_defs = {
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Version": "{STABLE_BUILD_GIT_TAG}",
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.Commit": "{STABLE_BUILD_GIT_COMMIT}",
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd.BuildTime": "{BUILD_TIME}",
    }
    go_binary(
        name = name,
        embed = binary_embed,
        visibility = ["//visibility:public"],
        x_defs = x_defs,
    )

    go_binary(
        name = "%s_race" % name,
        embed = binary_embed,
        race = "on",
        tags = ["manual"],
        visibility = ["//visibility:public"],
        x_defs = x_defs,
    )

    go_binary(
        name = "%s_linux" % name,
        embed = binary_embed,
        goarch = "amd64",
        goos = "linux",
        tags = ["manual"],
        visibility = ["//visibility:public"],
        x_defs = x_defs,
    )

    go_binary(
        name = "%s_linux_race" % name,
        embed = binary_embed,
        goarch = "amd64",
        goos = "linux",
        race = "on",
        tags = ["manual"],
        visibility = ["//visibility:public"],
        x_defs = x_defs,
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

    container_image(
        name = "nonroot_go_image_static",
        base = "@go_image_static//image",
        user = "nonroot",
        tags = ["manual"],
    )

    container_image(
        name = "nonroot_go_debug_image_base",
        base = "@go_debug_image_base//image",
        user = "nonroot",
        tags = ["manual"],
    )

    go_image(
        name = "container",
        base = ":nonroot_go_image_static",
        binary = ":%s_linux" % name,
        tags = ["manual"],
    )

    go_image(
        name = "container_race",
        base = ":nonroot_go_debug_image_base",
        binary = ":%s_linux_race" % name,
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_tag",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_tag_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}-race",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_commit",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "{STABLE_BUILD_GIT_COMMIT}",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_commit_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "{STABLE_BUILD_GIT_COMMIT}-race",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_latest",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "latest",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_latest_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "latest-race",
        tags = ["manual"],
    )
