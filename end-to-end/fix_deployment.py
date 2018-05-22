#!python

import sys

import yaml

import dpath

namespace = sys.argv[1]
ambassador_id = sys.argv[2]
input_yaml_path = sys.argv[3]
output_yaml_path = sys.argv[4]

manifest = list(yaml.safe_load_all(open(input_yaml_path, 'r')))

kinds_to_delete = {
    "ClusterRole": True,
    "ServiceAccount": True,
    "ServiceAccount": True,
    "ClusterRoleBinding": True,
}

keep = []

for x in range(len(manifest)):
    kind = dpath.util.get(manifest, "/%d/kind" % x)
    name = dpath.util.get(manifest, "/%d/metadata/name" % x)

    if kind in kinds_to_delete:
        # print("Skipping %s %s" % (kind, name))
        continue

    print("Adding namespace %s to %s %s" % (namespace, kind, name))
    dpath.util.new(manifest, "/%d/metadata/namespace" % x, namespace)

    if kind == 'Deployment':
        print("Setting environment for %s %s" % (kind, name))
        path = "/%d/spec/template/spec/containers/0/env" % x
        dpath.util.new(manifest, path, [
            {
                'name': 'AMBASSADOR_NAMESPACE',
                'value': namespace
            },
            {
                'name': 'AMBASSADOR_ID',
                'value': ambassador_id
            }            
        ])
    
    keep.append(manifest[x])

yaml.safe_dump_all(keep, open(output_yaml_path, "w"), default_flow_style=False)
