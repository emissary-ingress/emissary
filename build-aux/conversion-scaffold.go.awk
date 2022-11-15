BEGIN {
	print("//+build scaffold")
	print("")
	print("package " pkgname)
	inFunc=0
	curFunc=""
}

match($0, /^func auto(Convert_[^(]+)(\(.*)/, m) {
	if (inFunc) {
		print("  return nil")
		print("}")
		print("")
		inFunc=0
	}
	curFunc=\
		"func " m[1] m[2] \
		"  if err := auto" m[1] "(in, out, s); err != nil {" \
		"    return err" \
		"  }"
}

/INFO|WARN/ {
	if (!inFunc) {
		print(curFunc)
		inFunc=1
	}
	print
}

END {
	if (inFunc) {
		print("  return nil")
		print("}")
	}
}
