# See https://github.com/rules-proto-grpc/rules_proto_grpc#developing-custom-plugins
# See https://github.com/rules-proto-grpc/rules_proto_grpc/issues/16

load("@rules_proto_grpc//:plugin.bzl", "ProtoPluginInfo")
load(
    "@rules_proto_grpc//:aspect.bzl",
    "ProtoLibraryAspectNodeInfo",
    "proto_compile_aspect_attrs",
    "proto_compile_aspect_impl",
    "proto_compile_attrs",
    "proto_compile_impl",
)

# Create aspect
go_proto_validate_aspect = aspect(
    implementation = proto_compile_aspect_impl,
    provides = [ProtoLibraryAspectNodeInfo],
    attr_aspects = ["deps"],
    attrs = dict(
        proto_compile_aspect_attrs,
        _plugins = attr.label_list(
            doc = "List of protoc plugins to apply",
            providers = [ProtoPluginInfo],
            default = [
                Label("//build:go_validate_plugin"),
            ],
        ),
        _prefix = attr.string(
            doc = "String used to disambiguate aspects when generating outputs",
            default = "go_proto_validate_aspect",
        ),
    ),
    toolchains = ["@rules_proto_grpc//protobuf:toolchain_type"],
)

# Create compile rule to apply aspect
_rule = rule(
    implementation = proto_compile_impl,
    attrs = dict(
        proto_compile_attrs,
        deps = attr.label_list(
            mandatory = True,
            providers = [ProtoInfo, ProtoLibraryAspectNodeInfo],
            aspects = [go_proto_validate_aspect],
        ),
    ),
)

# Create macro for converting attrs and passing to compile
def go_validate_compile(**kwargs):
    _rule(
        verbose_string = "{}".format(kwargs.get("verbose", 0)),
        merge_directories = True,
        **{k: v for k, v in kwargs.items() if k != "merge_directories"}
    )
