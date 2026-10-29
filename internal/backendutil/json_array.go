package backendutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeLimitedArray decodes a provider array without allocating a slice for
// more entries than the caller's page limit.
func DecodeLimitedArray[T any](data []byte, maxItems int) ([]T, error) {
	if maxItems < 0 {
		return nil, fmt.Errorf("negative JSON array limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('[') {
		return nil, fmt.Errorf("expected JSON array")
	}
	rows := make([]T, 0, min(maxItems, 100))
	for decoder.More() {
		if len(rows) == maxItems {
			return nil, fmt.Errorf("provider returned more than %d items in a page", maxItems)
		}
		var row T
		if err := decoder.Decode(&row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing data after provider array")
		}
		return nil, err
	}
	return rows, nil
}
