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

// TestDatumWireCborRejectsNonCanonicalStoredBytes covers datum encodings that
// plutigo cannot reproduce. Hashing the stored bytes while the witness set
// carries the re-encoded bytes is what silently locks funds, so Apollo refuses
// the datum instead.
func TestDatumWireCborRejectsNonCanonicalStoredBytes(t *testing.T) {
	tests := []struct {
		name     string
		cborHex  string
		wantWire string
	}{
		{
			name: "definite byte string above the 64 byte chunk limit",
			cborHex: "d8799f5864" +
				strings.Repeat("ab", 100) + "ff",
			wantWire: "d8799f5f5840" + strings.Repeat("ab", 64) +
				"5824" + strings.Repeat("ab", 36) + "ffff",
		},
		{
			name:     "non-minimal integer",
			cborHex:  "1801",
			wantWire: "01",
		},
		{
			name:     "two byte integer holding a small value",
			cborHex:  "190001",
			wantWire: "01",
		},
		{
			name:     "bignum tag around a small value",
			cborHex:  "c24101",
			wantWire: "01",
		},
		{
			name:     "indefinite length byte string",
			cborHex:  "5f4161ff",
			wantWire: "4161",
		},
		{
			name:     "indefinite length constr fields",
			cborHex:  "d8669f18809f01ffff",
			wantWire: "d8668218809f01ff",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			datum := decodeDatum(t, test.cborHex)
			wire, err := cbor.Encode(&datum)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(wire) != test.wantWire {
				t.Fatalf(
					"wire bytes = %x, want %s",
					wire,
					test.wantWire,
				)
			}
			storedHash := common.Blake2b256Hash(datum.Cbor())
			wireHash := common.Blake2b256Hash(wire)
			if storedHash == wireHash {
				t.Fatalf(
					"stored and wire bytes hash alike (%x); not a divergent case",
					storedHash.Bytes(),
				)
			}

			if _, err := DatumWireCbor(&datum); err == nil {
				t.Fatal("expected DatumWireCbor to reject the datum")
			} else {
				if !strings.Contains(err.Error(), storedHash.String()) {
					t.Errorf(
						"error does not name the original hash %s: %v",
						storedHash.String(),
						err,
					)
				}
				if !strings.Contains(err.Error(), wireHash.String()) {
					t.Errorf(
						"error does not name the wire hash %s: %v",
						wireHash.String(),
						err,
					)
				}
			}

			if _, err := DatumHash(&datum); err == nil {
				t.Fatal("expected DatumHash to reject the datum")
			}
		})
	}
}

func TestAddDatumRejectsNonCanonicalStoredBytes(t *testing.T) {
	datum := decodeDatum(t, "d8799f5864"+strings.Repeat("ab", 100)+"ff")
	cc := setupFixedContext()
	a := New(cc).AddDatum(&datum)
	if len(a.datums) != 0 {
		t.Fatalf("expected the datum to be rejected, got %d", len(a.datums))
	}
	if _, err := a.Complete(); err == nil {
		t.Fatal("expected Complete to report the datum error")
	} else if !strings.Contains(err.Error(), "AddDatum") {
		t.Fatalf("unexpected error: %v", err)
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

func TestPayToContractWithDatumHashRejectsNonCanonicalDatum(t *testing.T) {
	datum := decodeDatum(t, "d8799f5864"+strings.Repeat("ab", 100)+"ff")
	cc := setupFixedContext()
	addr := testAddress(t)
	_, err := New(cc).PayToContractWithDatumHash(addr, &datum, 2_000_000)
	if err == nil {
		t.Fatal("expected PayToContractWithDatumHash to reject the datum")
	}
	if !strings.Contains(err.Error(), "canonical Plutus form") {
		t.Fatalf("unexpected error: %v", err)
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
