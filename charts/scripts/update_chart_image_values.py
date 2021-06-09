# we're using ruamel.yaml instead of using yq+bash to parse the values file because we want to keep
# all the ordering, comments, formatting, etc.

# This is just a simple script to replace the image tag and repository values in our helm charts so
# that humanz don't have to do it.

import os.path
import sys
import argparse
import ruamel.yaml


def main(values_file, image_tag, repo=None):
    image_key = 'tag'
    repo_key = 'repository'
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=2)
    with open(values_file, 'r') as f:
        helm_values = yaml.load(f.read())

    if 'image' not in helm_values:
        helm_values['image'] = {}
    helm_values['image'][image_key] = image_tag
    if repo is not None:
        helm_values['image'][repo_key] = repo

    with open(values_file, 'w') as f:
        yaml.dump(helm_values, f)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Edit image values for ambassador helm charts.')

    parser.add_argument('--values-file', help='values file to edit', required=True)
    parser.add_argument('--tag', help='value for image tag', required=True)
    parser.add_argument('--repo', help='value for image repo')

    args = parser.parse_args()

    if not os.path.isfile(args.values_file):
        print(f'--values-file {args.values_file} is not a valid file path')
        sys.exit(1)

    main(args.values_file, args.tag, args.repo)
