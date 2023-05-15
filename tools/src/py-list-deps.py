#!/usr/bin/env python3
# -*- fill-column: 70 -*-

# Copyright 2020, 2022 Datawire. All rights reserved.
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

# This script looks at Python3 source files in a directory, and parses
# them to return a list of distributions that those Python sources
# depend on.
#
# A quick refresher on PEP 561 terminology:
#
# - "distribution": a bundle that gets shared around (eggs and
#   wheels are implementations of "distribution")
#
# - "module": a thing you can `import` (could be implemented either as
#   `MOD.py` or as `MOD/__init__.py`)
#
# - "package": a module that contains more modules within it (a
#   `MOD/__init__.py`)

# This script started life as a script that generates Bazel 'BUILD'
# files, and was more-or-less a clone of
# <https://github.com/tuomasr/pazel>.  Why we wrote it, instead of
# just using Pazel:
#
#  1. Pazel doesn't actually work; with Python 3.8 on GNU/Linux, the
#     _is_in_stdlib function raises an exception
#  2. Pazel doesn't understand relative imports.
#  3. Pazel is too configurable; the configuration is complex enough
#     that it's simpler/easier/faster to just have a purpose-built
#     script that you can modify.
#
# And a good thing too; having a purpose-built script that we can
# modify meant that we can leverage it to be useful without Bazel.

import argparse
import ast
import importlib
import os
import re
import sys
import sysconfig
from typing import List, NamedTuple, Optional, Set

DEV_PY_FILES = r'|'.join(['setup.py', 'conftest.py', '/tests/', '/kat/', 'test_.*.py',])

def parse_members(filepath: str) -> Set[str]:
    """parse_members parses a .py file and returns a set of all of the
    symbols exposed by file; whether they come from that file itself
    or from its imports.

    For example, given a file containing

        from foo import bar

        def baz():
            pass

    parse_members would return {'bar', 'baz'}.

    In concept, parse_members should also support things like classes
    and variables, but because this script only ever calls
    parse_members on `__init__.py`, and `__init__.py` files tend to be
    pretty minimal, I got lazy and only implemented support for
    imports and functions.  It would be good to support all of the
    statement types, but imports and function definitions are the only
    statement types that Emissary uses in any `__init__.py` files, so
    this is good enough.

    """
    with open(filepath, 'r') as filehandle:
        filecontent = filehandle.read()

    members = set()
    for node in ast.parse(filecontent).body:
        # XXX: This doesn't recognize all the the statement types,
        # just the ones that Emissary currently uses in __init__.py
        # files.
        if isinstance(node, ast.Import) or  isinstance(node, ast.ImportFrom):
            members.update(alias.asname or alias.name for alias in node.names)
        elif isinstance(node, ast.FunctionDef):
            members.update([node.name])
    return members

class ImportedItem(NamedTuple):
    """
    import mod             → ImportedItem(mod, None)
    from pkg import mod    → ImportedItem(pkg, mod)
    from mod import member → ImportedItem(mod, member)
    """
    module: str
    member: Optional[str]

def parse_imports(filepath: str) -> List[ImportedItem]:
    """parse_imports parses a .py file and returns a list of all of the
    things that the file imports.

    """
    with open(filepath, 'r') as filehandle:
        filecontent = filehandle.read()

    imports = []
    for node in ast.parse(filecontent, filename=filepath).body:
        if isinstance(node, ast.Import):
            #     import {node.names}
            imports += [ImportedItem(alias.name, None) for alias in node.names]
        elif isinstance(node, ast.ImportFrom):
            #     from {'.'*node.level}{node.module} import {node.names}
            modname = ('.'*(node.level or 0)) + (node.module or '')
            imports += [ImportedItem(modname, alias.name) for alias in node.names]
    return imports

def dirpath_is_in_stdlib(dirpath: str) -> bool:
    # Where isort does a similar thing, they have a comment saying the
    # calls to 'os.path.normcase' are important on Windows.  Not that
    # we expect this to work on Windows without more modification.

    # The calls to 'os.path.realpath' are important on macOS because
    # `brew install`d Python will have
    # `sysconfig.get_paths()['stdlib']` start off
    #   "/usr/local/opt/python@3.9/Frameworks"
    # while the entries in `sys.path` start off
    #   "/usr/local/Cellar/python@3.9/3.9.10/Frameworks"
    # so we need to resolve symlinks for them to be comparable.

    dirpath = os.path.normcase(os.path.realpath(dirpath))

    if ('site-packages' in dirpath) or ('dist-packages' in dirpath):
        return False

    stdlib_prefix = os.path.normcase(os.path.realpath(sysconfig.get_paths()['stdlib']))
    if dirpath.startswith(stdlib_prefix):
        return True

    return False

def is_in_stdlib(item: ImportedItem) -> bool:
    if item.module.startswith('.'):
        return False

    # This function works by temporarily removing everything except
    # for the stdlib from `sys.path`, then `try`ing to import the
    # thing; if the import succeeds we set in_stdlib=True; if the
    # import throws an ImportError or an AttributeError then we know
    # that it's not in stdlib.

    original_sys_path = sys.path
    sys.path = [d for d in sys.path if dirpath_is_in_stdlib(d)]

    in_stdlib = False
    try:
        module = importlib.import_module(item.module)
        # If the import was of the form `from module import member`,
        # then importlib.import_module has only checked that `module`
        # is in stdlib, so now we need to check that it contains
        # `member`.
        if item.member:
            getattr(module, item.member)
        in_stdlib = True
    except (ImportError, AttributeError):
        pass

    sys.path = original_sys_path
    return in_stdlib

def is_local(localpaths: List[str], item: ImportedItem) -> bool:
    if item.module.startswith('.'):
        return True

    if import_to_filepath(localpaths, item) is not None:
        return True

    return False

def isfile_case(filepath: str) -> bool:
    """Like os.path.isfile, but requires the case of the basename to match
    (relevent on macOS case-insensitive filesystem).
    """
    return os.path.isfile(filepath) and (os.path.basename(filepath) in os.listdir(os.path.dirname(filepath)))

def import_to_filepath(localpaths: List[str], item: ImportedItem, fromfilepath: Optional[str]=None) -> Optional[str]:
    modname = item.module
    if modname.startswith('.'):
        if not fromfilepath:
            raise Exception("this should not happen")
        fromdir = fromfilepath
        while modname.startswith('.'):
            fromdir = os.path.dirname(fromdir)
            modname = modname[1:]
        localpaths = [fromdir]

    for dirpath in localpaths:
        basefilepath = os.path.join(dirpath, *modname.split('.'))
        if isfile_case(basefilepath+".py"):
            return basefilepath+".py"
        if os.path.isfile(os.path.join(basefilepath, '__init__.py')):
            if item.member is None:
                return os.path.join(basefilepath, '__init__.py')
            if item.member in parse_members(os.path.join(basefilepath, '__init__.py')):
                return os.path.join(basefilepath, '__init__.py')
            if isfile_case(os.path.join(basefilepath, item.member+'.py')):
                return os.path.join(basefilepath, item.member+'.py')

    return None

def mod_startswith(modname: str, prefix: str) -> bool:
    return (modname == prefix) or modname.startswith(prefix+'.')

def import_to_distribname(item: ImportedItem) -> str:
    if mod_startswith(item.module, 'scout'):
        return 'scout.py'
    elif mod_startswith(item.module, 'yaml'):
        return 'pyyaml'
    elif mod_startswith(item.module, 'pkg_resources'):
        return 'setuptools'
    elif mod_startswith(item.module, 'google.protobuf'):
        return 'protobuf'
    elif mod_startswith(item.module, 'semantic_version'):
        return 'semantic-version'
    elif mod_startswith(item.module, 'pythonjsonlogger'):
        return 'python-json-logger'
    else:
        return item.module.split('.')[0]

def deps_for_pyfile(inputdir: str, filepath: str) -> Set[str]:
    localpaths = [inputdir]
    if not os.path.isfile(os.path.join(os.path.dirname(filepath), '__init__.py')):
        localpaths.append(os.path.dirname(filepath))

    imports = parse_imports(filepath)

    deps: Set[str] = set()
    for item in imports:
        if is_in_stdlib(item):
            continue
        elif is_local(localpaths, item):
            continue
        else:
            deps.add(import_to_distribname(item))

    return deps

def main(inputdirs: List[str], include_dev: bool = False) -> Set[str]:
    # setuptools 49.1.2 by opt-in and 60.0.0 by opt-out add a global
    # .pth file that depending on SETUPTOOLS_USE_DISTUTILS overrides
    # the stdlib distutils with a version of distutils built on top of
    # setuptools.  But if our is_in_stdlib() removes setuptools from
    # sys.path then importing that distutils will fail, which is wrong
    # for us because distutils is in fact in the stdlib.  So opt-out.
    #
    # ... disable it if it's already been enabled during Python start-up
    try:
        import _distutils_hack
        _distutils_hack.remove_shim()
    except ImportError:
        pass
    # ... prevent it from being enabled again
    os.environ['SETUPTOOLS_USE_DISTUTILS'] = 'stdlib'

    deps = set()
    for inputdir in inputdirs:
        for dirpath, dirnames, filenames in os.walk(inputdir, topdown=True):
            if 'build' in dirnames:
                dirnames.remove('build')
            if 'dist' in dirnames:
                dirnames.remove('dist')
            if '__pycache__' in dirnames:
                dirnames.remove('__pycache__')
            for filename in sorted(filenames):
                filepath = os.path.join(dirpath, filename)
                if filename.endswith('.py'):
                    if not include_dev:
                        if re.search(DEV_PY_FILES, filepath):
                            continue
                    deps = deps.union(deps_for_pyfile(inputdir, filepath))
    return deps

if __name__ == "__main__":
    argparser = argparse.ArgumentParser(description="Look at Python3 source files, and print out a list of Python distributions that those sources depend on")
    argparser.add_argument('--include-dev', action=argparse.BooleanOptionalAction)
    argparser.add_argument('input_dir', nargs='+')
    args = argparser.parse_args()
    deps = main(inputdirs=args.input_dir, include_dev=bool(args.include_dev))
    print("\n".join(sorted(deps)))
