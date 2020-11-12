package dcontext_test

import (
	"context"
)

type Data struct{}

func DoWork(_ context.Context) error { return nil }

func DoWorkOnData(_ context.Context, _ Data) error { return nil }

func DoShutdown(_ context.Context) error { return nil }
