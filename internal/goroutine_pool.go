package internal

import (
	"context"
	"goroutine_pool/internal/types"
)

type GoroutinePool interface {
	Submit(context.Context, *types.Work)
	Init(context.Context, int)
	Run(context.Context)
	Result(ctx context.Context) (*types.Work, bool)
	Stop()
	ToString() string
}
