"""
Macros for cmd.
"""

load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_go//go:def.bzl", "go_binary")
load("@io_bazel_rules_docker//container:container.bzl", "container_push")

def define_command_targets(
        name,
        binary_embed,
        race_targets = True,
        base_image = "@go_image_static//image",
        base_image_race = "@go_debug_image_base//image"):
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
        name = "push_docker_commit",
        format = "Docker",
        image = ":container",
        registry = "registry.gitlab.com",
        repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
        tag = "{STABLE_BUILD_GIT_COMMIT}",
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
            name = "push_docker_commit_race",
            format = "Docker",
            image = ":container_race",
            registry = "registry.gitlab.com",
            repository = "{STABLE_CONTAINER_REPOSITORY_PATH}/%s" % name,
            tag = "{STABLE_BUILD_GIT_COMMIT}-race",
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
