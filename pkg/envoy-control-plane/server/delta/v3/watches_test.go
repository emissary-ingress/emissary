package delta

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
)

func TestDeltaWatches(t *testing.T) {
	t.Run("watches response channels are properly closed when the watches are cancelled", func(t *testing.T) {
		watches := newWatches()

		cancelCount := 0
		var channels []chan cache.DeltaResponse
		// create a few watches, and ensure that the cancel function are called and the channels are closed
		for i := 0; i < 5; i++ {
			newWatch := watch{}
			if i%2 == 0 {
				newWatch.cancel = func() { cancelCount++ }
				newWatch.responses = make(chan cache.DeltaResponse)
				channels = append(channels, newWatch.responses)
			}

			watches.deltaWatches[strconv.Itoa(i)] = newWatch
		}

		watches.Cancel()

		assert.Equal(t, 3, cancelCount)
		for _, channel := range channels {
			select {
			case _, ok := <-channel:
				assert.False(t, ok, "a channel was not closed")
			default:
				assert.Fail(t, "a channel was not closed")
			}
		}
	})
}
