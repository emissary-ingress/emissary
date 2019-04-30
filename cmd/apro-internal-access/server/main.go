package server

func Main(version string, sharedSecretPath string) {
	s := NewServer(sharedSecretPath)
	s.ServeHTTP()
}
