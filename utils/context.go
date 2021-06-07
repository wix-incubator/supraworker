package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type ContextKey int

const (
	CtxWorkerIdKey ContextKey = iota
	CtxJobIDKey
	CtxRequestTimeoutKey
	CtxRetryIdKey
)

var DefaultRequestTimeout = 120 * time.Second

// FromRequestTimeout generates a new context with the given context as its parent
// and stores the request timout with the context. The request timeout
// can be retrieved again using RequestTimeoutFromContext.
func FromRequestTimeout(ctx context.Context, requestTimeout time.Duration) context.Context {
	return context.WithValue(ctx, CtxRequestTimeoutKey, requestTimeout)
}

// RequestTimeoutFromContext returns the time duration stored in the context with
// FromRequestTimeout. If no request timeout was stored in the context, the second argument is
// false. Otherwise it is true.
func RequestTimeoutFromContext(ctx context.Context) (time.Duration, bool) {
	value, ok := ctx.Value(CtxRequestTimeoutKey).(time.Duration)
	if !ok {
		value = DefaultRequestTimeout
	}
	return value, ok
}

// RetryIDFromContext returns the retry ID stored in the context with
// FromRetryID. If no retry ID was stored in the context, the second argument is
// false. Otherwise it is true.
func RetryIDFromContext(ctx context.Context) (int, bool) {
	retryID, ok := ctx.Value(CtxRetryIdKey).(int)
	return retryID, ok
}

// FromRetryID generates a new context with the given context as its parent
// and stores the given retry ID with the context. The retry ID
// can be retrieved again using RetryIDFromContext.
func FromRetryID(ctx context.Context, workerID int) context.Context {
	return context.WithValue(ctx, CtxRetryIdKey, workerID)
}

// FromWorkerID generates a new context with the given context as its parent
// and stores the given worker ID with the context. The instance ID
// can be retrieved again using WorkerIDFromContext.
func FromWorkerID(ctx context.Context, workerID string) context.Context {
	return context.WithValue(ctx, CtxWorkerIdKey, workerID)
}

// WorkerIDFromContext returns the worker ID stored in the context with
// FromWorkerID. If no workerID was stored in the context, the second argument is
// false. Otherwise it is true.
func WorkerIDFromContext(ctx context.Context) (string, bool) {
	workerID, ok := ctx.Value(CtxWorkerIdKey).(string)
	return workerID, ok
}

// JobIDFromContext returns the job ID stored in the context with FromJobID. If
// no job ID was stored in the context, the second argument is false. Otherwise
// it is true.
func JobIDFromContext(ctx context.Context) (string, bool) {
	jobID, ok := ctx.Value(CtxJobIDKey).(string)
	return jobID, ok
}

// FromJobID generates a new context with the given context as its parent and
// stores the given job ID with the context. The job ID can be retrieved again
// using JobIDFromContext.
func FromJobID(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, CtxJobIDKey, jobID)
}

// LoggerFromContext returns a logrus.Entry with the PID of the current process
// set as a field, and also includes every field set using the From* functions
// this package.
func LoggerFromContext(ctx context.Context, logger *logrus.Entry) *logrus.Entry {
	if logger == nil {
		logger = logrus.WithFields(logrus.Fields{"package": "context"})
	}

	entry := logger.WithField("pid", os.Getpid())
	if ctx == nil {
		return entry
	}
	if workerID, ok := WorkerIDFromContext(ctx); ok {
		entry = entry.WithField("worker_id", workerID)
	}
	if retryID, ok := RetryIDFromContext(ctx); ok {
		entry = entry.WithField("retry", retryID)
	}
	jobID, hasJobID := JobIDFromContext(ctx)
	if hasJobID {
		entry = entry.WithField("job_id", jobID)
	}

	return entry
}
