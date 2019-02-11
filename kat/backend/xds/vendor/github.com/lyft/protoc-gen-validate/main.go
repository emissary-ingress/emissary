package main

import (
	"github.com/lyft/protoc-gen-star"
	"github.com/lyft/protoc-gen-validate/module"
)

func main() {
	pgs.
		Init(pgs.DebugEnv("DEBUG_PGV")).
		RegisterModule(module.Validator()).
		RegisterPostProcessor(pgs.GoFmt()).
		Render()
}
