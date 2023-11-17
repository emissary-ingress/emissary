# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import os

from setuptools import find_packages, setup

# from ambassador.VERSION import Version
Version = "0.0.0-dev"

requirements = open("requirements.txt", "r").read().split("\n")


def collect_data_files(dirpath):
    return [
        (subdirpath, [os.path.join(subdirpath, filename) for filename in filenames])
        for subdirpath, folders, filenames in os.walk(dirpath)
    ]


template_files = collect_data_files("templates")
schema_files = collect_data_files("schemas")
kat_files = [
    (
        subdirpath,
        [os.path.join(subdirpath, filename) for filename in filenames if filename.endswith("go")],
    )
    for subdirpath, folders, filenames in os.walk("kat")
]

data_files = [("", ["ambassador.version"])] + template_files + schema_files + kat_files

setup(
    name="ambassador",
    # version=versioneer.get_version(),
    # cmdclass=versioneer.get_cmdclass(),
    version=Version,
    packages=find_packages(exclude=["tests"]),
    # include_package_data=True,
    install_requires=requirements,
    data_files=data_files,
    entry_points={
        "console_scripts": [
            "ambassador=ambassador_cli.ambassador:main",
            "diagd=ambassador_diag.diagd:main",
            "grab-snapshots=ambassador_cli.grab_snapshots:main",
            "ert=ambassador_cli.ert:main",
        ]
    },
    author="datawire.io",
    author_email="dev@datawire.io",
    url="https://www.getambassador.io",
    download_url="https://github.com/datawire/ambassador",
    keywords=["kubernetes", "microservices", "api gateway", "envoy", "ambassador"],
    classifiers=[],
)
