package backendutil

import (
	"errors"
	"fmt"
	"sync"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

type utxoReference struct {
	hash  common.Blake2b256
	index uint32
}

// UtxoReferenceSet rejects repeated output references across provider pages.
// Its methods are safe for concurrent callers.
type UtxoReferenceSet struct {
	mu   sync.Mutex
	seen map[utxoReference]struct{}
}

// Add rejects a provider response that repeats an output reference.
func (s *UtxoReferenceSet) Add(utxo common.Utxo) error {
	if utxo.Id == nil {
		return errors.New("UTxO has no input reference")
	}
	ref := utxoReference{hash: utxo.Id.Id(), index: utxo.Id.Index()}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen == nil {
		s.seen = make(map[utxoReference]struct{})
	}
	if _, exists := s.seen[ref]; exists {
		return fmt.Errorf("provider repeated UTxO reference %s", utxo.Id.String())
	}
	s.seen[ref] = struct{}{}
	return nil
}
