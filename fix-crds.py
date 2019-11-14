#!python

import sys

import copy
import json
import os
import yaml

def fix_labels(crd):
    labels = crd['metadata'].get('labels', {})
    labels['product'] = 'aes'
    crd['metadata']['labels'] = labels

def fix_categories(crd):
    categories = crd['spec']['names'].get('categories', [])

    if 'ambassador-crds' not in categories:
        # sys.stderr.write(f"CRD {crd['metadata']['name']} missing ambassador-crds category\n")
        categories.append('ambassador-crds')
        crd['spec']['names']['categories'] = categories 

crds = []

for in_path in sys.argv[1:]:
    crds += list(yaml.safe_load_all(open(in_path, "r")))

crds_to_save = []

for crd in crds:
    # Fix labels and categories...
    fix_labels(crd)
    fix_categories(crd)

    # ...then save the resulting CRD into A/Pro and into Edge Stack.
    crds_to_save.append(crd)

print("# GENERATED FILE: edits made by hand will not be preserved.")
print()
print(yaml.safe_dump_all(crds_to_save))
