load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_proto_grpc//go:defs.bzl", "go_grpc_compile")
load("//build:build.bzl", "copy_to_workspace")
load("@bazel_skylib//lib:paths.bzl", "paths")
load("//build:validate.bzl", "go_validate_compile")

def go_grpc_generate(src, workspace_relative_target_directory, deps = []):
    proto_library(
        name = "proto",
        srcs = [src],
        tags = ["manual"],
        visibility = ["//visibility:public"],
        deps = deps,
    )

    go_grpc_compile(
        name = "grpc_compile",
        tags = ["manual"],
        deps = [":proto"],
    )

    copy_to_workspace(
        name = "extract_generated",
        file_to_copy = paths.split_extension(src)[0] + ".pb.go",
        label = ":grpc_compile",
        workspace_relative_target_directory = workspace_relative_target_directory,
    )

    for dep in deps:
        if dep == "@com_github_envoyproxy_protoc_gen_validate//validate:validate_proto":
            go_validate_compile(
                name = "proto_compile_validate",
                tags = ["manual"],
                deps = [":proto"],
            )
            copy_to_workspace(
                name = "extract_generated_validator",
                file_to_copy = paths.split_extension(src)[0] + ".pb.validate.go",
                label = ":proto_compile_validate",
                workspace_relative_target_directory = workspace_relative_target_directory,
            )
            break
