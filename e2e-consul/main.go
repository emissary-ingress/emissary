package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if ok, err := TestTLSSecretExists(); !ok {
		fmt.Println(err)
		os.Exit(1)
	}
}

func TestTLSSecretExists() (bool, error) {
	timeout := time.After(60 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return false, errors.New("timed out")
		case <-tick:
			fmt.Println("Checking for secret...")
			data, err := kubectlGetSecret("", "ambassador-consul-connect")
			if err != nil {
				return false, err
			} else if data != "" {
				fmt.Println(data)
				return true, nil
			}
		}
	}
}

func kubectlGetSecret(namespace string, name string) (string, error) {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return "", err
	}

	namespaceArg := make([]string, 0)
	if namespace == "" {
		namespaceArg = append(namespaceArg, "--namespace=" + namespace)
	}

	args := []string{"get", "secret", name, "--output=json", "--ignore-not-found"}
	args = append(args, namespaceArg...)
	cmd := exec.Command(kubectl, args...)
	cmd.Env = []string{fmt.Sprintf("KUBECONFIG=%s", os.Getenv("KUBECONFIG"))}

	out, err := cmd.Output()
	return string(out), err
}
