package plutusencoder

import (
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

func TestUnmarshalNonPointer(t *testing.T) {
	var d SimpleDatum
	err := UnmarshalPlutus(data.NewInteger(big.NewInt(0)), d)
	if err == nil {
		t.Error("expected error for non-pointer")
	}
}

func TestUnmarshalNilPointer(t *testing.T) {
	err := UnmarshalPlutus(data.NewInteger(big.NewInt(0)), (*SimpleDatum)(nil))
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}
