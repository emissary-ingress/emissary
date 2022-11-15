BEGIN {
	print("package " pkgname)
	print("")
	object=0
}

/\/\/ \+kubebuilder:object:root=true/ {
	object=1
}

/^type \S+ struct/ && object {
	if (!match($2, /List$/)) {
		print "func(*" $2 ") Hub() {}"
	}
	object=0
}
