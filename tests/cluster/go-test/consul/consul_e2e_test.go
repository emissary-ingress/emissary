// +build test

package consul_test

import (
	"fmt"
	"testing"
	"time"
)

func TestTLSSecretExists(t *testing.T) {
	t.Parallel()

	timeout := time.After(60 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			fatal(t, "timed out")
		case <-tick:
			fmt.Println("Checking for secret...")
			data, err := kubectlGetSecret("", "ambassador-consul-connect")
			if err != nil {
				fatal(t, err)
			} else if data != "" {
				fmt.Println(data)
				return
			}
		}
	}
}
