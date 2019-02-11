package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	harness "github.com/lyft/protoc-gen-validate/tests/harness/go"
)

func Work(wg *sync.WaitGroup, in <-chan TestCase, out chan<- TestResult) {
	for tc := range in {
		ok, skip := execTestCase(tc)
		out <- TestResult{ok, skip}
	}
	wg.Done()
}

func execTestCase(tc TestCase) (ok, skip bool) {
	any, err := ptypes.MarshalAny(tc.Message)
	if err != nil {
		log.Printf("unable to convert test case %q to Any - %v", tc.Name, err)
		return false, false
	}

	b, err := proto.Marshal(&harness.TestCase{Message: any})
	if err != nil {
		log.Printf("unable to marshal test case %q - %v", tc.Name, err)
		return false, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wg := new(sync.WaitGroup)
	wg.Add(len(Harnesses))

	errs := make(chan error, len(Harnesses))
	skips := make(chan string, len(Harnesses))

	for _, h := range Harnesses {
		h := h
		go func() {
			defer wg.Done()

			res, err := h.Exec(ctx, bytes.NewReader(b))
			if err != nil {
				errs <- err
				return
			}

			if res.Error {
				errs <- fmt.Errorf("%s: internal harness error: %s", h.Name, res.Reason)
			} else if res.Valid != tc.Valid {
				if res.AllowFailure {
					skips <- fmt.Sprintf("%s: ignoring test failure: %s", h.Name, res.Reason)
				} else if tc.Valid {
					errs <- fmt.Errorf("%s: expected valid, got: %s", h.Name, res.Reason)
				} else {
					errs <- fmt.Errorf("%s: expected invalid, but got valid", h.Name)
				}
			}
		}()
	}

	wg.Wait()
	close(errs)
	close(skips)

	ok = true

	for err := range errs {
		log.Printf("[%s] %v", tc.Name, err)
		ok = false
	}
	for out := range skips {
		log.Printf("[%s] %v", tc.Name, out)
		skip = true
	}

	return
}
