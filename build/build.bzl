load("@com_github_atlassian_bazel_tools//multirun:def.bzl", "command")
load("@bazel_skylib//lib:shell.bzl", "shell")

def copy_to_workspace(name, label, file_to_copy, workspace_relative_target_directory):
    command(
        name = name,
        command = "//build:copy_to_workspace",
        data = [label],
        arguments = ["$(rootpaths %s)" % label, file_to_copy, workspace_relative_target_directory],
        visibility = ["//visibility:public"],
    )

# This macro expects target directory for the file as an additional command line argument.
def copy_absolute(name, label, file_to_copy):
    command(
        name = name,
        command = "//build:copy_absolute",
        data = [label],
        arguments = ["$(rootpaths %s)" % label, file_to_copy],
        visibility = ["//visibility:public"],
    )
