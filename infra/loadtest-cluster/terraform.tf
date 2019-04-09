// configures the terraform state storage backend... best to not muck with this if you do not understand it.
//
terraform {
	backend "gcs" {
		bucket  = "datawireio-terraform"
		prefix  = "ambassador-pro-loadtest-cluster-infra"
	}
}
