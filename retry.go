package gocb

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/couchbase/gocbcore/v10"
)

// RetryRequest is a request that can possibly be retried.
type RetryRequest interface {
	RetryAttempts() uint32
	Identifier() string
	Idempotent() bool
	RetryReasons() []RetryReason
}

// RetryReason represents the reason for an operation possibly being retried.
type RetryReason interface {
	AllowsNonIdempotentRetry() bool
	AlwaysRetry() bool
	Description() string
}

var (
	// UnknownRetryReason indicates that the operation failed for an unknown reason.
	UnknownRetryReason = RetryReason(gocbcore.UnknownRetryReason)

	// SocketNotAvailableRetryReason indicates that the operation failed because the underlying socket was not available.
	SocketNotAvailableRetryReason = RetryReason(gocbcore.SocketNotAvailableRetryReason)

	// ServiceNotAvailableRetryReason indicates that the operation failed because the requested service was not available.
	ServiceNotAvailableRetryReason = RetryReason(gocbcore.ServiceNotAvailableRetryReason)

	// NodeNotAvailableRetryReason indicates that the operation failed because the requested node was not available.
	NodeNotAvailableRetryReason = RetryReason(gocbcore.NodeNotAvailableRetryReason)

	// KVNotMyVBucketRetryReason indicates that the operation failed because it was sent to the wrong node for the vbucket.
	KVNotMyVBucketRetryReason = RetryReason(gocbcore.KVNotMyVBucketRetryReason)

	// KVCollectionOutdatedRetryReason indicates that the operation failed because the collection ID on the request is outdated.
	KVCollectionOutdatedRetryReason = RetryReason(gocbcore.KVCollectionOutdatedRetryReason)

	// KVErrMapRetryReason indicates that the operation failed for an unsupported reason but the KV error map indicated
	// that the operation can be retried.
	KVErrMapRetryReason = RetryReason(gocbcore.KVErrMapRetryReason)

	// KVLockedRetryReason indicates that the operation failed because the document was locked.
	KVLockedRetryReason = RetryReason(gocbcore.KVLockedRetryReason)

	// KVTemporaryFailureRetryReason indicates that the operation failed because of a temporary failure.
	KVTemporaryFailureRetryReason = RetryReason(gocbcore.KVTemporaryFailureRetryReason)

	// KVSyncWriteInProgressRetryReason indicates that the operation failed because a sync write is in progress.
	KVSyncWriteInProgressRetryReason = RetryReason(gocbcore.KVSyncWriteInProgressRetryReason)

	// KVSyncWriteRecommitInProgressRetryReason indicates that the operation failed because a sync write recommit is in progress.
	KVSyncWriteRecommitInProgressRetryReason = RetryReason(gocbcore.KVSyncWriteRecommitInProgressRetryReason)

	// ServiceResponseCodeIndicatedRetryReason indicates that the operation failed and the service responded stating that
	// the request should be retried.
	ServiceResponseCodeIndicatedRetryReason = RetryReason(gocbcore.ServiceResponseCodeIndicatedRetryReason)

	// SocketCloseInFlightRetryReason indicates that the operation failed because the socket was closed whilst the operation
	// was in flight.
	SocketCloseInFlightRetryReason = RetryReason(gocbcore.SocketCloseInFlightRetryReason)

	// CircuitBreakerOpenRetryReason indicates that the operation failed because the circuit breaker on the connection
	// was open.
	CircuitBreakerOpenRetryReason = RetryReason(gocbcore.CircuitBreakerOpenRetryReason)

	// QueryIndexNotFoundRetryReason indicates that the operation failed to to a missing query index
	QueryIndexNotFoundRetryReason = RetryReason(gocbcore.QueryIndexNotFoundRetryReason)

	// QueryPreparedStatementFailureRetryReason indicates that the operation failed due to a prepared statement failure
	QueryPreparedStatementFailureRetryReason = RetryReason(gocbcore.QueryPreparedStatementFailureRetryReason)

	// AnalyticsTemporaryFailureRetryReason indicates that an analytics operation failed due to a temporary failure
	AnalyticsTemporaryFailureRetryReason = RetryReason(gocbcore.AnalyticsTemporaryFailureRetryReason)

	// SearchTooManyRequestsRetryReason indicates that a search operation failed due to too many requests
	SearchTooManyRequestsRetryReason = RetryReason(gocbcore.SearchTooManyRequestsRetryReason)

	// QueryErrorRetryable indicates that the operation is retryable as indicated by the query engine.
	// Uncommitted: This API may change in the future.
	QueryErrorRetryable = RetryReason(gocbcore.QueryErrorRetryable)

	// NotReadyRetryReason indicates the SDK connections are not setup and ready to be used.
	NotReadyRetryReason = RetryReason(gocbcore.NotReadyRetryReason)
)

// RetryAction is used by a RetryStrategy to calculate the duration to wait before retrying an operation.
// Returning a value of 0 indicates to not retry.
type RetryAction interface {
	Duration() time.Duration
}

// NoRetryRetryAction represents an action that indicates to not retry.
type NoRetryRetryAction struct {
}

// Duration is the length of time to wait before retrying an operation.
func (ra *NoRetryRetryAction) Duration() time.Duration {
	return 0
}

// WithDurationRetryAction represents an action that indicates to retry with a given duration.
type WithDurationRetryAction struct {
	WithDuration time.Duration
}

// Duration is the length of time to wait before retrying an operation.
func (ra *WithDurationRetryAction) Duration() time.Duration {
	return ra.WithDuration
}

// RetryStrategy is to determine if an operation should be retried, and if so how long to wait before retrying.
type RetryStrategy interface {
	RetryAfter(req RetryRequest, reason RetryReason) RetryAction
}

// BackoffCalculator defines how backoff durations will be calculated by the retry API.
type BackoffCalculator func(retryAttempts uint32) time.Duration

// BestEffortRetryStrategy represents a strategy that will keep retrying until it succeeds (or the caller times out
// the request).
type BestEffortRetryStrategy struct {
	BackoffCalculator BackoffCalculator
}

// NewBestEffortRetryStrategy returns a new BestEffortRetryStrategy which will use the supplied calculator function
// to calculate retry durations. If calculator is nil then a controlled backoff will be used.
func NewBestEffortRetryStrategy(calculator BackoffCalculator) *BestEffortRetryStrategy {
	if calculator == nil {
		calculator = BackoffCalculator(gocbcore.ExponentialBackoff(1*time.Millisecond, 500*time.Millisecond, 2))
	}

	return &BestEffortRetryStrategy{BackoffCalculator: calculator}
}

// RetryAfter calculates and returns a RetryAction describing how long to wait before retrying an operation.
func (rs *BestEffortRetryStrategy) RetryAfter(req RetryRequest, reason RetryReason) RetryAction {
	if req.Idempotent() || reason.AllowsNonIdempotentRetry() {
		return &WithDurationRetryAction{WithDuration: rs.BackoffCalculator(req.RetryAttempts())}
	}

	return &NoRetryRetryAction{}
}

type internalRetryRequest interface {
	RetryAttempts() uint32
	Identifier() string
	Idempotent() bool
	RetryReasons() []RetryReason

	retryStrategy() RetryStrategy
	recordRetryAttempt(reason RetryReason)
}

// retryOrchMaybeRetry will possibly retry an operation according to the strategy belonging to the request.
// It will use the reason to determine whether or not the failure reason is one that can be retried.
func retryOrchMaybeRetry(req internalRetryRequest, reason RetryReason) (bool, time.Time) {
	if reason.AlwaysRetry() {
		duration := gocbcore.ControlledBackoff(req.RetryAttempts())
		logDebugf("Will retry request. Backoff=%s, OperationID=%s. Reason=%s", duration, req.Identifier(), reason)

		req.recordRetryAttempt(reason)

		return true, time.Now().Add(duration)
	}

	retryStrategy := req.retryStrategy()
	if retryStrategy == nil {
		return false, time.Time{}
	}

	action := retryStrategy.RetryAfter(req, reason)
	if action == nil {
		logDebugf("Won't retry request.  OperationID=%s. Reason=%s", req.Identifier(), reason)
		return false, time.Time{}
	}

	duration := action.Duration()
	if duration == 0 {
		logDebugf("Won't retry request.  OperationID=%s. Reason=%s", req.Identifier(), reason)
		return false, time.Time{}
	}

	logDebugf("Will retry request. Backoff=%s, OperationID=%s. Reason=%s", duration, req.Identifier(), reason)
	req.recordRetryAttempt(reason)

	return true, time.Now().Add(duration)
}

type retriableRequestPs struct {
	// reasons is effectively a set, so we can't just use len(reasons) for num attempts.
	reasons  []RetryReason
	attempts uint32

	operation        string
	identifier       string
	idempotent       bool
	sendFn           func(ctx context.Context) (interface{}, error)
	strategy         RetryStrategy
	rootTraceContext RequestSpanContext
}

func newRetriableRequestPS(operation string, idempotent bool, rootContext RequestSpanContext, strategy RetryStrategy,
	sendFn func(ctx context.Context) (interface{}, error)) *retriableRequestPs {
	return &retriableRequestPs{
		operation:        operation,
		identifier:       uuid.NewString()[:6],
		idempotent:       idempotent,
		sendFn:           sendFn,
		rootTraceContext: rootContext,
		strategy:         strategy,
	}
}

func (w *retriableRequestPs) RetryAttempts() uint32 {
	return w.attempts
}

func (w *retriableRequestPs) Identifier() string {
	return w.identifier
}

func (w *retriableRequestPs) Idempotent() bool {
	return w.idempotent
}

func (w *retriableRequestPs) RetryReasons() []RetryReason {
	return w.reasons
}

func (w *retriableRequestPs) Send(ctx context.Context) (interface{}, error) {
	return w.sendFn(ctx)
}

func (w *retriableRequestPs) Operation() string {
	return w.operation
}

func (w *retriableRequestPs) retryStrategy() RetryStrategy {
	return w.strategy
}

func (w *retriableRequestPs) recordRetryAttempt(reason RetryReason) {
	w.attempts++
	found := false
	for i := 0; i < len(w.reasons); i++ {
		if w.reasons[i] == reason {
			found = true
			break
		}
	}

	// if idx is out of the range of retryReasons then it wasn't found.
	if !found {
		w.reasons = append(w.reasons, reason)
	}
}

func handleRetriableRequest(ctx context.Context, createdTime time.Time, tracer RequestTracer, req *retriableRequestPs,
	retryReasonFn func(err error) RetryReason) (interface{}, error) {
	for {
		logSchedf("Writing request ID=%s, OP=%s", req.identifier, req.operation)
		span := tracer.RequestSpan(req.rootTraceContext, "dispatch_to_server")
		res, err := req.Send(ctx)
		span.End()
		logSchedf("Handling response ID=%s, OP=%s", req.identifier, req.operation)

		if err != nil {
			gocbErr := mapPsErrorToGocbError(err, req.Idempotent())

			if errors.Is(gocbErr, ErrTimeout) {
				return nil, &TimeoutError{
					InnerError:    err,
					OperationID:   req.Operation(),
					Opaque:        req.Identifier(),
					TimeObserved:  time.Since(createdTime),
					RetryReasons:  req.RetryReasons(),
					RetryAttempts: req.RetryAttempts(),
				}
			}

			retryReason := retryReasonFn(gocbErr)
			if retryReason == nil {
				return nil, gocbErr
			}

			shouldRetry, retryWait := retryOrchMaybeRetry(req, retryReason)
			if !shouldRetry {
				return nil, gocbErr
			}

			select {
			case <-time.After(time.Until(retryWait)):
				continue
			case <-ctx.Done():
				err := ctx.Err()
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, &TimeoutError{
						InnerError:    ErrUnambiguousTimeout,
						OperationID:   req.Operation(),
						Opaque:        req.Identifier(),
						TimeObserved:  time.Since(createdTime),
						RetryReasons:  req.RetryReasons(),
						RetryAttempts: req.RetryAttempts(),
					}
				} else {
					return nil, makeGenericError(ErrRequestCanceled, nil)
				}
			}
		}

		return res, nil
	}
}
