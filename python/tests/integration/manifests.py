import os
import subprocess
from typing import Dict, Final

def _get_images() -> Dict[str, str]:
    ret: Dict[str, str] = {}

    image_names = [
        'test-auth',
        'test-ratelimit',
        'test-shadow',
        'test-stats',
        'kat-client',
        'kat-server',
    ]

    if image := os.environ.get('AMBASSADOR_DOCKER_IMAGE'):
        ret['emissary'] = image
    else:
        image_names.append('emissary')

    try:
        subprocess.run(['make']+[f'docker/{name}.docker.push.remote' for name in image_names],
                       check=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    except subprocess.CalledProcessError as err:
        raise Exception(f"{err.stdout}{err}") from err

    for name in image_names:
        with open(f'docker/{name}.docker.push.remote', 'r') as fh:
            # file contents:
            #   line 1: image ID
            #   line 2: tag 1
            #   line 3: tag 2
            #   ...
            tag = fh.readlines()[1].strip()
            ret[name] = tag

    return ret

images: Final = _get_images()

_file_cache: Dict[str, str] = {}

def load(manifest_name: str) -> str:
    if manifest_name in _file_cache:
        return _file_cache[manifest_name]
    manifest_dir = __file__[:-len('.py')]
    manifest_file = os.path.join(manifest_dir, manifest_name+'.yaml')
    manifest_content = open(manifest_file, 'r').read()
    _file_cache[manifest_name] = manifest_content
    return manifest_content

def format(st: str, /, **kwargs):
        serviceAccountExtra = ''
        if os.environ.get("DEV_USE_IMAGEPULLSECRET", False):
            serviceAccountExtra = """
imagePullSecrets:
- name: dev-image-pull-secret
"""
        return st.format(serviceAccountExtra=serviceAccountExtra,
                         images=images,
                         **kwargs)

# Use .replace instead of .format because there are other '{word}' things in 'description' fields
# that would cause KeyErrors when .format erroneously tries to evaluate them.
CRDmanifests: Final[str] = (
    load('crds')
    .replace('{images[emissary]}', images['emissary'])
    .replace('{serviceAccountExtra}', format('{serviceAccountExtra}'))
)
