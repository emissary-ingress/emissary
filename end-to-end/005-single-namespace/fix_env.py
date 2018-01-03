#!python

import sys

import yaml

import dpath

manifest = list(yaml.safe_load_all(open(sys.argv[1], 'r')))

for x in range(len(manifest)):
    if dpath.util.get(manifest, "/%d/kind" % x) == 'Deployment':
        path = "/%d/spec/template/spec/containers/0/env" % x
        dpath.util.new(manifest, path, [
            { 
                'name': 'AMBASSADOR_SINGLE_NAMESPACE',
                'value': 'YES'
            },
            {
                'name': 'AMBASSADOR_NAMESPACE',
                'value': 'default'
            }
        ])

yaml.safe_dump_all(manifest, open(sys.argv[2], "w"), default_flow_style=False)
