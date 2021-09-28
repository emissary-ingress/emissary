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


# ensure that the yaml docs are sorted in the same way as in the requirements.
# order actually matters here. for example, we need the namespace show up before any
# namespaced resources.
# Also this ensures that all the "required" resources make it into the final yaml
def same_sort(requirements, yaml_docs):
    sorted_resources = []
    for req in requirements.get('resources'):
        req_key = get_requirement_key(req)
        if req_key not in yaml_docs:
            raise Exception('Resource %s not found in generated yaml' % req_key)
        sorted_resources.append(yaml_docs[req_key])
    return sorted_resources


class RequirementChecker():

    def __init__(self, requirements):
        self.requirements = {}
        for req in requirements:
            key = get_requirement_key(req)
            self.requirements[key] = True


    def is_required(self, resource):
        key = get_resource_key(resource)
        return key in self.requirements


def main(templated_helm_file, require_file):
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=2)
    with open(templated_helm_file, 'r') as f:
        templated_helm = yaml.load_all(f.read())
    with open(require_file, 'r') as f:
        requirements = yaml.load(f.read())
    checker = RequirementChecker(requirements.get('resources'))

    new_doc = {}
    for yaml_doc in templated_helm:
        if yaml_doc is None:
            continue
        if checker.is_required(yaml_doc):
            new_doc[get_resource_key(yaml_doc)] = yaml_doc
    print('# GENERATED FILE: edits made by hand will not be preserved.')
    print('---')
    yaml.dump_all(same_sort(requirements, new_doc), sys.stdout)


if __name__ == '__main__':
    if len(sys.argv) != 3:
        print('USAGE: create_yaml.py [HELM_GENERATED_FILE] [REQUIREMENTS_FILE]')
        sys.exit(1)
    templated_helm = sys.argv[1]
    require_file = sys.argv[2]

    main(templated_helm, require_file)
