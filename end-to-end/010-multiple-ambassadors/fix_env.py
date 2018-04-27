#!python

import sys

import yaml

import dpath

id_num = sys.argv[1]
namespace = 'test-010-%s' % id_num

manifest = list(yaml.safe_load_all(open(sys.argv[2], 'r')))

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

    # print("Adding namespace %s to %s %s" % (namespace, kind, name))
    dpath.util.new(manifest, "/%d/metadata/namespace" % x, namespace)

    if kind == 'Deployment':
        # print("Setting environment for %s %s" % (kind, name))
        path = "/%d/spec/template/spec/containers/0/env" % x
        dpath.util.new(manifest, path, [
            {
                'name': 'AMBASSADOR_NAMESPACE',
                'value': namespace
            },
            {
                'name': 'AMBASSADOR_ID',
                'value': 'ambassador-%s' % id_num
            }            
        ])
    
    keep.append(manifest[x])

yaml.safe_dump_all(keep, open(sys.argv[3], "w"), default_flow_style=False)
