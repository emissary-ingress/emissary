// Copyright 2020 Datawire.  All rights reserved.

package derrgroup_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/ambassador/pkg/derrgroup"
)

func TestNameCollision(t *testing.T) {
	assert := assert.New(t)
	group := new(derrgroup.Group)
	group.Go("foo", func() error { return nil })
	assert.NoError(group.Wait())
	group.Go("bar", func() error { return nil })
	assert.NoError(group.Wait())
	group.Go("foo", func() error { return nil })
	assert.Error(group.Wait())
}
