package assert

import (
	"fmt"
	"runtime"
)

func Assert(something bool) {
	if !something {
		pc := make([]uintptr, 10)
		runtime.Callers(2, pc)
		f := runtime.FuncForPC(pc[0])
		file, line := f.FileLine(pc[0])
		panic(fmt.Sprintf("assertion failed at %s:%d %s\n", file, line, f.Name()))
	}
}
