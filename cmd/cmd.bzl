"""
Macros for cmd.
"""

load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_push")

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

    go_image(
        name = "container",
        binary = ":%s_linux" % name,
        tags = ["manual"],
    )

    go_image(
        name = "container_race",
        binary = ":%s_linux_race" % name,
        tags = ["manual"],
    )

    container_push(
        name = "push_docker",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}-{STABLE_BUILD_GIT_COMMIT}",
        tags = ["manual"],
    )

    container_push(
        name = "push_docker_race",
        format = "Docker",
        image = ":container_race",
        registry = "registry.gitlab.com",
        repository = "gitlab-org/cluster-integration/gitlab-agent/%s" % name,
        tag = "{STABLE_BUILD_GIT_TAG}-{STABLE_BUILD_GIT_COMMIT}-race",
        tags = ["manual"],
    )
