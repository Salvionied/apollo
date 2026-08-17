package apollo

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	plutigoData "github.com/blinklabs-io/plutigo/data"
)

// blake2bOfNothing is what common.Datum.Hash returns for a datum that was
// built in Go rather than decoded from CBOR: its stored CBOR is empty, so the
// method hashes the empty string. Any datum hash equal to this is a bug.
const blake2bOfNothing = "0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787" +
	"faab45cdf12fe3a8"

func decodeDatum(t *testing.T, cborHex string) common.Datum {
	t.Helper()
	raw, err := hex.DecodeString(cborHex)
	if err != nil {
		t.Fatal(err)
	}
	var datum common.Datum
	if err := datum.UnmarshalCBOR(raw); err != nil {
		t.Fatalf("decode datum %s: %v", cborHex, err)
	}
	return datum
}

// TestDatumHashUsesWireBytesForBuiltDatum covers the trap that
// common.Datum.Hash falls into for a datum that was never decoded.
func TestDatumHashUsesWireBytesForBuiltDatum(t *testing.T) {
	datum := common.Datum{Data: plutigoData.NewInteger(big.NewInt(99))}
	if len(datum.Cbor()) != 0 {
		t.Fatalf("expected no stored CBOR, got %x", datum.Cbor())
	}
	if got := datum.Hash().String(); got != blake2bOfNothing {
		t.Fatalf(
			"common.Datum.Hash on a built datum = %s, want %s",
			got,
			blake2bOfNothing,
		)
	}

	got, err := DatumHash(&datum)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := cbor.Encode(&datum)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(wire) != "1863" {
		t.Fatalf("unexpected wire bytes %x", wire)
	}
	want := common.Blake2b256Hash(wire)
	if got != want {
		t.Fatalf("DatumHash = %x, want %x", got.Bytes(), want.Bytes())
	}
	if got.String() == blake2bOfNothing {
		t.Fatal("DatumHash returned the hash of the empty string")
	}
	// DatumWireCbor pins the wire bytes, so the obvious method now agrees.
	if datum.Hash() != want {
		t.Fatalf(
			"common.Datum.Hash = %x after pinning, want %x",
			datum.Hash().Bytes(),
			want.Bytes(),
		)
	}
}

func TestDatumWireCborKeepsCanonicalStoredBytes(t *testing.T) {
	datum := decodeDatum(t, "d8799f4161ff")
	got, err := DatumWireCbor(&datum)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != "d8799f4161ff" {
		t.Fatalf("DatumWireCbor = %x, want d8799f4161ff", got)
	}
	hash, err := DatumHash(&datum)
	if err != nil {
		t.Fatal(err)
	}
	if hash != datum.Hash() {
		t.Fatalf(
			"DatumHash %x disagrees with common.Datum.Hash %x",
			hash.Bytes(),
			datum.Hash().Bytes(),
		)
	}
}

// TestDatumWireCborPreservesNonCanonicalStoredBytes covers datum encodings
// that must remain byte-for-byte stable when they are re-encoded.
func TestDatumWireCborPreservesNonCanonicalStoredBytes(t *testing.T) {
	tests := []struct {
		name    string
		cborHex string
	}{
		{
			name: "definite byte string above the 64 byte chunk limit",
			cborHex: "d8799f5864" +
				strings.Repeat("ab", 100) + "ff",
		},
		{name: "non-minimal integer", cborHex: "1801"},
		{name: "two byte integer holding a small value", cborHex: "190001"},
		{name: "bignum tag around a small value", cborHex: "c24101"},
		{name: "indefinite length byte string", cborHex: "5f4161ff"},
		{name: "indefinite length constr fields", cborHex: "d8669f18809f01ffff"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			datum := decodeDatum(t, test.cborHex)
			wire, err := cbor.Encode(&datum)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(wire) != test.cborHex {
				t.Fatalf(
					"wire bytes = %x, want %s",
					wire,
					test.cborHex,
				)
			}
			storedHash := common.Blake2b256Hash(datum.Cbor())
			wireHash := common.Blake2b256Hash(wire)
			if storedHash != wireHash {
				t.Fatalf("stored and wire hashes differ: %x != %x", storedHash.Bytes(), wireHash.Bytes())
			}

			if got, err := DatumWireCbor(&datum); err != nil {
				t.Fatal(err)
			} else if hex.EncodeToString(got) != test.cborHex {
				t.Fatalf("DatumWireCbor = %x, want %s", got, test.cborHex)
			}

			if got, err := DatumHash(&datum); err != nil {
				t.Fatal(err)
			} else if got != storedHash {
				t.Fatalf("DatumHash = %x, want %x", got.Bytes(), storedHash.Bytes())
			}
		})
	}
}

func TestAddDatumPreservesNonCanonicalStoredBytes(t *testing.T) {
	datum := decodeDatum(t, "d8799f5864"+strings.Repeat("ab", 100)+"ff")
	cc := setupFixedContext()
	a := New(cc).AddDatum(&datum)
	if len(a.datums) != 1 {
		t.Fatalf("expected the datum to be preserved, got %d", len(a.datums))
	}
}

// TestPayToContractWithDatumHashPreservesDatumIdentity requires that Apollo
// either declares the hash the caller's datum bytes actually have, or refuses
// the payment. Silently declaring some other hash creates an output at a script
// address whose datum the counterparty cannot match, locking the funds.
func TestPayToContractWithDatumHashPreservesDatumIdentity(t *testing.T) {
	datum := decodeDatum(t, "d8799f5864"+strings.Repeat("ab", 100)+"ff")
	want := common.Blake2b256Hash(datum.Cbor())

	cc := setupFixedContext()
	addr := testAddress(t)
	a, err := New(cc).PayToContractWithDatumHash(addr, &datum, 2_000_000)
	if err != nil {
		// Refusing a datum whose bytes cannot be preserved is correct.
		return
	}
	if len(a.payments) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(a.payments))
	}
	txOut, err := a.payments[0].ToTxOut()
	if err != nil {
		t.Fatal(err)
	}
	got := txOut.DatumHash()
	if got == nil {
		t.Fatal("output carries no datum hash")
	}
	if *got != want {
		t.Fatalf(
			"output declares datum hash %x for a datum whose bytes hash to %x",
			got.Bytes(),
			want.Bytes(),
		)
	}
}

func TestPayToContractWithDatumHashPreservesNonCanonicalDatum(t *testing.T) {
	datum := decodeDatum(t, "d8799f5864"+strings.Repeat("ab", 100)+"ff")
	cc := setupFixedContext()
	addr := testAddress(t)
	a, err := New(cc).PayToContractWithDatumHash(addr, &datum, 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.datums) != 1 || len(a.payments) != 1 {
		t.Fatalf("expected one datum and one payment, got %d and %d", len(a.datums), len(a.payments))
	}
	txOut, err := a.payments[0].ToTxOut()
	if err != nil {
		t.Fatal(err)
	}
	if got := txOut.DatumHash(); got == nil || *got != common.Blake2b256Hash(datum.Cbor()) {
		t.Fatalf("datum hash = %v, want %x", got, common.Blake2b256Hash(datum.Cbor()).Bytes())
	}
}

func TestDatumWireCborRejectsNil(t *testing.T) {
	if _, err := DatumWireCbor(nil); err == nil {
		t.Fatal("expected an error for a nil datum")
	}
	if _, err := DatumHash(nil); err == nil {
		t.Fatal("expected an error for a nil datum")
	}
}
