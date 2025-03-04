package gocb

import gocbcore "github.com/couchbase/gocbcore/v10"

type connectionManager interface {
	connect() error
	openBucket(bucketName string) error
	buildConfig(cluster *Cluster) error
	connection(bucketName string) (*gocbcore.Agent, error)
	close() error

	getKvProvider(bucketName string) (kvProvider, error)
	getKvBulkProvider(bucketName string) (kvBulkProvider, error)
	getKvCapabilitiesProvider(bucketName string) (kvCapabilityVerifier, error)
	getViewProvider(bucketName string) (viewProvider, error)
	getQueryProvider() (queryProvider, error)
	getQueryIndexProvider() (queryIndexProvider, error)
	getAnalyticsProvider() (analyticsProvider, error)
	getSearchProvider() (searchProvider, error)
	getHTTPProvider(bucketName string) (httpProvider, error)
	getDiagnosticsProvider(bucketName string) (diagnosticsProvider, error)
	getWaitUntilReadyProvider(bucketName string) (waitUntilReadyProvider, error)
	getCollectionsManagementProvider(bucketName string) (collectionsManagementProvider, error)
	getBucketManagementProvider() (bucketManagementProvider, error)
	getSearchIndexProvider() (searchIndexProvider, error)
}

func (c *Cluster) newConnectionMgr(protocol string) connectionManager {
	switch protocol {
	case "couchbase2":
		return &psConnectionMgr{
			timeouts:     c.timeoutsConfig,
			tracer:       c.tracer,
			meter:        c.meter,
			defaultRetry: c.retryStrategyWrapper.wrapped,
		}
	default:
		return &stdConnectionMgr{
			retryStrategyWrapper: c.retryStrategyWrapper,
			transcoder:           c.transcoder,
			timeouts:             c.timeoutsConfig,
			tracer:               c.tracer,
			meter:                c.meter,
		}
	}
}
