package gocb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

type searchIndexProviderCore struct {
	mgmtProvider mgmtProvider

	tracer RequestTracer
	meter  *meterWrapper
}

var _ searchIndexProvider = (*searchIndexProviderCore)(nil)

func (sm *searchIndexProviderCore) GetAllIndexes(opts *GetAllSearchIndexOptions) ([]SearchIndex, error) {
	if opts == nil {
		opts = &GetAllSearchIndexOptions{}
	}

	start := time.Now()
	defer sm.meter.ValueRecord(meterValueServiceManagement, "manager_search_get_all_indexes", start)

	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_get_all_indexes", "management")
	span.SetAttribute("db.operation", "GET /api/index")
	defer span.End()

	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        "GET",
		Path:          "/api/index",
		IsIdempotent:  true,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return nil, err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return nil, idxErr
		}

		return nil, makeMgmtBadStatusError("failed to get index", &req, resp)
	}

	var indexesResp jsonSearchIndexesResp
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&indexesResp)
	if err != nil {
		return nil, err
	}

	indexDefs := indexesResp.IndexDefs.IndexDefs
	var indexes []SearchIndex
	for _, indexData := range indexDefs {
		var index SearchIndex
		err := index.fromData(indexData)
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (sm *searchIndexProviderCore) GetIndex(indexName string, opts *GetSearchIndexOptions) (*SearchIndex, error) {
	if opts == nil {
		opts = &GetSearchIndexOptions{}
	}

	start := time.Now()
	defer sm.meter.ValueRecord(meterValueServiceManagement, "manager_search_get_index", start)

	path := fmt.Sprintf("/api/index/%s", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_get_index", "management")
	span.SetAttribute("db.operation", "GET "+path)
	defer span.End()

	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        "GET",
		Path:          path,
		IsIdempotent:  true,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return nil, err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return nil, idxErr
		}

		return nil, makeMgmtBadStatusError("failed to get index", &req, resp)
	}

	var indexResp jsonSearchIndexResp
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&indexResp)
	if err != nil {
		return nil, err
	}

	var indexDef SearchIndex
	err = indexDef.fromData(*indexResp.IndexDef)
	if err != nil {
		return nil, err
	}

	return &indexDef, nil
}

func (sm *searchIndexProviderCore) UpsertIndex(indexDefinition SearchIndex, opts *UpsertSearchIndexOptions) error {
	if opts == nil {
		opts = &UpsertSearchIndexOptions{}
	}

	if indexDefinition.Name == "" {
		return invalidArgumentsError{"index name cannot be empty"}
	}
	if indexDefinition.Type == "" {
		return invalidArgumentsError{"index type cannot be empty"}
	}

	start := time.Now()
	defer sm.meter.ValueRecord(meterValueServiceManagement, "manager_search_upsert_index", start)

	path := fmt.Sprintf("/api/index/%s", url.PathEscape(indexDefinition.Name))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_upsert_index", "management")
	span.SetAttribute("db.operation", "PUT "+path)
	defer span.End()

	indexData, err := indexDefinition.toData()
	if err != nil {
		return err
	}

	b, err := json.Marshal(indexData)
	if err != nil {
		return err
	}

	req := mgmtRequest{
		Service: ServiceTypeSearch,
		Method:  "PUT",
		Path:    path,
		Headers: map[string]string{
			"cache-control": "no-cache",
		},
		Body:          b,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return idxErr
		}

		return makeMgmtBadStatusError("failed to create index", &req, resp)
	}

	return nil
}

func (sm *searchIndexProviderCore) DropIndex(indexName string, opts *DropSearchIndexOptions) error {
	if opts == nil {
		opts = &DropSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	start := time.Now()
	defer sm.meter.ValueRecord(meterValueServiceManagement, "manager_search_drop_index", start)

	path := fmt.Sprintf("/api/index/%s", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_drop_index", "management")
	span.SetAttribute("db.operation", "DELETE "+path)
	defer span.End()

	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        "DELETE",
		Path:          path,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		return makeMgmtBadStatusError("failed to drop the index", &req, resp)
	}

	return nil
}

func (sm *searchIndexProviderCore) AnalyzeDocument(indexName string, doc interface{}, opts *AnalyzeDocumentOptions) ([]interface{}, error) {
	if opts == nil {
		opts = &AnalyzeDocumentOptions{}
	}

	if indexName == "" {
		return nil, invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/analyzeDoc", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_analyze_document", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	b, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        "POST",
		Path:          path,
		Body:          b,
		IsIdempotent:  true,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return nil, err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return nil, idxErr
		}

		return nil, makeMgmtBadStatusError("failed to analyze document", &req, resp)
	}

	var analysis struct {
		Status   string        `json:"status"`
		Analyzed []interface{} `json:"analyzed"`
	}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&analysis)
	if err != nil {
		return nil, err
	}

	return analysis.Analyzed, nil
}

func (sm *searchIndexProviderCore) GetIndexedDocumentsCount(indexName string, opts *GetIndexedDocumentsCountOptions) (uint64, error) {
	if opts == nil {
		opts = &GetIndexedDocumentsCountOptions{}
	}

	if indexName == "" {
		return 0, invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/count", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_get_indexed_documents_count", "management")
	span.SetAttribute("db.operation", "GET "+path)
	defer span.End()

	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        "GET",
		Path:          path,
		IsIdempotent:  true,
		RetryStrategy: opts.RetryStrategy,
		Timeout:       opts.Timeout,
		parentSpanCtx: span.Context(),
	}
	resp, err := sm.doMgmtRequest(opts.Context, req)
	if err != nil {
		return 0, err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return 0, idxErr
		}

		return 0, makeMgmtBadStatusError("failed to get the indexed documents count", &req, resp)
	}

	var count struct {
		Count uint64 `json:"count"`
	}
	jsonDec := json.NewDecoder(resp.Body)
	err = jsonDec.Decode(&count)
	if err != nil {
		return 0, err
	}

	return count.Count, nil
}

func (sm *searchIndexProviderCore) PauseIngest(indexName string, opts *PauseIngestSearchIndexOptions) error {
	if opts == nil {
		opts = &PauseIngestSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/ingestControl/pause", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_pause_ingest", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) ResumeIngest(indexName string, opts *ResumeIngestSearchIndexOptions) error {
	if opts == nil {
		opts = &ResumeIngestSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/ingestControl/resume", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_resume_ingest", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) AllowQuerying(indexName string, opts *AllowQueryingSearchIndexOptions) error {
	if opts == nil {
		opts = &AllowQueryingSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/queryControl/allow", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_allow_querying", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) DisallowQuerying(indexName string, opts *AllowQueryingSearchIndexOptions) error {
	if opts == nil {
		opts = &AllowQueryingSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/queryControl/disallow", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_disallow_querying", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) FreezePlan(indexName string, opts *AllowQueryingSearchIndexOptions) error {
	if opts == nil {
		opts = &AllowQueryingSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/planFreezeControl/freeze", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_freeze_plan", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) UnfreezePlan(indexName string, opts *AllowQueryingSearchIndexOptions) error {
	if opts == nil {
		opts = &AllowQueryingSearchIndexOptions{}
	}

	if indexName == "" {
		return invalidArgumentsError{"indexName cannot be empty"}
	}

	path := fmt.Sprintf("/api/index/%s/planFreezeControl/unfreeze", url.PathEscape(indexName))
	span := createSpan(sm.tracer, opts.ParentSpan, "manager_search_unfreeze_plan", "management")
	span.SetAttribute("db.operation", "POST "+path)
	defer span.End()

	return sm.performControlRequest(
		opts.Context,
		span.Context(),
		"POST",
		path,
		opts.Timeout,
		opts.RetryStrategy)
}

func (sm *searchIndexProviderCore) performControlRequest(
	ctx context.Context,
	tracectx RequestSpanContext,
	method, uri string,
	timeout time.Duration,
	retryStrategy RetryStrategy,
) error {
	req := mgmtRequest{
		Service:       ServiceTypeSearch,
		Method:        method,
		Path:          uri,
		IsIdempotent:  true,
		Timeout:       timeout,
		RetryStrategy: retryStrategy,
		parentSpanCtx: tracectx,
	}

	resp, err := sm.doMgmtRequest(ctx, req)
	if err != nil {
		return err
	}
	defer ensureBodyClosed(resp.Body)

	if resp.StatusCode != 200 {
		idxErr := sm.tryParseErrorMessage(&req, resp)
		if idxErr != nil {
			return idxErr
		}

		return makeMgmtBadStatusError("failed to perform the control request", &req, resp)
	}

	return nil
}

func (sm *searchIndexProviderCore) checkForRateLimitError(statusCode uint32, errMsg string) error {
	errMsg = strings.ToLower(errMsg)

	var err error
	if statusCode == 400 && strings.Contains(errMsg, "num_fts_indexes") {
		err = ErrQuotaLimitedFailure
	} else if statusCode == 429 {
		if strings.Contains(errMsg, "num_concurrent_requests") {
			err = ErrRateLimitedFailure
		} else if strings.Contains(errMsg, "num_queries_per_min") {
			err = ErrRateLimitedFailure
		} else if strings.Contains(errMsg, "ingress_mib_per_min") {
			err = ErrRateLimitedFailure
		} else if strings.Contains(errMsg, "egress_mib_per_min") {
			err = ErrRateLimitedFailure
		}
	}

	return err
}

func (sm *searchIndexProviderCore) tryParseErrorMessage(req *mgmtRequest, resp *mgmtResponse) error {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		logDebugf("Failed to read search index response body: %s", err)
		return nil
	}

	if err := sm.checkForRateLimitError(resp.StatusCode, string(b)); err != nil {
		return makeGenericMgmtError(err, req, resp, string(b))
	}

	var bodyErr error
	if strings.Contains(strings.ToLower(string(b)), "index not found") {
		bodyErr = ErrIndexNotFound
	} else if strings.Contains(strings.ToLower(string(b)), "index with the same name already exists") {
		bodyErr = ErrIndexExists
	} else {
		bodyErr = errors.New(string(b))
	}

	return makeGenericMgmtError(bodyErr, req, resp, string(b))
}

func (sm *searchIndexProviderCore) doMgmtRequest(ctx context.Context, req mgmtRequest) (*mgmtResponse, error) {
	resp, err := sm.mgmtProvider.executeMgmtRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

type jsonSearchIndexResp struct {
	Status   string           `json:"status"`
	IndexDef *jsonSearchIndex `json:"indexDef"`
}

type jsonSearchIndexDefs struct {
	IndexDefs   map[string]jsonSearchIndex `json:"indexDefs"`
	ImplVersion string                     `json:"implVersion"`
}

type jsonSearchIndexesResp struct {
	Status    string              `json:"status"`
	IndexDefs jsonSearchIndexDefs `json:"indexDefs"`
}

type jsonSearchIndex struct {
	UUID         string                 `json:"uuid"`
	Name         string                 `json:"name"`
	SourceName   string                 `json:"sourceName"`
	Type         string                 `json:"type"`
	Params       map[string]interface{} `json:"params"`
	SourceUUID   string                 `json:"sourceUUID"`
	SourceParams map[string]interface{} `json:"sourceParams"`
	SourceType   string                 `json:"sourceType"`
	PlanParams   map[string]interface{} `json:"planParams"`
}
