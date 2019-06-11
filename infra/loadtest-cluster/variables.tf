variable "project" {
	description = "Name of the Google Cloud project"
}

variable "cluster_location" {
	description = "Name of the Google Cloud region or zone where the cluster is created"
	default = "us-central1"
}

variable "cluster_version" {
	description = "Version of Kubernetes to provision"
}
