#!python

import sys

import copy
import json
import os
import yaml

import dpath.util

def fix_labels(crd):
    labels = crd['metadata'].get('labels', {})
    labels['product'] = 'aes'
    crd['metadata']['labels'] = labels

def transform_oss_file(which, oss_file):
    # Load up the OSS Ambassador Deployment.

    oss_deployment = None

    for obj in yaml.safe_load_all(oss_file):
        kind = obj['kind']
        name = obj['metadata']['name']

        if kind == 'Deployment' and name == 'ambassador':
            oss_deployment = obj

    # Next, fix up the Ambassador deployment for A/Pro.
    fix_labels(oss_deployment)
    deployment = copy.deepcopy(oss_deployment)

    # This is common between A/Pro and Edge Stack.
    dpath.util.merge(deployment, {
        'metadata': {
            'namespace': 'ambassador'
        },
        'spec': {
            'replicas': 1,
            'template': {
                'spec': {
                    'terminationGracePeriodSeconds': 0,
                    'volumes': [
                        { 'name': 'ambassador-edge-stack-secrets',
                          'secret': { 'secretName': 'ambassador-edge-stack' }
                        }
                    ]
                }
            }
        }
       })

    dpath.util.merge(dpath.util.get(deployment, 'spec/template/spec/containers/0'), {
        'name': 'aes',
        'imagePullPolicy': 'Always',
        'env': [
            { 'name':  'REDIS_URL',
              'value': 'ambassador-redis:6379' },
            { 'name':  'AMBASSADOR_URL',
              'value': 'https://ambassador.default.svc.cluster.local' },
            { 'name':  'POLL_EVERY_SECS',
              'value': '60' },
            { 'name':  'AMBASSADOR_INTERNAL_URL',
              'value': 'https://127.0.0.1:8443' },
            { 'name':  'AMBASSADOR_ADMIN_URL',
              'value': 'http://127.0.0.1:8877' }
        ],
        'volumeMounts': [
            { 'name':      'ambassador-edge-stack-secrets',
              'mountPath': '/.config/ambassador',
              'readOnly':  True
            }
        ],
        'resources': {
            'limits': {
                'cpu': '1000m',
                'memory': '600Mi'
            },
            'requests': {
                'cpu': '200m',
                'memory': '300Mi'
            }
        }
       })

    # I don't think we really want to do this.
    # dpath.util.delete(deployment, 'spec/template/spec/containers/0/resources')
    dpath.util.delete(deployment, 'spec/template/spec/containers/0/livenessProbe/initialDelaySeconds')
    dpath.util.delete(deployment, 'spec/template/spec/containers/0/readinessProbe/initialDelaySeconds')

    # A few things change, though, between the two.
    if which == 'apro':
        dpath.util.merge(dpath.util.get(deployment, 'spec/template/spec/containers/0'), {
            'image': '-XXX-MARKER-XXX-',
            'env': [
                { 'name':  'SCOUT_DISABLE',
                  'value': '1' },
                # { 'name':  'AES_LOG_LEVEL',
                #   'value': 'DEBUG' }
            ]
        })
        dpath.util.delete(deployment, 'spec|template|metadata|labels|app.kubernetes.io/managed-by', separator='|')
    elif which == 'edge_stack':
        dpath.util.set(deployment, 'spec/template/spec/containers/0/image', 'quay.io/datawire/aes:$version$')

    # OK. Render the deployment as text...
    text = yaml.safe_dump(deployment)

    # ...and, for non-Edge-Stack, swap in the funky image template variable.
    if which != 'edge_stack':
        text = text.replace('-XXX-MARKER-XXX-', '{{env "AES_IMAGE"}}')

    return text

def transform_apro_file(which, apro_file):
    # Next, grab the template leader...
    ret = ""
    for line in apro_file.readlines():
        if line.startswith('#===='):
            continue

        if line.startswith('#devmagic:'):
            if which == 'edge_stack':
                continue
            else:
                line = line[len('#devmagic:'):]

        ret += line
    return ret

def main(which, oss_path, apro_path):
    if which not in ['apro', 'edge_stack']:
        sys.stderr.write('only apro and edge_stack are valid\n')
        sys.exit(1)

    sys.stdout.write('# GENERATED FILE: edits made by hand will not be preserved.\n')

    sys.stdout.write(transform_apro_file(which, open(apro_path, "r")))

    sys.stdout.write(transform_oss_file(which, open(oss_path, "r")))

if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2], sys.argv[3])

