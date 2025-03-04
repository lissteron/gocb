package gocb

// Scope represents a single scope within a bucket.
type Scope struct {
	scopeName string
	bucket    *Bucket

	timeoutsConfig TimeoutsConfig

	transcoder           Transcoder
	retryStrategyWrapper *coreRetryStrategyWrapper
	tracer               RequestTracer
	meter                *meterWrapper

	useMutationTokens bool

	getKvProvider         func() (kvProvider, error)
	getKvBulkProvider     func() (kvBulkProvider, error)
	getQueryProvider      func() (queryProvider, error)
	getQueryIndexProvider func() (queryIndexProvider, error)
	getAnalyticsProvider  func() (analyticsProvider, error)
	getTransactions       func() *Transactions
}

func newScope(bucket *Bucket, scopeName string) *Scope {
	return &Scope{
		scopeName: scopeName,
		bucket:    bucket,

		timeoutsConfig: bucket.timeoutsConfig,

		transcoder:           bucket.transcoder,
		retryStrategyWrapper: bucket.retryStrategyWrapper,
		tracer:               bucket.tracer,
		meter:                bucket.meter,

		useMutationTokens: bucket.useMutationTokens,

		getKvProvider:         bucket.getKvProvider,
		getKvBulkProvider:     bucket.getKvBulkProvider,
		getQueryProvider:      bucket.getQueryProvider,
		getQueryIndexProvider: bucket.getQueryIndexProvider,
		getAnalyticsProvider:  bucket.getAnalyticsProvider,
		getTransactions:       bucket.getTransactions,
	}
}

// Name returns the name of the scope.
func (s *Scope) Name() string {
	return s.scopeName
}

// BucketName returns the name of the bucket to which this collection belongs.
// UNCOMMITTED: This API may change in the future.
func (s *Scope) BucketName() string {
	return s.bucket.Name()
}

// Collection returns an instance of a collection.
func (s *Scope) Collection(collectionName string) *Collection {
	return newCollection(s, collectionName)
}
