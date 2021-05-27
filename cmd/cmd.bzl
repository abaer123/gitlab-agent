"""
Macros for cmd.
"""

load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_bundle")
load("@io_bazel_rules_docker//contrib:push-all.bzl", "container_push")

def push_bundle(name, images):
    bundle_name = name + "_bundle"
    container_bundle(
        name = bundle_name,
        images = images,
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )
    container_push(
        name = name,
        bundle = ":" + bundle_name,
        format = "Docker",
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )

def define_command_targets(
        name,
        binary_embed,
        race_targets = True,
        base_image = "@go_image_static//image",
        base_image_race = "@go_debug_image_base//image"):
    x_defs = {
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd.Version": "{STABLE_BUILD_GIT_TAG}",
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd.Commit": "{STABLE_BUILD_GIT_COMMIT}",
        "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/cmd.BuildTime": "{BUILD_TIME}",
    }
    go_binary(
        name = name,
        embed = binary_embed,
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

    go_image(
        name = "container",
        base = base_image,
        binary = ":%s_linux" % name,
        tags = ["manual"],
        visibility = ["//visibility:public"],
    )

    if race_targets:
        go_binary(
            name = "%s_race" % name,
            embed = binary_embed,
            race = "on",
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

        go_image(
            name = "container_race",
            base = base_image_race,
            binary = ":%s_linux_race" % name,
            tags = ["manual"],
            visibility = ["//visibility:public"],
        )
