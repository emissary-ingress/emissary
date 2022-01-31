#!/usr/bin/env python

# This script is to help generate any flat yaml files from the ambassador helm chart.
#
# This script takes two arguments:
#   1. A multi-doc yaml file generated from running:
#       `helm template ambassador -f [VALUES_FILE.yaml] -n [NAMESPACE] ./charts/emissary-ingress`
#   2. A yaml file listing the required kubernetes resources from the generated helm template to
#   output to stdout. See ../aes/require.yaml for an example
#
# This script will output to stdout the resources from 1) iff they are referenced in 2). It will
# preserve the ordering from 2), and will error if any resources named in 2) are missing in 1)
import sys
import ruamel.yaml


NO_NAMESPACE = '__no_namespace'


def get_resource_key(resource):
    metadata = resource.get('metadata', {})
    namespace = metadata['namespace'] if 'namespace' in metadata else NO_NAMESPACE

    return '{}.{}.{}'.format(resource['kind'], metadata['name'], namespace)


def get_requirement_key(req):
    if 'kind' not in req or 'name' not in req:
        raise Exception('Malformed requirement %s' % req)
    ns = req['namespace'] if 'namespace' in req else NO_NAMESPACE
    return '{}.{}.{}'.format(req['kind'], req['name'], ns)


def main(templated_helm_file, require_file):
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=2)
    with open(templated_helm_file, 'r') as f:
        templated_helm = {}
        for yaml_doc in yaml.load_all(f.read()):
            if yaml_doc is None:
                continue
            templated_helm[get_resource_key(yaml_doc)] = yaml_doc
    with open(require_file, 'r') as f:
        requirements = yaml.load(f.read())

    print('# GENERATED FILE: edits made by hand will not be preserved.')
    # Print out required resources in the order they appear in require_file.  Order actually matters
    # here, for example, we need the namespace show up before any namespaced resources.
    for requirement in requirements.get('resources'):
        print('---')
        key = get_requirement_key(requirement)
        if key not in templated_helm:
            raise Exception(f'Resource {key} not found in generated yaml (known resources are: {templated_helm.keys()})')
        yaml.dump(templated_helm[key], sys.stdout)


if __name__ == '__main__':
    if len(sys.argv) != 3:
        print('USAGE: create_yaml.py [HELM_GENERATED_FILE] [REQUIREMENTS_FILE]')
        sys.exit(1)
    templated_helm = sys.argv[1]
    require_file = sys.argv[2]

    main(templated_helm, require_file)
