resource "google_compute_network" "cluster_network" {
	auto_create_subnetworks = true
	name                    = "${local.name}"
	description             = "${local.name} network"
	project                 = "${var.project}"
	routing_mode            = "REGIONAL"
}

resource "google_compute_firewall" "default" {
	name    = "nodeports"
	project = "${var.project}"
	network = "${google_compute_network.cluster_network.self_link}"

	allow {
		protocol = "icmp"
	}

	allow {
		protocol = "tcp"
		ports    = ["30000-32767"]
	}
}

resource "google_service_account" "cluster_service_account" {
	account_id   = "${local.name}"
	display_name = "${local.name} svc account in ${var.project}"
	project      = "${var.project}"
}

resource "google_project_iam_member" "service_account" {
	count   = "${length(concat(local.iam_roles, var.service_account_iam_roles))}"
	project = "${var.project}"
	role    = "${element(concat(local.iam_roles, var.service_account_iam_roles), count.index)}"
	member  = "serviceAccount:${google_service_account.cluster_service_account.email}"
}

resource "null_resource" "kubeconfig" {
	triggers {
		cluster_id = "${google_container_cluster.cluster.id}"
	}

	provisioner "local-exec" {
		environment {
			KUBECONFIG = "${local.kubeconfig_output_path}"
		}

		command = "gcloud container clusters get-credentials ${local.name} --zone ${var.cluster_location} --project=${var.project}"
	}
}

resource "google_container_cluster" "cluster" {
	description = "${local.name} in ${var.cluster_location}"

	enable_kubernetes_alpha  = "${var.enable_kubernetes_alpha}"
	enable_legacy_abac       = "${var.enable_abac}"
	logging_service          = "${var.logging_service}"
	min_master_version       = "${var.cluster_version}"
	monitoring_service       = "${var.monitoring_service}"
	name                     = "${local.name}"
	project                  = "${var.project}"
	network                  = "${google_compute_network.cluster_network.self_link}"
	location                 = "${var.cluster_location}"
	resource_labels          = "${merge(local.resource_labels, var.resource_labels)}"
	initial_node_count       = 1
	remove_default_node_pool = true

	addons_config {
		horizontal_pod_autoscaling {
	  		disabled = "${var.disable_horizontal_pod_autoscaling}"
		}

		http_load_balancing {
	  		disabled = "${var.disable_http_load_balancing}"
		}

		kubernetes_dashboard {
	  		disabled = "${var.disable_kubernetes_dashboard}"
		}

		network_policy_config {
	  		disabled = "${var.disable_network_policy_config}"
		}
	}

	maintenance_policy {
		daily_maintenance_window {
	  		start_time = "${var.daily_maintenance_window}"
		}
	}

	master_auth {
		// This is considered a legacy authentication mechanism by Google Cloud and is why the module defaults these values
		// to empty string ("") in variables.tf
		password = "${var.master_username}"
		username = "${var.master_password}"

		// This is considered a legacy authentication mechanism by Google Cloud and is why the module defaults this value
		// to false in variables.tf
		client_certificate_config {
		  issue_client_certificate = "${var.enable_client_certificate}"
		}
	}

	network_policy {
		enabled  = "${var.enable_network_policy}"
		provider = "${var.network_policy_provider}"
	}
}
