package pf

import (
	"fmt"
	"unsafe"
)

// #include "string.h"
// #include "stdlib.h"
import "C"

// cStringCopy eraps the strlcpy c function. It copies the passed
// source string into the target dst pointer (char buffer) of a fixed
// size. If the size of the string to copy is to big, to copy the string#
// the operation will return ann error.
func cStringCopy(dst *C.char, src string, size int) error {
	srcStr := unsafe.Pointer(C.CString(src))
	defer C.free(srcStr)
	if C.strlcpy((*C.char)(dst), (*C.char)(srcStr), C.size_t(size)) >= C.size_t(size) {
		return fmt.Errorf("strlcpy: string '%s' to long (max: %d was: %d)",
			src, size, len(src))
	}
	return nil
}

func bytesCopy(dst []byte, src string, size int) error {
	if len(src) > size {
		return fmt.Errorf("source string exceeds max length(%d): %s", size, src)
	}
	copy(dst, src)
	return nil
}
