package main

import "github.com/lyft/protoc-gen-star"

func main() {
	pgs.Init(pgs.IncludeGo(), pgs.DebugEnv("DEBUG"), pgs.MultiPackage()).
		RegisterPlugin(JSONify()).
		RegisterModule(ASTPrinter()).
		RegisterPostProcessor(pgs.GoFmt()).
		Render()
}
