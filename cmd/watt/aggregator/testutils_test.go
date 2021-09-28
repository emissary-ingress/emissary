package aggregator_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/k8s"
)

var (
	CLOSE interface{} = &struct{}{}
)

type Timeout time.Duration

func expect(t *testing.T, ch interface{}, values ...interface{}) {
	rch := reflect.ValueOf(ch)

	for idx, expected := range values {
		var timeoutExpected bool
		var timer *time.Timer
		switch exp := expected.(type) {
		case Timeout:
			timeoutExpected = true
			timer = time.NewTimer(time.Duration(exp))
		default:
			timeoutExpected = false
			timer = time.NewTimer(10 * time.Second)
		}

		chosen, value, ok := reflect.Select([]reflect.SelectCase{
			{
				Dir:  reflect.SelectRecv,
				Chan: rch,
			},
			{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(timer.C),
			},
		})

		if timeoutExpected && chosen != 1 {
			if ok {
				panic(fmt.Sprintf("expected timeout, got %v", value.Interface()))
			} else {
				panic("expected timeout, got CLOSE")
			}
		}

		if timeoutExpected && chosen == 1 {
			continue
		}

		if ok {
			switch exp := expected.(type) {
			case func(interface{}) bool:
				if !exp(value) {
					panic(fmt.Sprintf("predicate %d failed value %v", idx, value))
				}
			case func(string) bool:
				if !exp(value.String()) {
					panic(fmt.Sprintf("predicate %d failed value %v", idx, value))
				}
			case func([]k8s.Resource) bool:
				val, ok := value.Interface().([]k8s.Resource)
				if !ok {
					panic(fmt.Sprintf("expected a []k8s.Resource, got %v", value.Type()))
				}
				if !exp(val) {
					panic(fmt.Sprintf("predicate %d failed value %v", idx, value))
				}
			case func([]watchapi.ConsulWatchSpec) bool:
				val, ok := value.Interface().([]watchapi.ConsulWatchSpec)
				if !ok {
					panic(fmt.Sprintf("expected a []ConsulWatchSpec, got %v", value.Type()))
				}
				if !exp(val) {
					panic(fmt.Sprintf("predicate %d failed value %v", idx, value))
				}
			default:
				require.Equal(t, expected, value.Interface(), "read unexpected value from channel")
			}
		} else if expected != CLOSE {
			panic(fmt.Sprintf("expected %v, got CLOSE", expected))
		}
	}
}
