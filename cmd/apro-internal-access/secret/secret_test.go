package secret

import (
	"io/ioutil"
	"os"
	"testing"
)

// Different file contents get different secrets, the same file contents gets
// the same secret.
func TestLoadSecret(t *testing.T) {
	file1, _ := ioutil.TempFile("", "prefix")
	file2, _ := ioutil.TempFile("", "prefix")
	file3, _ := ioutil.TempFile("", "prefix")
	defer os.Remove(file1.Name())
	defer os.Remove(file2.Name())
	defer os.Remove(file3.Name())

	file1.WriteString("abc")
	file2.WriteString("abc")
	file3.WriteString("def")

	secret1 := LoadSecret(file1.Name())
	secret2 := LoadSecret(file2.Name())
	secret3 := LoadSecret(file3.Name())
	if secret1 != secret2 {
		t.Errorf("Same contents resulted in different secret")
	}
	if secret1 == secret3 {
		t.Errorf("Different contents resulted in same secret")
	}
	if len(secret1) != 64 {
		t.Errorf("Wrong length for secret: %d", len(secret1))
	}
}
