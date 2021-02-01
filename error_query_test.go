package gocb

import "encoding/json"

func (suite *UnitTestSuite) TestQueryError() {
	aErr := QueryError{
		InnerError:      ErrIndexFailure,
		Statement:       "select * from dataset",
		ClientContextID: "12345",
		Errors: []QueryErrorDesc{{
			Code:    1000,
			Message: "error 1000",
		}},
		Endpoint:      "http://127.0.0.1:8093",
		RetryReasons:  []RetryReason{QueryIndexNotFoundRetryReason},
		RetryAttempts: 3,
	}

	b, err := json.Marshal(aErr)
	suite.Require().Nil(err)

	suite.Assert().Equal(
		[]byte("{\"msg\":\"index failure\",\"statement\":\"select * from dataset\",\"client_context_id\":\"12345\",\"errors\":[{\"Code\":1000,\"Message\":\"error 1000\"}],\"endpoint\":\"http://127.0.0.1:8093\",\"retry_reasons\":[\"QUERY_INDEX_NOT_FOUND\"],\"retry_attempts\":3}"),
		b,
	)
	suite.Assert().Equal(
		"index failure | {\"statement\":\"select * from dataset\",\"client_context_id\":\"12345\",\"errors\":[{\"Code\":1000,\"Message\":\"error 1000\"}],\"endpoint\":\"http://127.0.0.1:8093\",\"retry_reasons\":[\"QUERY_INDEX_NOT_FOUND\"],\"retry_attempts\":3}",
		aErr.Error(),
	)
}
