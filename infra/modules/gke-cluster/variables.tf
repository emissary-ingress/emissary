variable "auto_repair" {
  default = true
}

variable "auto_upgrade" {
  default = true
}

variable "cluster_location" {
	description = "Google Cloud region or zone where the cluster masters reside"
}

variable "daily_maintenance_window" {
  // Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/maintenance-window

  description = "Start of the four hour period when cluster maintenance occurs (fmt: HH:MM) (tz: UTC)."
  default     = "04:00"
}

variable "disable_horizontal_pod_autoscaling" {
  default = false
}

variable "disable_http_load_balancing" {
  default = false
}

variable "disable_kubernetes_dashboard" {
  default = true
}

variable "disable_network_policy_config" {
  default = true
}

variable "enable_abac" {
  description = "Enable legacy ABAC authorization for the Kubernetes cluster"
  default = false
}

variable "enable_kubernetes_alpha" {
  description = "Enable access to Kubernetes alpha features"
  default = false
}

variable "enable_network_policy" {
  // Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/network-policy

  description = "Enable Network Policy configuration"
  default = false
}

variable "enable_pod_security_policy" {
  default = false
}

variable "enable_client_certificate" {
  default = false
}

variable "environment" {
  description = "Name of the operational environment"
}

variable "logging_service" {
  default = "none"
}

variable "kubeconfig_output_path" {
	default = ""
}

variable "master_username" {
  description = "Username for HTTP basic authentication to the cluster master"
  default = "" // when set empty string ("") along with "master_password" then HTTP basic authentication is disabled
}

variable "master_password" {
  description = "Password for HTTP basic authentication to the cluster master"
  default = "" // when set empty string ("") along with "master_password" then HTTP basic authentication is disabled
}

variable "cluster_version" {
	description = "Version of kubernetes to run"
}

variable "monitoring_service" {
  default = "none"
}

variable "name" {
  description = "Name of the Kubernetes cluster"
}

variable "network_policy_provider" {
  // Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/network-policy

  description = "Network Policy provider name. Needs to be set if 'enable_network_policy' is true"
  default     = "PROVIDER_UNSPECIFIED"
}

variable "oauth_scopes" {
  type = "list"
  description = "The OAuth scopes to apply to the compute instance..."
  default = ["https://www.googleapis.com/auth/cloud-platform"]
}

variable "resource_labels" {
  type = "map"
  description = "Key/value resource labels attached to the compute infrastructure owned by the cluster"
  default = {}
}

variable "project" {
  description = "Google Cloud project that contains the cluster"
}

variable "service_account_iam_roles" {
  type = "list"
  description = "Specific IAM roles to grant the Service Account"
  default = []
}

// -----------------------------------------------------------------------------
// Default Node Pool Configuration
// -----------------------------------------------------------------------------

//variable "node_disk_size_gb" {
//  default = 10
//}
//
//variable "node_disk_type" {
//  default = "pd-standard"
//}
//
//variable "node_image_type" {
//  default = "COS"
//}
//
//variable "node_local_ssd_count" {
//  default = 0
//}
//
//variable "node_labels" {
//  type        = "map"
//  description = "Map of key/value pairs to label Nodes with. These will propogate into Kubernetes as Node Labels"
//  default     = {}
//}
//
//variable "node_machine_type" {
//  description = "Name of the Google Compute Engine machine type"
//  default = "n1-standard-2"
//}
//
//variable "node_metadata" {
//  // Unlikely that this ever needs to be set.
//  // Reference: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/NodeConfig (see: metadata)
//
//  type        = "map"
//  description = "Metadata key/value pairs assigned to instances in the cluster"
//  default     = {}
//}
//
//variable "node_preemptible" {
//  description = "Run the underlying Compute Engine VM as preemptible instances (not recommended for default nodes)"
//  default = false
//}
//
//variable "node_taints" {
//  type        = "list"
//  description = "List of kubernetes taints to apply to each node"
//  default     = []
//}
//
//variable "node_tags" {
//  // Unlikely that this ever needs to be set.
//  // Reference: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/NodeConfig (see: tags[])
//
//  type        = "list"
//  description = "List of tags applied to each node. Used for load balancer routing"
//  default     = []
//}
//
//variable "node_workload_metadata_config" {
//  // Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/metadata-concealment
//  description = "How much Node VM metadata to expose to Kubernetes Pods"
//  type = "map"
//  default = {
//    node_metadata = "SECURE"
//  }
//}
