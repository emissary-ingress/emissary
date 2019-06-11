output "region" {
	value = "${google_container_cluster.cluster.region}"
}

output "cluster_name" {
  	value = "${google_container_cluster.cluster.name}"
}

output "cluster_base_name" {
  	value = "${local.name}"
}

output "master_version" {
  	value       = "${google_container_cluster.cluster.master_version}"
  	description = "Current version of the master in the cluster."
}

output "endpoint" {
  	value       = "${google_container_cluster.cluster.endpoint}"
  	description = "IP address of this cluster's Kubernetes master"
}

output "instance_group_urls" {
  	value       = "${google_container_cluster.cluster.instance_group_urls}"
  	description = "List of instance group URLs which have been assigned to the cluster"
}

output "maintenance_window" {
  	value       = "${google_container_cluster.cluster.maintenance_policy.0.daily_maintenance_window.0.duration}"
	description = "Duration of the time window, automatically chosen to be smallest possible in the given scenario. Duration will be in RFC3339 format PTnHnMnS"
}

output "username" {
  	value       = "${google_container_cluster.cluster.master_auth.0.username}"
  	description = "The username to login on the master Kubernetes"
}

output "password" {
  	value       = "${google_container_cluster.cluster.master_auth.0.password}"
  	description = "The password to login on the master Kubernetes"
}

output "client_certificate" {
  	value       = "${google_container_cluster.cluster.master_auth.0.client_certificate}"
  	description = "Base64 encoded public certificate used by clients to authenticate to the cluster endpoint"
}

output "client_key" {
  	value       = "${google_container_cluster.cluster.master_auth.0.client_key}"
  	description = "Base64 encoded private key used by clients to authenticate to the cluster endpoint"
}

output "cluster_ca_certificate" {
  	value       = "${google_container_cluster.cluster.master_auth.0.cluster_ca_certificate}"
  	description = "Base64 encoded public certificate that is the root of trust for the cluster"
}

output "cluster_network_name" {
  	value = "${google_compute_network.cluster_network.name}"
  	description = "Name of the Google Cloud network that contains the cluster"
}

output "cluster_network_self_link" {
  	value = "${google_compute_network.cluster_network.self_link}"
  	description = "URL of the Google Cloud network that contains the cluster"
}

output "cluster_service_account_email" {
  	value = "${google_service_account.cluster_service_account.email}"
}

output "kubeconfig_output_path" {
	value = "${local.kubeconfig_output_path}"
}
