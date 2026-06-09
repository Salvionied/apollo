package apollo

import (
	"math"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

func TestBufferExUnits(t *testing.T) {
	tests := []struct {
		name   string
		value  int64
		factor float64
		want   int64
	}{
		{"zero", 0, 1.2, 0},
		{"negative clamped to zero", -100, 1.2, 0},
		{"normal value buffered", 1000, 1.2, 1200},
		{"near MaxInt64 saturates instead of wrapping negative", math.MaxInt64, 1.2, math.MaxInt64},
		{"just above overflow threshold saturates", math.MaxInt64/6*5 + 1, 1.2, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bufferExUnits(tt.value, tt.factor)
			if got != tt.want {
				t.Errorf("bufferExUnits(%d, %v) = %d, want %d", tt.value, tt.factor, got, tt.want)
			}
			if got < 0 {
				t.Errorf("bufferExUnits(%d, %v) = %d, must never be negative", tt.value, tt.factor, got)
			}
		})
	}
}

func TestMintNormalizesPolicyIdCase(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	// Mixed-case hex sorts differently as a string ('A' < 'a') than as bytes
	// (0xab < 0xac), which would misbind mint redeemer indexes. Mint must
	// normalize policy IDs to lowercase.
	lower := "ab012345678901234567890123456789012345678901234567890123"
	upper := "AC012345678901234567890123456789012345678901234567890123"
	redeemer := common.Datum{}
	a = a.Mint(NewUnit(lower, "746f6b656e", 1), &redeemer, nil)
	a = a.Mint(NewUnit(upper, "746f6b656e", 1), &redeemer, nil)

	sorted := a.sortedMintPolicyIds()
	if len(sorted) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(sorted))
	}
	if sorted[0] != "ab012345678901234567890123456789012345678901234567890123" ||
		sorted[1] != "ac012345678901234567890123456789012345678901234567890123" {
		t.Errorf("policies not normalized to lowercase byte order: %v", sorted)
	}
	for policy := range a.mintRedeemers {
		if policy != "ab012345678901234567890123456789012345678901234567890123" &&
			policy != "ac012345678901234567890123456789012345678901234567890123" {
			t.Errorf("mintRedeemers key not lowercased: %q", policy)
		}
	}
}

func TestMintDedupsPolicyIdAcrossCase(t *testing.T) {
	cc := setupFixedContext()
	a := New(cc)

	// The same policy supplied in two cases must collapse to one entry, not
	// produce an extra redeemer index.
	policy := "ab012345678901234567890123456789012345678901234567890123"
	a = a.Mint(NewUnit(policy, "61", 1), nil, nil)
	a = a.Mint(NewUnit("AB012345678901234567890123456789012345678901234567890123", "62", 1), nil, nil)

	sorted := a.sortedMintPolicyIds()
	if len(sorted) != 1 {
		t.Fatalf("expected 1 unique policy, got %d: %v", len(sorted), sorted)
	}
	if sorted[0] != policy {
		t.Errorf("expected %q, got %q", policy, sorted[0])
	}
}

func TestPaymentFromTxOutRejectsOversizedLovelace(t *testing.T) {
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		t.Fatalf("failed to parse address: %v", err)
	}
	out := NewBabbageOutput(addr, NewSimpleValue(uint64(math.MaxInt64)+1), nil, nil)
	if _, err := PaymentFromTxOut(&out); err == nil {
		t.Error("expected error for lovelace above MaxInt64, got nil")
	}
	out = NewBabbageOutput(addr, NewSimpleValue(uint64(math.MaxInt64)), nil, nil)
	if _, err := PaymentFromTxOut(&out); err != nil {
		t.Errorf("expected MaxInt64 lovelace to be accepted, got: %v", err)
	}
}

func TestNewPaymentFromValueRejectsOversizedLovelace(t *testing.T) {
	addr, err := common.NewAddress(validTestAddrBech32)
	if err != nil {
		t.Fatalf("failed to parse address: %v", err)
	}
	if _, err := NewPaymentFromValue(addr, NewSimpleValue(uint64(math.MaxInt64)+1)); err == nil {
		t.Error("expected error for lovelace above MaxInt64, got nil")
	}
}
