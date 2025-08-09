#!/usr/bin/env python3
#
# SPDX-FileCopyrightText: 2025 Buoyant Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Copyright 2025 Buoyant Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.  You may obtain
# a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import ClassVar

import sys

import yaml
import argparse
import os


class ImageStyle:
    def __init__(self, imagesuffix, dockerfile, build_args=None):
        self.imagesuffix = imagesuffix
        self.dockerfile = dockerfile
        self.build_args = build_args


class BuildStyle:
    ARCHITECTURES: ClassVar[list[str]] = [ "arm64", "amd64" ]

    def __init__(self, name, image_styles, architectures=None):
        if architectures is None:
            architectures = BuildStyle.ARCHITECTURES

        self.name = name
        self.architectures = architectures
        self.image_styles = image_styles


class Build:
    def __init__(self,
                 name,
                 build_styles=None,
                 extra_files=None,
    ):
        self.name = name
        self.build_styles = build_styles
        self.extra_files = extra_files


def SimpleBuild(name, binary=None):
    if binary is None:
        binary = name

    image_styles = [
        ImageStyle(binary, name)
    ]

    build_styles = [
        BuildStyle(binary, image_styles),
    ]

    return Build(name, build_styles=build_styles)


def DockerOnlyBuild(name):
    image_styles = [
        ImageStyle(None, name),
    ]

    build_styles = [
        BuildStyle(None, image_styles),
    ]

    return Build(name, build_styles=build_styles, extra_files=[ f"docker/{name}" ])

def main():
    parser = argparse.ArgumentParser(description="Generate gorel YAML definitions.")
    parser.add_argument("--arch", type=str, default="arm64,amd64", help="Comma-separated architectures to use (default: %(default)s)")
    parser.add_argument("--header", type=str, help="File to print before YAML output")
    parser.add_argument("--footer", type=str, help="File to print after YAML output")
    args = parser.parse_args()

    # Update default architectures globally

    BuildStyle.ARCHITECTURES = args.arch.split(",")

    # Rebuild BUILDS with possibly new architectures
    builds = [
        Build("emissary",
                build_styles=[
                    BuildStyle("busyambassador",
                                image_styles=[
                                    ImageStyle(
                                        "",
                                        "emissary",
                                        build_args=[
                                            "--build-arg=ENVOY_IMAGE={{ .Env.ENVOY_IMAGE }}",
                                        ]
                                    ),
                                ],
                    ),
                ],
                extra_files=[ "LICENSE", "python", "pyproject.toml" ],
        ),
        SimpleBuild("apiext"),
        SimpleBuild("kat-client"),
        SimpleBuild("kat-server"),
        DockerOnlyBuild("test-auth"),
        DockerOnlyBuild("test-shadow"),
        DockerOnlyBuild("test-stats"),
    ]

    build_defs = {}
    docker_defs = []
    manifest_defs = []

    for build in builds:
        for build_style in build.build_styles:
            style_name = build_style.name

            build_id = style_name

            if style_name is not None:
                build_def = {
                    "id": build_id,
                    "main": f"./cmd/{style_name}",
                    "binary": style_name,
                    "env": [ "CGO_ENABLED=0" ],
                    "goos": [ "linux" ],
                    "goarch": list(build_style.architectures),
                }

                build_defs[build_id] = build_def

            for image in build_style.image_styles:
                dockerfile = f"docker/{image.dockerfile}/Dockerfile"
                name_env = image.dockerfile.replace("-", "_").upper() + "_IMAGE"
                image_name = "{{ .Env.REGISTRY }}/{{ .Env.%s }}" % name_env

                currents = []
                latests = []

                for arch in build_style.architectures:
                    libarch = None

                    if arch == "arm64":
                        libarch = "aarch64"
                    elif arch == "amd64":
                        libarch = "x86_64"

                    if libarch is None:
                        raise ValueError(f"Unsupported architecture for {build.name}: {arch}")

                    current_image = "%s:{{ .Version }}-%s" % (image_name, arch)
                    latest_image = "%s:latest-%s" % (image_name, arch)

                    currents.append(current_image)
                    latests.append(latest_image)

                    build_flags = [
                        f"--platform=linux/{arch}",
                        f"--build-arg=LIBARCH={libarch}",
                    ]

                    if image.build_args:
                        build_flags.extend(image.build_args)

                    docker_def = {
                        "use": "buildx",
                        "goos": "linux",
                        "goarch": arch,
                        "dockerfile": dockerfile,
                        "image_templates": [ current_image, latest_image ],
                        "build_flag_templates": build_flags,
                    }

                    if build_id is not None:
                        docker_def["ids"] = [ build_id ]

                    if build.extra_files:
                        docker_def["extra_files"] = list(build.extra_files)

                    docker_defs.append(docker_def)

                current_manifest_name = "%s:{{ .Version }}" % image_name
                latest_manifest_name = "%s:latest" % image_name

                current_manifest_def = {
                    "name_template": current_manifest_name,
                    "image_templates": currents,
                    "create_flags": [ "--insecure" ],
                    "push_flags": [ "--insecure" ],
                }

                latest_manifest_def = {
                    "name_template": latest_manifest_name,
                    "image_templates": latests,
                    "create_flags": [ "--insecure" ],
                    "push_flags": [ "--insecure" ],
                }

                manifest_defs.append(current_manifest_def)
                manifest_defs.append(latest_manifest_def)

    gorel = {
        "builds": list(build_defs.values()),
        "dockers": docker_defs,
        "docker_manifests": manifest_defs,
    }

    if args.header:
        with open(args.header, "r") as f:
            print(f.read(), end="")

    print(yaml.safe_dump(gorel))

    if args.footer:
        with open(args.footer, "r") as f:
            print(f.read(), end="")

if __name__ == "__main__":
    main()
