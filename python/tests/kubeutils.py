import tempfile

from tests.runutils import run_with_retry


def meta_action_kube_artifacts(namespace, artifacts, action, retries=0):
    temp_file = tempfile.NamedTemporaryFile()
    temp_file.write(artifacts.encode())
    temp_file.flush()

    command = ["tools/bin/kubectl", action, "-f", temp_file.name]
    if namespace is None:
        namespace = "default"

    if namespace is not None:
        command.extend(["-n", namespace])

    run_with_retry(command, retries=retries)
    temp_file.close()


def apply_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action="apply", retries=1)


def delete_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action="delete")
