# we're using ruamel.yaml instead of using yq+bash to parse the values file because we want to keep
# all the ordering, comments, formatting, etc.

# This is just a simple script to replace the image tag and repository values in our helm charts so
# that humanz don't have to do it.

import sys
import ruamel.yaml


def main(values_file, image_tag, repo=None):
    yaml = ruamel.yaml.YAML()
    yaml.indent(mapping=2)
    with open(values_file, 'r') as f:
        helm_values = yaml.load(f.read())

    if 'image' not in helm_values:
        helm_values['image'] = {}
    helm_values['image']['tag'] = image_tag
    if repo is not None:
        helm_values['image']['repository'] = repo

    with open(values_file, 'w') as f:
        yaml.dump(helm_values, f)


if __name__ == '__main__':
    if len(sys.argv) < 3:
        print('USAGE: create_yaml.py [VALUES_FILE] [IMAGE_TAG] ([REPO])')
        sys.exit(1)
    repo = None
    values_file = sys.argv[1]
    image_tag = sys.argv[2]
    if len(sys.argv) > 3:
        repo = sys.argv[3]

    main(values_file, image_tag, repo)
