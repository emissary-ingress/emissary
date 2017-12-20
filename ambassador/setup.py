import setuptools
from setuptools import setup, find_packages

import os

from ambassador.VERSION import Version

print("setuptools %s" % setuptools.__version__)

requirements = open("requirements.txt", "r").read().split("\n")

def collect_data_files(dirpath):
    return [
        (subdirpath,
         [ os.path.join(subdirpath, filename) 
           for filename in filenames ])
        for subdirpath, folders, filenames in os.walk(dirpath)
    ]

template_files = collect_data_files("templates")
schema_files = collect_data_files("schemas")

data_files = template_files + schema_files

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
        'console_scripts': [
            'ambassador=ambassador.cli:main',
            'diagd=ambassador_diag.diagd:main'
      ]
    },

    author="datawire.io",
    author_email="dev@datawire.io",
    url="https://www.getambassador.io",
    download_url="https://github.com/datawire/ambassador",
    keywords=[
        "kubernetes",
        "microservices",
        "api gateway",
        "envoy",
        "ambassador"
    ],
    classifiers=[],
)
