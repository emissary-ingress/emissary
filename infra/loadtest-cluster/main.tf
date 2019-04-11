data "google_container_engine_versions" "kubernetes_version" {
	project        = "${var.project}"
	location       = "${var.cluster_location}"

	// the trailing "dot" below is signfigant
	// see: https://www.terraform.io/docs/providers/google/d/google_container_engine_versions.html#version_prefix
	version_prefix = "${var.cluster_version}."
}

module "cluster" {
	source = "../modules/gke-cluster"

	environment        = "sbx"
	name               = "loadtest"
	project            = "${var.project}"
	cluster_location   = "${var.cluster_location}"
	cluster_version    = "${data.google_container_engine_versions.kubernetes_version.latest_master_version}"

	resource_labels = {
		"datawire-team"    = "development"
		"datawire-env"     = "sbx"
		"datawire-project" = "ambassador-pro"
	}
}

module "generator_node_pool" {
	source = "../modules/gke-node_pool"

	project         = "${var.project}"
	preemptible     = true
	name            = "generators"
	cluster         = "${module.cluster.cluster_name}"
	location        = "${var.cluster_location}"
	service_account = "${module.cluster.cluster_service_account_email}"
	node_count      = 1 // "node_count" is a bit of misnomer... it is nodes per region and there are <N> regions.
	machine_type    = "n1-standard-4"
}

module "backends_node_pool" {
	source = "../modules/gke-node_pool"

	project         = "${var.project}"
	preemptible     = true
	name            = "backends"
	cluster         = "${module.cluster.cluster_name}"
	location        = "${var.cluster_location}"
	service_account = "${module.cluster.cluster_service_account_email}"
	node_count      = 1 // "node_count" is a bit of misnomer... it is nodes per region and there are <N> regions.
	machine_type    = "n1-standard-4"
}
