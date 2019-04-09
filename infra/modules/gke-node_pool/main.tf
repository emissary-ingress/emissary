resource "google_container_node_pool" "pool" {
	cluster     = "${var.cluster}"
	name        = "${var.name}"
	project     = "${var.project}"
	location    = "${var.location}"
	node_count  = "${var.node_count}"

	management {
		auto_repair  = "${var.auto_repair}"
		auto_upgrade = "${var.auto_upgrade}"
	}

	node_config {
		disk_size_gb    = "${var.disk_size_gb}"
		disk_type       = "${var.disk_type}"
		local_ssd_count = "${var.local_ssd_count}"
		labels          = "${var.labels}"
		image_type      = "${var.image_type}"
		machine_type    = "${var.machine_type}"
		metadata        = "${local.metadata}"
		oauth_scopes    = ["${var.oauth_scopes}"]
		preemptible     = "${var.preemptible}"
		service_account = "${var.service_account}"
		tags            = ["${var.tags}"]
	}
}
