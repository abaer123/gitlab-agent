def extract_generated(name, label, file_to_copy, root_relative_directory):
    native.sh_binary(
        name = name,
        srcs = ["//build:copy.sh"],
        data = [label],
        tags = ["manual"],
        args = [root_relative_directory, file_to_copy, "$(locations %s)" % label],
    )
