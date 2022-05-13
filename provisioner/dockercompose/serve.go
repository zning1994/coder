package dockercompose

import (
	"context"

	"cdr.dev/slog"
	"github.com/coder/coder/provisionersdk"
)

// Serve starts the echo provisioner.
func Serve(ctx context.Context, options *provisionersdk.ServeOptions, log slog.Logger) error {
	return provisionersdk.Serve(ctx, &provisionerServer{log: log}, options)
}

type provisionerServer struct {
	log slog.Logger
}
