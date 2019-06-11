locals {
	name     = "${lower(replace(var.name, "/[^a-zA-Z0-9-]*/", ""))}"
	metadata = "${merge(var.metadata, map("disable-legacy-endpoints", "true"))}"
}
