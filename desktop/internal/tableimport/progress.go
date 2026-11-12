package tableimport

import "context"

type progressKey struct{}

// WithProgress reports transport stages without model content or credentials.
// The callback is called synchronously by the request goroutine.
func WithProgress(ctx context.Context, callback func(string)) context.Context {
	return context.WithValue(ctx, progressKey{}, callback)
}

func reportProgress(ctx context.Context, stage string) {
	if callback, ok := ctx.Value(progressKey{}).(func(string)); ok && callback != nil && ctx.Err() == nil {
		callback(stage)
	}
}
