#!/usr/bin/env python3

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

import ast
import importlib
import os
import sys
import sysconfig
from typing import List, NamedTuple, Optional, Set, AbstractSet


class ImportedItem(NamedTuple):
    """
    import mod             → ImportedItem(mod, None)
    from pkg import mod    → ImportedItem(pkg, mod)
    from mod import member → ImportedItem(mod, member)
    """
    module: str
    member: Optional[str]

def parse_imports(scriptbody: str) -> List[ImportedItem]:
    imports = []
    for node in ast.parse(scriptbody).body:
        if isinstance(node, ast.Import):
            #     import {node.names}
            imports += [ImportedItem(alias.name, None) for alias in node.names]
        elif isinstance(node, ast.ImportFrom):
            #     from {'.'*node.level}{node.module} import {node.names}
            modname = ('.'*(node.level or 0)) + (node.module or '')
            imports += [ImportedItem(modname, alias.name) for alias in node.names]
    return imports

def is_in_stdlib(item: ImportedItem) -> bool:
    if item.module.startswith('.'):
        return False

    original_sys_path = sys.path
    # Where isort does a similar thing, they have a comment saying the
    # call to 'os.path.normcase' is important on Windows.  Not that we
    # expect this to work on Windows without more modification.
    stdlib_prefix = os.path.normcase(sysconfig.get_paths()['stdlib'])
    sys.path = [d for d in sys.path if (
        os.path.normcase(d).startswith(stdlib_prefix) and
        ('site-packages' not in d) and
        ('dist-packages' not in d))]

    in_stdlib = False
    try:
        module = importlib.import_module(item.module)
        if item.member:
            getattr(module, item.member)
        in_stdlib = True
    except (ImportError, AttributeError):
        pass

    sys.path = original_sys_path
    return in_stdlib

def is_local(inputdir: str, item: ImportedItem) -> bool:
    if item.module.startswith('.'):
        return True

    if import_to_filepath(inputdir, item) is not None:
        return True

    return False

def import_to_filepath(inputdir: str, item: ImportedItem, fromfilepath: Optional[str]=None) -> Optional[str]:
    modname = item.module
    fromdir = inputdir
    if modname.startswith('.'):
        if not fromfilepath:
            raise Exception("this should not happen")
        fromdir = fromfilepath
        while modname.startswith('.'):
            fromdir = os.path.dirname(fromdir)
            modname = modname[1:]

    basefilepath = os.path.join(fromdir, *modname.split('.'))
    if os.path.isfile(basefilepath+".py"):
        return basefilepath+".py"
    if os.path.isfile(os.path.join(basefilepath, '__init__.py')):
        if item.member is None:
            return os.path.join(basefilepath, '__init__.py')
        if os.path.isfile(os.path.join(basefilepath, item.member+'.py')):
            return os.path.join(basefilepath, item.member+'.py')
        # FIXME(lukeshu): Should validate that .member exists, given
        # that this is used by is_local().
        return os.path.join(basefilepath, '__init__.py')
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
    else:
        return item.module.split('.')[0]

def rules_for_script(workspacedir: str, inputdir: str, filepath: str) -> List[str]:
    with open(filepath, 'r') as filehandle:
        filecontent = filehandle.read()
    imports = parse_imports(filecontent)

    deps: Set[str] = set()
    for item in imports:
        if is_in_stdlib(item):
            continue
        elif is_local(inputdir, item):
            depfilepath = import_to_filepath(inputdir, item, fromfilepath=filepath)
            if depfilepath is None:
                raise Exception("this should not happen")
            if os.path.dirname(depfilepath) == os.path.dirname(filepath):
                deps.add(quote(f":{os.path.basename(depfilepath)[:-len('.py')]}"))
            else:
                dirpart, filepart = os.path.split(os.path.relpath(depfilepath, start=workspacedir))
                deps.add(quote(f"//{dirpart}:{filepart[:-len('.py')]}"))
        else:
            deps.add(f"requirement({quote(import_to_distribname(item))})")

    if len(deps) == 0:
        depstr = '[]'
    else:
        depstr = "[\n        " + ",\n        ".join(sorted_deps(deps))+",\n    ]"
    return [f"""py_thing(
    name = {quote(os.path.basename(filepath)[:-len('.py')])},
    srcs = [{quote(os.path.basename(filepath))}],
    deps = {depstr},
)"""]

def sorted_deps(deps: AbstractSet[str]) -> List[str]:
    def keyfn(dep: str) -> str:
        if dep.startswith('":'):
            return '0-'+dep
        elif dep.startswith('"//'):
            return '1-'+dep
        elif dep.startswith('"@'):
            return '2-'+dep
        else:
            return '3-'+dep
    return sorted(deps, key=keyfn)

def quote(s: str) -> str:
    return '"' + s + '"'

def main(workspacedir: str, inputdirs: List[str]):
    for inputdir in inputdirs:
        for dirpath, _, filenames in os.walk(inputdir):
            build_rules: List[str] = []
            for filename in sorted(filenames):
                filepath = os.path.join(dirpath, filename)
                if filename == "BUILD" or filename == "BUILD.bazel":
                    os.remove(filepath)
                    continue
                elif filename.endswith('.py'):
                    build_rules += rules_for_script(workspacedir, inputdir, filepath)
            if build_rules:
                build_rules.insert(0, "# File generated by ./generate; DO NOT EDIT.")
                with open(os.path.join(dirpath, 'BUILD'), 'w') as buildfile:
                    buildfile.write("\n\n".join(build_rules)+"\n")

if __name__ == "__main__":
    main(".", ["."])
