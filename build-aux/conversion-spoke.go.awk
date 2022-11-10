BEGIN {
	print("package " pkgname)
	print("")
	print("import (")
	print("  k8sRuntime \"k8s.io/apimachinery/pkg/runtime\"")
	print("  \"sigs.k8s.io/controller-runtime/pkg/conversion\"")
	print(")")
	print("")

	# This chunk is all static; so why doesn't it go in a separate
	# non-generated file?  Well, it's just here to make the actual
	# .ConvertFrom() and .ConvertTo() methods below simpler.  They could be
	# inlined in the actual methods--but in my (LukeShu's) opinion, that
	# makes the AWK code too difficult to edit; so they're split out to
	# here.  But splitting them out further and moving them out of the AWK
	# would be too much--it'd create more things to keep in-sync and
	# smearing the implementation across multiple files would make things
	# harder, not easier.
	print("func convertFrom(src conversion.Hub, dst conversion.Convertible) error {")
	print("  scheme := conversionScheme()")
	print("  var cur k8sRuntime.Object = src")
	print("  for i := len(conversionIntermediates) - 1; i >= 0; i-- {")
	print("    gv := conversionIntermediates[i]")
	print("    var err error")
	print("    cur, err = scheme.ConvertToVersion(cur, gv)")
	print("    if err != nil {")
	print("      return err")
	print("    }")
	print("  }")
	print("  return scheme.Convert(cur, dst, nil)")
	print("}")
	print("")
	print("func convertTo(src conversion.Convertible, dst conversion.Hub) error {")
	print("  scheme := conversionScheme()")
	print("  var cur k8sRuntime.Object = src")
	print("  for _, gv := range conversionIntermediates {")
	print("    var err error")
	print("    cur, err = scheme.ConvertToVersion(cur, gv)")
	print("    if err != nil {")
	print("      return err")
	print("    }")
	print("  }")
	print("  return scheme.Convert(cur, dst, nil)")
	print("}")
	print("")

	object=0
}

/\/\/ \+kubebuilder:object:root=true/ {
	object=1
}

/^type \S+ struct/ && object {
	if (!match($2, /List$/)) {
		print "func(dst *" $2 ") ConvertFrom(src conversion.Hub) error { return convertFrom(src, dst) }"
		print "func(src *" $2 ") ConvertTo(dst conversion.Hub) error { return convertTo(src, dst) }"
	}
	object=0
}
