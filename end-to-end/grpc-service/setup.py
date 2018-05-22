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
