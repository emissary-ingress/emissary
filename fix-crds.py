#!python

import sys

import copy
import json
import os
import yaml

from packaging import version

kubeversion = ''

def have_kubeversion(required_version):
    global kubeversion
    return version.parse(kubeversion) >= version.parse(required_version)

old_pro_crds = {
    'Filter',
    'FilterPolicy',
    'RateLimit'}

old_oss_crds = {
    'AuthService',
    'ConsulResolver',
    'KubernetesEndpointResolver',
    'KubernetesServiceResolver',
    'LogService',
    'Mapping',
    'Module',
    'RateLimitService',
    'TCPMapping',
    'TLSContext',
    'TracingService'}

def fix_crd(crd):
    # sanity check
    if crd['kind'] != 'CustomResourceDefinition' or not crd['apiVersion'].startswith('apiextensions.k8s.io/'):
        raise f"not a CRD: {crd}"

    # fix apiVersion
    if have_kubeversion('1.16'):
        crd['apiVersion'] = 'apiextensions.k8s.io/v1'
    else:
        crd['apiVersion'] = 'apiextensions.k8s.io/v1beta1'

    # fix CRD versions
    if have_kubeversion('1.11'):
        if 'version' in crd['spec']:
            del crd['spec']['version']
        if crd['spec']['names']['kind'] in old_pro_crds:
            crd['spec']['versions'] = [
                { 'name': 'v1beta1', 'served': True, 'storage': False },
                { 'name': 'v1beta2', 'served': True, 'storage': False },
                { 'name': 'v2', 'served': True, 'storage': True },
            ]
        elif crd['spec']['names']['kind'] in old_oss_crds:
            crd['spec']['versions'] = [
                { 'name': 'v1', 'served': True, 'storage': False },
                { 'name': 'v2', 'served': True, 'storage': True },
            ]
        else:
            crd['spec']['versions'] = [
                { 'name': 'v2', 'served': True, 'storage': True },
            ]
    else:
        if 'versions' in crd['spec']:
            del crd['spec']['versions']
        crd['spec']['version'] = 'v2'

    # fix labels
    labels = crd['metadata'].get('labels', {})
    labels['product'] = 'aes'
    crd['metadata']['labels'] = labels

    # fix categories
    categories = crd['spec']['names'].get('categories', [])
    if 'ambassador-crds' not in categories:
        # sys.stderr.write(f"CRD {crd['metadata']['name']} missing ambassador-crds category\n")
        categories.append('ambassador-crds')
    crd['spec']['names']['categories'] = categories

def main():
    global kubeversion

    kubeversion = sys.argv[1]
    crds = []
    for in_path in sys.argv[2:]:
        crds += list(yaml.safe_load_all(open(in_path, "r")))

    for crd in crds:
        fix_crd(crd)

    print("# GENERATED FILE: edits made by hand will not be preserved.")
    print()
    print(yaml.safe_dump_all(crds))

if __name__ == "__main__":
    main()
