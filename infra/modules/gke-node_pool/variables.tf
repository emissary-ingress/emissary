variable "cluster" {
  description = "name of the Kubernetes cluster to attach the node pool"
}

variable "location" {
	description = "Google Cloud region or zone where the node pool is created"
}

variable "name" {
	description = "name of the node pool"
}

variable "project" {
  description = "Google Cloud project that contains the cluster"
}

variable "auto_repair" {
  default = true
}

variable "auto_upgrade" {
  default = true
}

variable "node_count" {
	default = 1
}

variable "disk_size_gb" {
  default = 10
}

variable "disk_type" {
  default = "pd-standard"
}

variable "image_type" {
  default = "COS"
}

variable "local_ssd_count" {
  default = 0
}

variable "labels" {
  type        = "map"
  description = "Map of key/value pairs to label Nodes with. These will propogate into Kubernetes as Node Labels"
  default     = {}
}

variable "machine_type" {
  description = "Name of the Google Compute Engine machine type"
  default = "n1-standard-1"
}

variable "metadata" {
  // Unlikely that this ever needs to be set.
  // Reference: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/NodeConfig (see: metadata)

  type        = "map"
  description = "Metadata key/value pairs assigned to instances in the cluster"
  default     = {}
}

variable "oauth_scopes" {
  type = "list"
  description = "The OAuth scopes to apply to the compute instance..."
  default = ["https://www.googleapis.com/auth/cloud-platform"]
}

variable "preemptible" {
  description = "Run the underlying Compute Engine VM as preemptible instances (not recommended for default nodes)"
  default = false
}

variable "service_account" {
  description = "Name of the service account to use for Node VMs"
}

variable "taints" {
  type        = "list"
  description = "List of kubernetes taints to apply to each node"
  default     = []
}

variable "tags" {
  // Unlikely that this ever needs to be set.
  // Reference: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/NodeConfig (see: tags[])

  type        = "list"
  description = "List of tags applied to each node. Used for load balancer routing"
  default     = []
}

