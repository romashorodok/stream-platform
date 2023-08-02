package container

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/go-logr/zapr"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	CONTEXT_LOGR = "logr_logger"
)

func WithLogr(ctx context.Context) *logr.Logger {
	l, ok := ctx.Value(CONTEXT_LOGR).(*logr.Logger)

	if !ok {
		opts := zap.Options{Development: true}
		zaprLogger := zap.NewRaw(zap.UseFlagOptions(&opts))
		logrLogger := zapr.NewLogger(zaprLogger)
		return &logrLogger
	}

	return l
}
