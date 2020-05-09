def extract_generated(name, label, root_relative_directory):
    native.sh_binary(
        name = name,
        srcs = ["//build:copy.sh"],
        data = [label],
        args = ["$(location %s)" % label, root_relative_directory],
    )
