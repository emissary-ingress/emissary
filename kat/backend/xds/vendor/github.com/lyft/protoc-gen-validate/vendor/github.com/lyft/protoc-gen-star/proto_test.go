package pgs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSyntax_SupportsRequiredPrefix(t *testing.T) {
	t.Parallel()
	assert.True(t, Proto2.SupportsRequiredPrefix())
	assert.False(t, Proto3.SupportsRequiredPrefix())
}

func TestProtoType_IsInt(t *testing.T) {
	t.Parallel()

	yes := []ProtoType{
		Int64T, UInt64T, SFixed64, SInt64, Fixed64T,
		Int32T, UInt32T, SFixed32, SInt32, Fixed32T,
	}

	no := []ProtoType{
		DoubleT, FloatT, BoolT, StringT,
		GroupT, MessageT, BytesT, EnumT,
	}

	for _, pt := range yes {
		assert.True(t, pt.IsInt())
	}

	for _, pt := range no {
		assert.False(t, pt.IsInt())
	}
}

func TestProtoType_IsNumeric(t *testing.T) {
	t.Parallel()

	yes := []ProtoType{
		Int64T, UInt64T, SFixed64, SInt64, Fixed64T,
		Int32T, UInt32T, SFixed32, SInt32, Fixed32T,
		DoubleT, FloatT,
	}

	no := []ProtoType{
		BoolT, StringT, GroupT,
		MessageT, BytesT, EnumT,
	}

	for _, pt := range yes {
		assert.True(t, pt.IsNumeric())
	}

	for _, pt := range no {
		assert.False(t, pt.IsNumeric())
	}
}

func TestProtoType_IsSlice(t *testing.T) {
	t.Parallel()

	yes := []ProtoType{BytesT}

	no := []ProtoType{
		Int64T, UInt64T, SFixed64, SInt64, Fixed64T,
		Int32T, UInt32T, SFixed32, SInt32, Fixed32T,
		DoubleT, FloatT, BoolT, StringT, GroupT,
		MessageT, EnumT,
	}

	for _, pt := range yes {
		assert.True(t, pt.IsSlice())
	}

	for _, pt := range no {
		assert.False(t, pt.IsSlice())
	}
}
