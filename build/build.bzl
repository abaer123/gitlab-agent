load("@com_github_ash2k_bazel_tools//multirun:def.bzl", "command")
load("@bazel_skylib//lib:shell.bzl", "shell")
load("@io_bazel_rules_go//go:def.bzl", "go_test")

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

# go_custom_test is a macro around go_test that sets size="small" and race="on" if these
# arguments are not set explicitly.
def go_custom_test(size = "small", race = "on", **kwargs):
    go_test(
        size = size,
        race = race,
        **kwargs
    )
