package delta

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeltaWatches(t *testing.T) {
	t.Run("watches response channels are properly closed when the watches are canceled", func(t *testing.T) {
		watches := newWatches()

		cancelCount := 0
		// create a few watches, and ensure that the cancel function are called and the channels are closed
		for i := 0; i < 5; i++ {
			newWatch := watch{}
			if i%2 == 0 {
				newWatch.cancel = func() { cancelCount++ }
			}

			watches.deltaWatches[strconv.Itoa(i)] = newWatch
		}

		watches.Cancel()

		assert.Equal(t, 3, cancelCount)
	})
}
