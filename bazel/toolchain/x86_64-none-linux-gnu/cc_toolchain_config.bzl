"""
C++ Toolchain Configuration for x86_64 (Native x86_64)

This file defines the C++ toolchain for native x86_64 (Intel/AMD 64-bit) builds on Linux.

Architecture: x86_64 (Native Intel/AMD 64-bit)
Tools: gcc, g++ (native system compiler)
OS Support: CentOS Stream 9 (GCC 12.x)

Related Files:
  - cc_toolchain_config_cs10.bzl: CentOS Stream 10 variant (GCC 14.x)
  - ../aarch64-none-linux-gnu/cc_toolchain_config.bzl: ARM64 cross-compile variant
  - ../s390x-none-linux-gnu/cc_toolchain_config.bzl: IBM System z cross-compile variant
  - BUILD: Defines cc_toolchain rule that uses this config
  - ../toolchain.bzl: Registers all toolchains

Note: This is the native/default toolchain. The file structure matches aarch64 and s390x
variants for consistency, even though x86_64 does not require cross-compiler tools.
"""

load("@bazel_tools//tools/build_defs/cc:action_names.bzl", "ACTION_NAMES")
load(
    "@bazel_tools//tools/cpp:cc_toolchain_config_lib.bzl",
    "feature",
    "flag_group",
    "flag_set",
    "tool_path",
)

all_link_actions = [
    ACTION_NAMES.cpp_link_executable,
    ACTION_NAMES.cpp_link_dynamic_library,
    ACTION_NAMES.cpp_link_nodeps_dynamic_library,
]


all_compile_actions = [
    ACTION_NAMES.assemble,
    ACTION_NAMES.c_compile,
    ACTION_NAMES.clif_match,
    ACTION_NAMES.cpp_compile,
    ACTION_NAMES.cpp_header_parsing,
    ACTION_NAMES.cpp_module_codegen,
    ACTION_NAMES.cpp_module_compile,
    ACTION_NAMES.linkstamp_compile,
    ACTION_NAMES.lto_backend,
    ACTION_NAMES.preprocess_assemble,
]

def _impl(ctx):
    # Tool paths for native x86_64 builds.
    # These point to the standard system compiler tools (gcc, g++, etc).
    # Path: /usr/bin/* (standard system compilers)
    tool_paths = [
        tool_path(
            name = "ar",
            path = "/usr/bin/ar",
        ),
        tool_path(
            name = "cpp",
            path = "/usr/bin/cpp",
        ),
        tool_path(
            name = "gcc",
            path = "/usr/bin/gcc",
        ),
        tool_path(
            name = "gcov",
            path = "/usr/bin/gcov",
        ),
        tool_path(
            name = "ld",
            path = "/usr/bin/ld",
        ),
        tool_path(
            name = "nm",
            path = "/usr/bin/nm",
        ),
        tool_path(
            name = "objdump",
            path = "/usr/bin/objdump",
        ),
        tool_path(
            name = "strip",
            path = "/usr/bin/strip",
        ),
    ]

    # Update the value of __TOOLCHAIN_SYSROOT__ every time the list
    # of cxx_builtin_include_directories has been modified.
    #
    # This is needed to make bazel realize that the updated toolchain
    # is different from the previous one and handle caching correctly.
    #
    # https://github.com/kubevirt/kubevirt/pull/8404#issuecomment-1275096374
    default_compiler_flags = feature(
        name = "default_compiler_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = all_compile_actions,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-no-canonical-prefixes",
                            "-fno-canonical-system-headers",
                            "-Wno-builtin-macro-redefined",
                            "-D__DATE__=\"redacted\"",
                            "-D__TIMESTAMP__=\"redacted\"",
                            "-D__TIME__=\"redacted\"",
                            "-D__TOOLCHAIN_SYSROOT__=\"centos-stream-9\"",
                        ],
                    ),
                ],
            ),
        ],
    )

    default_linker_flags = feature(
        name = "default_linker_flags",
        enabled = False,
        flag_sets = [
            flag_set(
                actions = all_link_actions,
                flag_groups = ([
                    flag_group(
                        flags = [
                            "",
                        ],
                    ),
                ]),
            ),
        ],
    )

    features = [
        default_compiler_flags,
        default_linker_flags,
    ]

    return cc_common.create_cc_toolchain_config_info(
        ctx = ctx,
        cxx_builtin_include_directories = [
            "/usr/lib/gcc/x86_64-redhat-linux/11/include",
            "/usr/include",
        ],
        features = features,
        toolchain_identifier = "x86_64-toolchain",
        host_system_name = "local",
        target_system_name = "unknown",
        target_cpu = "unknown",
        target_libc = "unknown",
        compiler = "unknown",
        abi_version = "unknown",
        abi_libc_version = "unknown",
        tool_paths = tool_paths,
    )

cc_toolchain_config = rule(
    implementation = _impl,
    attrs = {},
    provides = [CcToolchainConfigInfo],
)
