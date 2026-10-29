package backendutil

import (
	"errors"
	"fmt"
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

const (
	MaxUtxosPerQuery     = 10_000
	MaxUtxoBytesPerQuery = 32 * 1024 * 1024
)

type utxoReference struct {
	hash  common.Blake2b256
	index uint32
}

// UtxoQueryBudget bounds the response data retained by one address lookup.
// Its methods are safe for concurrent reference-script hydration.
type UtxoQueryBudget struct {
	mu    sync.Mutex
	items int
	bytes int
	seen  map[utxoReference]struct{}
}

// AddPage checks page bytes and item count before the caller converts or
// retains any of its entries.
func (b *UtxoQueryBudget) AddPage(items, bytes int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if items < 0 || bytes < 0 || items > MaxUtxosPerQuery-b.items || bytes > MaxUtxoBytesPerQuery-b.bytes {
		return fmt.Errorf("UTxO query exceeds %d items or %d bytes", MaxUtxosPerQuery, MaxUtxoBytesPerQuery)
	}
	b.items += items
	b.bytes += bytes
	return nil
}

// AddBytes accounts for reference scripts resolved outside the UTxO page.
func (b *UtxoQueryBudget) AddBytes(bytes int) error {
	return b.AddPage(0, bytes)
}

// AddReference rejects a provider response that repeats an output reference.
func (b *UtxoQueryBudget) AddReference(utxo common.Utxo) error {
	if utxo.Id == nil {
		return errors.New("UTxO has no input reference")
	}
	ref := utxoReference{hash: utxo.Id.Id(), index: utxo.Id.Index()}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seen == nil {
		b.seen = make(map[utxoReference]struct{})
	}
	if _, exists := b.seen[ref]; exists {
		return fmt.Errorf("provider repeated UTxO reference %s", utxo.Id.String())
	}
	b.seen[ref] = struct{}{}
	return nil
}
