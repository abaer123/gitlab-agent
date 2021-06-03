load("//build:build.bzl", "copy_to_workspace")

def copy_grpc_files(module_name, label):
    copy_to_workspace(
        name = "extract_%s_rpc_pb" % module_name,
        file_to_copy = "rpc_pb.rb",
        label = label,
        workspace_relative_target_directory = "pkg/ruby/lib/internal/module/%s/rpc" % module_name,
    )

    copy_to_workspace(
        name = "extract_%s_rpc_services_pb" % module_name,
        file_to_copy = "rpc_services_pb.rb",
        label = label,
        workspace_relative_target_directory = "pkg/ruby/lib/internal/module/%s/rpc" % module_name,
    )
