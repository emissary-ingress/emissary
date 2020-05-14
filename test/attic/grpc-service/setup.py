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

from setuptools import setup, find_packages

setup(
    name="grpc-basic",
    version="head",
    packages=find_packages(exclude=["tests"]),
    include_package_data=True,
    install_requires=[
        "grpcio",
        "grpcio-tools"
    ],
    author="datawire.io",
    author_email="dev@datawire.io",
    url="https://github.com/datawire/ambassador-examples/grpc-basic",
    download_url="https://github.com/datawire/ambassador-examples/grpc-basic",
    keywords=[],
    classifiers=[],
)
