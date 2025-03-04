package gocb

import (
	"context"
	"time"

	"github.com/couchbase/gocbcore/v10"
)

// ScanOptions are the set of options available to the Scan operation.
// VOLATILE: This API is subject to change at any time.
type ScanOptions struct {
	Transcoder Transcoder
	Timeout    time.Duration
	ParentSpan RequestSpan

	// Using a deadlined Context alongside a Timeout will cause the shorter of the two to cause cancellation, this
	// also applies to global level timeouts.
	// UNCOMMITTED: This API may change in the future.
	Context context.Context

	IDsOnly        bool
	ConsistentWith *MutationState

	// BatchByteLimit specifies a limit to how many bytes are sent from server to client on each partition batch.
	// Defaults to 15000. A value of 0 is equivalent to no limit.
	BatchByteLimit *uint32
	// BatchItemLimit specifies a limit to how many items are sent from server to client on each partition batch.
	// Defaults to 50. A value of 0 is equivalent to no limit.
	BatchItemLimit *uint32

	// Internal: This should never be used and is not supported.
	Internal struct {
		User string
	}
}

// ScanTerm represents a term that can be used during a Scan operation.
type ScanTerm struct {
	Term      string
	Exclusive bool
}

// ScanTermMinimum represents the minimum value that a ScanTerm can represent.
func ScanTermMinimum() *ScanTerm {
	return &ScanTerm{
		Term: "\x00",
	}
}

// ScanTermMaximum represents the maximum value that a ScanTerm can represent.
func ScanTermMaximum() *ScanTerm {
	return &ScanTerm{
		Term: "\xf48fbfbf",
	}
}

// ScanType represents the mode of execution to use for a Scan operation.
type ScanType interface {
	isScanType()
}

// NewRangeScanForPrefix creates a new range scan for the given prefix, starting at the prefix and ending at the prefix
// plus maximum.
// VOLATILE: This API is subject to change at any time.
func NewRangeScanForPrefix(prefix string) RangeScan {
	return RangeScan{
		From: &ScanTerm{
			Term: prefix,
		},
		To: &ScanTerm{
			Term: prefix + ScanTermMaximum().Term,
		},
	}
}

// RangeScan indicates that the Scan operation should scan a range of keys.
type RangeScan struct {
	From *ScanTerm
	To   *ScanTerm
}

func (rs RangeScan) isScanType() {}

func (rs RangeScan) toCore() (*gocbcore.RangeScanCreateRangeScanConfig, error) {
	to := rs.To
	from := rs.From

	rangeOptions := &gocbcore.RangeScanCreateRangeScanConfig{}
	if from != nil {
		if from.Exclusive {
			rangeOptions.ExclusiveStart = []byte(from.Term)
		} else {
			rangeOptions.Start = []byte(from.Term)
		}
	}
	if to != nil {
		if to.Exclusive {
			rangeOptions.ExclusiveEnd = []byte(to.Term)
		} else {
			rangeOptions.End = []byte(to.Term)
		}
	}

	return rangeOptions, nil
}

// SamplingScan indicates that the Scan operation should perform random sampling.
type SamplingScan struct {
	Limit uint64
	Seed  uint64
}

func (rs SamplingScan) isScanType() {}

func (rs SamplingScan) toCore() (*gocbcore.RangeScanCreateRandomSamplingConfig, error) {
	if rs.Limit == 0 {
		return nil, makeInvalidArgumentsError("sampling scan limit must be greater than 0")
	}

	return &gocbcore.RangeScanCreateRandomSamplingConfig{
		Samples: rs.Limit,
		Seed:    rs.Seed,
	}, nil
}

// Scan performs a scan across a Collection, returning a stream of documents.
// VOLATILE: This API is subject to change at any time.
func (c *Collection) Scan(scanType ScanType, opts *ScanOptions) (*ScanResult, error) {
	if opts == nil {
		opts = &ScanOptions{}
	}

	agent, err := c.getKvProvider()
	if err != nil {
		return nil, err
	}

	if opts.Timeout == 0 {
		opts.Timeout = c.timeoutsConfig.KVScanTimeout
	}

	return agent.Scan(c, scanType, opts)
}
