package server

import ()

func Main(version string) {
	s := NewServer()
	s.ServeHTTP()
}
