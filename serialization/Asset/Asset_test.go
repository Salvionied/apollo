package Asset_test

import (
	"bytes"
	"testing"

	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
)

var assetNamet1 = AssetName.NewAssetNameFromString("test")
var assetNamet2 = AssetName.NewAssetNameFromString("test2")

func TestEquality(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	val2 := Asset.Asset[int64]{
		assetNamet1: 99,
	}
	if val0.Equal(val2) {
		t.Errorf("Amounts should not be equal")
	}
	val3 := Asset.Asset[int64]{
		assetNamet1: 100,
		assetNamet2: 100,
	}
	if val0.Equal(val3) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestClone(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	val1 := val0.Clone()
	if !val0.Equal(val1) {
		t.Errorf("Amounts should be equal")
	}
	if &val0 == &val1 {
		t.Errorf("Amounts should not be equal")
	}
}

func TestLess(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 50,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val0.Less(val1) {
		t.Errorf("Amounts should be equal")
	}
	if val1.Less(val0) {
		t.Errorf("Amounts should not be equal")
	}
	val2 := Asset.Asset[int64]{}
	if !val2.Less(val1) {
		t.Errorf("Amounts should be equal")
	}
	if val1.Less(val2) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestGreater(t *testing.T) {
	val0 := Asset.Asset[int64]{
		assetNamet1: 50,
	}
	val1 := Asset.Asset[int64]{
		assetNamet1: 100,
	}
	if !val1.Greater(val0) {
		t.Errorf("Amounts should be equal")
	}
	if val0.Greater(val1) {
		t.Errorf("Amounts should not be equal")
	}
	val2 := Asset.Asset[int64]{}
	if !val1.Greater(val2) {
		t.Errorf("Amounts should be equal")
	}
	if !val2.Greater(val1) {
		t.Errorf("Amounts should not be equal")
	}
}

func TestSub(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := Asset.Asset[int64]{}
	val1[assetNamet1] = 50
	val2 := val0.Sub(val1)
	if val2[assetNamet1] != 50 {
		t.Errorf("Amounts should be equal")
	}
	val3 := Asset.Asset[int64]{}
	val3 = val3.Sub(val1)
	if val3[assetNamet1] != -50 {
		t.Errorf("Amounts should be equal")
	}
}

func TestAdd(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := Asset.Asset[int64]{}
	val1[assetNamet1] = 50
	val2 := val0.Add(val1)
	if val2[assetNamet1] != 150 {
		t.Errorf("Amounts should be equal")
	}
	val3 := Asset.Asset[int64]{}
	val3 = val3.Add(val1)
	if val3[assetNamet1] != 50 {
		t.Errorf("Amounts should be equal")
	}
}

func TestInverted(t *testing.T) {
	val0 := Asset.Asset[int64]{}
	val0[assetNamet1] = 100
	val1 := val0.Inverted()
	if val1[assetNamet1] != -100 {
		t.Errorf("Amounts should be equal")
	}
}

func TestDeterministicCBOREncoding(t *testing.T) {
	asset1 := AssetName.NewAssetNameFromString("alpha")
	asset2 := AssetName.NewAssetNameFromString("beta")
	asset3 := AssetName.NewAssetNameFromString("gamma")

	a := Asset.Asset[int64]{
		asset3: 300,
		asset1: 100,
		asset2: 200,
	}

	var firstEncoding []byte
	for i := 0; i < 100; i++ {
		encoded, err := a.MarshalCBOR()
		if err != nil {
			t.Fatalf("Failed to marshal Asset: %v", err)
		}
		if firstEncoding == nil {
			firstEncoding = encoded
		} else if !bytes.Equal(encoded, firstEncoding) {
			t.Errorf(
				"Non-deterministic encoding on iteration %d: got %x, want %x",
				i, encoded, firstEncoding,
			)
		}
	}
}

func TestDeterministicCBOREncodingSortOrder(t *testing.T) {
	assetA := AssetName.NewAssetNameFromString("a")
	assetB := AssetName.NewAssetNameFromString("b")
	assetC := AssetName.NewAssetNameFromString("c")

	a1 := Asset.Asset[int64]{
		assetC: 3,
		assetA: 1,
		assetB: 2,
	}

	a2 := Asset.Asset[int64]{
		assetA: 1,
		assetB: 2,
		assetC: 3,
	}

	enc1, err := a1.MarshalCBOR()
	if err != nil {
		t.Fatalf("Failed to marshal a1: %v", err)
	}
	enc2, err := a2.MarshalCBOR()
	if err != nil {
		t.Fatalf("Failed to marshal a2: %v", err)
	}

	if !bytes.Equal(enc1, enc2) {
		t.Errorf(
			"Same assets with different map order produced different encodings",
		)
		t.Errorf("a1 encoding: %x", enc1)
		t.Errorf("a2 encoding: %x", enc2)
	}
}

// TestCanonicalCBOROrdering validates that asset names are sorted according to
// RFC 7049 Section 3.9 canonical CBOR encoding rules, as required by CIP-0021:
//  1. Shorter keys sort before longer keys
//  2. Keys of equal length are sorted lexicographically
func TestCanonicalCBOROrdering(t *testing.T) {
	// Helper to extract key order from encoded CBOR
	extractKeyOrder := func(encoded []byte) []string {
		var keys []string
		pos := 1 // skip map header
		for pos < len(encoded) {
			// Read key length from CBOR byte string header
			header := encoded[pos]
			if header < 0x40 || header > 0x5b {
				break // not a byte string
			}
			var keyLen int
			if header <= 0x57 {
				keyLen = int(header & 0x1f)
				pos++
			} else if header == 0x58 {
				keyLen = int(encoded[pos+1])
				pos += 2
			} else {
				break // longer length encoding not expected in tests
			}
			key := string(encoded[pos : pos+keyLen])
			keys = append(keys, key)
			pos += keyLen
			// Skip value (assume small integers for this test)
			pos++
		}
		return keys
	}

	tests := []struct {
		name     string
		assets   Asset.Asset[int64]
		expected []string
	}{
		{
			name: "shorter keys before longer keys",
			assets: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("AA"):  2,
				AssetName.NewAssetNameFromString("B"):   1,
				AssetName.NewAssetNameFromString("CCC"): 3,
			},
			expected: []string{"B", "AA", "CCC"},
		},
		{
			name: "same length sorted lexicographically",
			assets: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("cc"): 3,
				AssetName.NewAssetNameFromString("aa"): 1,
				AssetName.NewAssetNameFromString("bb"): 2,
			},
			expected: []string{"aa", "bb", "cc"},
		},
		{
			name: "mixed lengths with lexicographic tiebreaker",
			assets: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("ZZ"):  4,
				AssetName.NewAssetNameFromString("A"):   1,
				AssetName.NewAssetNameFromString("BB"):  3,
				AssetName.NewAssetNameFromString("Z"):   2,
				AssetName.NewAssetNameFromString("AAA"): 5,
				AssetName.NewAssetNameFromString("ZZZ"): 6,
			},
			// Length 1: A, Z (lex order)
			// Length 2: BB, ZZ (lex order)
			// Length 3: AAA, ZZZ (lex order)
			expected: []string{"A", "Z", "BB", "ZZ", "AAA", "ZZZ"},
		},
		{
			name: "CIP-0021 example: SpaceBud tokens",
			assets: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("SpaceBud2934"): 2,
				AssetName.NewAssetNameFromString("SpaceBud312"):  1,
				AssetName.NewAssetNameFromString("SpaceBud3843"): 3,
			},
			// SpaceBud312 (11 chars) before SpaceBud2934/SpaceBud3843 (12 chars)
			// SpaceBud2934 before SpaceBud3843 (lexicographic, '2' < '3')
			expected: []string{"SpaceBud312", "SpaceBud2934", "SpaceBud3843"},
		},
		{
			name: "empty asset name sorts first",
			assets: Asset.Asset[int64]{
				AssetName.NewAssetNameFromString("token"): 2,
				AssetName.NewAssetNameFromString(""):      1,
			},
			expected: []string{"", "token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := tt.assets.MarshalCBOR()
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			actual := extractKeyOrder(encoded)

			if len(actual) != len(tt.expected) {
				t.Fatalf(
					"Key count mismatch: got %d, want %d",
					len(actual),
					len(tt.expected),
				)
			}

			for i, key := range actual {
				if key != tt.expected[i] {
					t.Errorf(
						"Key order mismatch at position %d: got %q, want %q",
						i,
						key,
						tt.expected[i],
					)
					t.Errorf("Full actual order: %v", actual)
					t.Errorf("Full expected order: %v", tt.expected)
					break
				}
			}
		})
	}
}
