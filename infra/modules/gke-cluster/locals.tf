locals {
	name = "kube-${lower(var.environment)}-${lower(replace(var.name, "/[^a-zA-Z0-9-]*/", ""))}"

	// These OAuth scopes MUST ALWAYS be present in order for the cluster to function properly. They are merged with the
	// variable "additional_oauth_scopes".
	iam_roles = [
		"roles/logging.logWriter",
		"roles/storage.objectViewer",
		"roles/monitoring.admin"
	]

	resource_labels = {
		"datawire-terraformed" = true
	}

	kubeconfig_output_path = "${var.kubeconfig_output_path == "" ? pathexpand(format("%s/%s.kubeconfig", path.root, var.name)) : var.kubeconfig_output_path}"
}
