#!python

import sys

import glob
import json
import os
import tarfile

def sanitize_snapshot(path: str):
	watt_dict = json.loads(open(path, "r"). read())

	sanitized = {}

	# Consul is pretty easy. Just sort, using service-dc as the sort key.
	consul_elements = watt_dict.get('Consul')

	if consul_elements:
		csorted = {}

		for key, value in consul_elements.items():
			csorted[key] = sorted(value, key=lambda x: f'{x["Service"]-x["Id"]}')

		sanitized['Consul'] = csorted

	# Kube is harder because we need to sanitize Kube secrets.
	kube_elements = watt_dict.get('Kubernetes')

	if kube_elements:
		ksorted = {}

		for key, value in kube_elements.items():
			if not value:
				continue

			if key == 'secret':
				for secret in value:
					if "data" in secret:
						data = secret["data"]

						for k in data.keys():
							data[k] = f'-sanitized-{k}-'

					metadata = secret.get('metadata', {})
					annotations = metadata.get('annotations', {})

					# Wipe the last-applied-configuration annotation, too, because it 
					# often contains the secret data.
					if 'kubectl.kubernetes.io/last-applied-configuration' in annotations:
						annotations['kubectl.kubernetes.io/last-applied-configuration'] = '--sanitized--'

			# All the sanitization above happened in-place in value, so we can just
			# sort it.
			ksorted[key] = sorted(value, key=lambda x: x.get('metadata',{}).get('name'))

		sanitized['Kubernetes'] = ksorted

	return sanitized

# Open a tarfile for output...
with tarfile.open('sanitized.tgz', 'w:gz') as archive:
	# ...then iterate any snapshots, sanitize, and stuff 'em in the tarfile.
	# Note that the '.yaml' on the snapshot file name is a misnomer: when
	# watt is involved, they're actually JSON. It's a long story.

	for path in glob.glob('snapshots/snap*.yaml'):
		# The tarfile can be flat, rather than embedding everything
		# in a directory with a fixed name.
		b = os.path.basename(path)

		sanitized = sanitize_snapshot(path)

		if sanitized:
			with open('sanitized.json', 'w') as tmp:
				tmp.write(json.dumps(sanitized))

			archive.add('sanitized.json', arcname=b)
			os.unlink('sanitized.json')
