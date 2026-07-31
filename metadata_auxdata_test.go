package apollo

import (
	"encoding/hex"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// isCborNull reports whether b is a single CBOR null or undefined value, which
// is what gouroboros emits for auxiliary data with no stored encoding.
func isCborNull(b []byte) bool {
	return len(b) == 1 && (b[0] == 0xf6 || b[0] == 0xf7)
}

// splitTx decodes a serialized transaction into its four top-level CBOR
// elements: [body, witness_set, is_valid, auxiliary_data].
func splitTx(t *testing.T, txCbor []byte) []cbor.RawMessage {
	t.Helper()
	var elems []cbor.RawMessage
	if _, err := cbor.Decode(txCbor, &elems); err != nil {
		t.Fatalf("decode transaction: %v", err)
	}
	if len(elems) != 4 {
		t.Fatalf("expected a 4-element transaction, got %d", len(elems))
	}
	return elems
}

// buildWithMetadata completes a minimal transfer carrying metadata.
func buildWithMetadata(t *testing.T, metadata map[uint64]any) *Apollo {
	t.Helper()
	cc := setupFixedContext()
	addr := testAddress(t)
	addTestUtxo(cc, addr, 10_000_000, 0x01, 0)

	p, err := NewPayment(validTestAddrBech32, 2_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := New(cc).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		SetShelleyMetadata(metadata)

	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// TestMetadataIsSerializedIntoTheTransaction is the regression guard for
// auxiliary data being emitted as CBOR null. gouroboros serializes auxiliary
// data from TxMetadata.Cbor(), so a MetaMap built without SetCbor produces a
// transaction whose auxiliary_data is null while the body still declares
// auxiliary_data_hash -- the metadata is silently dropped and the transaction
// cannot validate.
func TestMetadataIsSerializedIntoTheTransaction(t *testing.T) {
	a := buildWithMetadata(t, map[uint64]any{
		674: map[string]any{"msg": "apollo"},
	})

	tx := a.GetTx()
	declared := tx.Body.TxAuxDataHash
	if declared == nil {
		t.Fatal("body has no auxiliary_data_hash despite metadata being set")
	}

	full, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	aux := splitTx(t, full)[3]

	if isCborNull(aux) {
		t.Fatalf(
			"auxiliary_data is CBOR null (%#x) but the body declares "+
				"auxiliary_data_hash %x",
			aux[0], declared[:],
		)
	}
	if len(aux) == 0 {
		t.Fatal("auxiliary_data is empty")
	}

	// The declared hash must be the hash of the bytes actually on the wire.
	if got := common.Blake2b256Hash(aux); got != *declared {
		t.Errorf(
			"blake2b256(wire auxiliary_data) = %x, body declares %x",
			got[:], declared[:],
		)
	}

	// And the wire bytes must be the metadata we asked for.
	if got, want := hex.EncodeToString(aux), hex.EncodeToString(
		tx.TxMetadata.Cbor(),
	); got != want {
		t.Errorf("wire auxiliary_data = %s, TxMetadata.Cbor() = %s", got, want)
	}
}

// TestMetadataRoundTripsThroughSerialization confirms the emitted auxiliary
// data decodes back to the metadata that was set, so the encoding is not
// merely non-null but correct.
func TestMetadataRoundTripsThroughSerialization(t *testing.T) {
	a := buildWithMetadata(t, map[uint64]any{
		674: map[string]any{"msg": "apollo"},
		1:   int64(42),
	})

	full, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	aux := splitTx(t, full)[3]

	// Shelley-era auxiliary data is a bare metadata map keyed by label.
	// Decoding generically asserts the wire format itself rather than relying
	// on any particular ledger-type decoder.
	var decoded map[uint64]cbor.RawMessage
	if _, err := cbor.Decode(aux, &decoded); err != nil {
		t.Fatalf("auxiliary_data does not decode as a metadata map: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("decoded %d metadata labels, want 2: %v", len(decoded), decoded)
	}
	for _, label := range []uint64{1, 674} {
		if _, ok := decoded[label]; !ok {
			t.Errorf("metadata label %d missing from auxiliary_data", label)
		}
	}

	// Label 1 was set to the integer 42.
	var got int64
	if _, err := cbor.Decode(decoded[1], &got); err != nil {
		t.Fatalf("decode label 1: %v", err)
	}
	if got != 42 {
		t.Errorf("metadata label 1 = %d, want 42", got)
	}
}

// TestMetadataSizeIsCountedInTheFee guards the fee consequence of the same
// bug: the size-estimation transactions also carried a MetaMap with no stored
// CBOR, so the fee was computed as though auxiliary data were a single byte.
func TestMetadataSizeIsCountedInTheFee(t *testing.T) {
	small := buildWithMetadata(t, map[uint64]any{1: int64(1)})

	// A metadata document large enough that its bytes dominate the delta.
	big := make(map[uint64]any, 8)
	for i := uint64(0); i < 8; i++ {
		big[i] = map[string]any{
			"msg": "0123456789012345678901234567890123456789012345678901234",
		}
	}
	large := buildWithMetadata(t, big)

	smallCbor, err := small.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	largeCbor, err := large.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}

	pp, err := small.Context.ProtocolParams()
	if err != nil {
		t.Fatal(err)
	}
	minFeeA := pp.MinFeeCoefficient

	smallFee, largeFee := small.GetTx().Body.TxFee, large.GetTx().Body.TxFee
	if len(largeCbor) <= len(smallCbor) {
		t.Fatalf(
			"larger metadata did not grow the transaction (%d -> %d)",
			len(smallCbor), len(largeCbor),
		)
	}
	if largeFee < smallFee {
		t.Fatalf("fee shrank with more metadata: %d -> %d", smallFee, largeFee)
	}
	if minFeeA <= 0 {
		t.Fatalf("MinFeeCoefficient is %d, want positive", minFeeA)
	}
	//nolint:gosec // length difference validated positive above
	sizeDelta := uint64(len(largeCbor) - len(smallCbor))
	feeDelta := largeFee - smallFee

	t.Logf(
		"size %d -> %d (delta %d), fee %d -> %d (delta %d), minFeeA %d",
		len(smallCbor), len(largeCbor), sizeDelta,
		smallFee, largeFee, feeDelta, minFeeA,
	)

	// The fee must cover the extra bytes at the per-byte coefficient.
	//nolint:gosec // minFeeA validated positive above
	if want := sizeDelta * uint64(minFeeA); feeDelta < want {
		t.Errorf(
			"fee grew by %d for %d extra bytes; needs at least %d "+
				"(metadata size not counted in the fee)",
			feeDelta, sizeDelta, want,
		)
	}
}

// TestEmptyMetadataStillSerializesConsistently covers SetShelleyMetadata with
// an empty map: whatever auxiliary_data is emitted, the declared hash must
// match it.
func TestEmptyMetadataStillSerializesConsistently(t *testing.T) {
	a := buildWithMetadata(t, map[uint64]any{})

	tx := a.GetTx()
	full, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	aux := splitTx(t, full)[3]

	if tx.Body.TxAuxDataHash == nil {
		// No hash declared: auxiliary_data must be absent/null too.
		if !isCborNull(aux) {
			t.Errorf(
				"no auxiliary_data_hash declared but auxiliary_data = %s",
				hex.EncodeToString(aux),
			)
		}
		return
	}
	if got := common.Blake2b256Hash(aux); got != *tx.Body.TxAuxDataHash {
		t.Errorf(
			"blake2b256(wire auxiliary_data) = %x, body declares %x",
			got[:], tx.Body.TxAuxDataHash[:],
		)
	}
}
