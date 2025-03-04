package gocb

import (
	"time"
)

// Bucket represents a single bucket within a cluster.
type Bucket struct {
	bucketName string

	timeoutsConfig TimeoutsConfig

	transcoder           Transcoder
	retryStrategyWrapper *coreRetryStrategyWrapper
	tracer               RequestTracer
	meter                *meterWrapper

	useServerDurations bool
	useMutationTokens  bool

	bootstrapError    error
	connectionManager connectionManager
	getTransactions   func() *Transactions
}

func newBucket(c *Cluster, bucketName string) *Bucket {
	return &Bucket{
		bucketName: bucketName,

		timeoutsConfig: c.timeoutsConfig,

		transcoder: c.transcoder,

		retryStrategyWrapper: c.retryStrategyWrapper,

		tracer: c.tracer,
		meter:  c.meter,

		useServerDurations: c.useServerDurations,
		useMutationTokens:  c.useMutationTokens,

		connectionManager: c.connectionManager,
		getTransactions:   c.Transactions,
	}
}

func (b *Bucket) setBootstrapError(err error) {
	b.bootstrapError = err
}

func (b *Bucket) getKvProvider() (kvProvider, error) {
	if b.bootstrapError != nil {
		return nil, b.bootstrapError
	}

	agent, err := b.connectionManager.getKvProvider(b.bucketName)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (b *Bucket) getKvCapabilitiesProvider() (kvCapabilityVerifier, error) {
	if b.bootstrapError != nil {
		return nil, b.bootstrapError
	}

	agent, err := b.connectionManager.getKvCapabilitiesProvider(b.bucketName)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (b *Bucket) getKvBulkProvider() (kvBulkProvider, error) {
	if b.bootstrapError != nil {
		return nil, b.bootstrapError
	}

	agent, err := b.connectionManager.getKvBulkProvider(b.bucketName)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (b *Bucket) getQueryProvider() (queryProvider, error) {
	if b.bootstrapError != nil {
		return nil, b.bootstrapError
	}

	agent, err := b.connectionManager.getQueryProvider()
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (b *Bucket) getQueryIndexProvider() (queryIndexProvider, error) {
	provider, err := b.connectionManager.getQueryIndexProvider()
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (b *Bucket) getAnalyticsProvider() (analyticsProvider, error) {
	if b.bootstrapError != nil {
		return nil, b.bootstrapError
	}

	agent, err := b.connectionManager.getAnalyticsProvider()
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// Name returns the name of the bucket.
func (b *Bucket) Name() string {
	return b.bucketName
}

// Scope returns an instance of a Scope.
func (b *Bucket) Scope(scopeName string) *Scope {
	return newScope(b, scopeName)
}

// DefaultScope returns an instance of the default scope.
func (b *Bucket) DefaultScope() *Scope {
	return b.Scope("_default")
}

// Collection returns an instance of a collection from within the default scope.
func (b *Bucket) Collection(collectionName string) *Collection {
	return b.DefaultScope().Collection(collectionName)
}

// DefaultCollection returns an instance of the default collection.
func (b *Bucket) DefaultCollection() *Collection {
	return b.DefaultScope().Collection("_default")
}

// ViewIndexes returns a ViewIndexManager instance for managing views.
func (b *Bucket) ViewIndexes() *ViewIndexManager {
	return &ViewIndexManager{
		mgmtProvider: b,
		bucketName:   b.Name(),
		tracer:       b.tracer,
		meter:        b.meter,
	}
}

// CollectionsV2 provides functions for managing collections.
// # UNCOMMITTED: This API may change in the future.
func (b *Bucket) CollectionsV2() *CollectionManagerV2 {
	return &CollectionManagerV2{
		getProvider: func() (collectionsManagementProvider, error) {
			return b.connectionManager.getCollectionsManagementProvider(b.Name())
		},
	}
}

// Collections provides functions for managing collections.
// Will be deprecated in favor of CollectionsV2 in the next minor release.
func (b *Bucket) Collections() *CollectionManager {
	// TODO: return error for unsupported collections
	return &CollectionManager{
		managerV2: b.CollectionsV2(),
	}
}

// WaitUntilReady will wait for the bucket object to be ready for use.
// At present this will wait until memd connections have been established with the server and are ready
// to be used before performing a ping against the specified services (except KeyValue) which also
// exist in the cluster map.
// If no services are specified then will wait until KeyValue is ready.
// Valid service types are: ServiceTypeKeyValue, ServiceTypeManagement, ServiceTypeQuery, ServiceTypeSearch,
// ServiceTypeAnalytics, ServiceTypeViews.
func (b *Bucket) WaitUntilReady(timeout time.Duration, opts *WaitUntilReadyOptions) error {
	if opts == nil {
		opts = &WaitUntilReadyOptions{}
	}

	if b.bootstrapError != nil {
		return b.bootstrapError
	}

	provider, err := b.connectionManager.getWaitUntilReadyProvider(b.bucketName)
	if err != nil {
		return err
	}

	err = provider.WaitUntilReady(
		opts.Context,
		time.Now().Add(timeout),
		opts,
	)
	if err != nil {
		return err
	}

	return nil
}
