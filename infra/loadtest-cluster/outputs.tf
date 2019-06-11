output "cluster_name" {
	value = "${module.cluster.cluster_name}"
}

output "generators_node_pool_name" {
	value = "${module.generator_node_pool.name}"
}

output "backends_node_pool_name" {
	value = "${module.backends_node_pool.name}"
}

output "kubeconfig_path" {
	value = "${module.cluster.kubeconfig_output_path}"
}
