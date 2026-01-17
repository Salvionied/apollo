package cbor

import (
	"encoding/hex"
	"testing"
)

func TestUnwrapSetTag(t *testing.T) {
	tests := []struct {
		name       string
		inputHex   string
		wantTagged bool
		wantErr    bool
	}{
		{
			name:       "tag 258 wrapped array",
			inputHex:   "d9010283010203", // #6.258([1, 2, 3])
			wantTagged: true,
			wantErr:    false,
		},
		{
			name:       "plain array",
			inputHex:   "83010203", // [1, 2, 3]
			wantTagged: false,
			wantErr:    false,
		},
		{
			name:       "empty tag 258 array",
			inputHex:   "d9010280", // #6.258([])
			wantTagged: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := hex.DecodeString(tt.inputHex)
			if err != nil {
				t.Fatalf("failed to decode hex: %v", err)
			}

			_, gotTagged, gotErr := UnwrapSetTag(input)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("UnwrapSetTag() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if gotTagged != tt.wantTagged {
				t.Errorf("UnwrapSetTag() tagged = %v, want %v", gotTagged, tt.wantTagged)
			}
		})
	}
}

func TestDecodeRawTag(t *testing.T) {
	// Test decoding a tag 258 wrapped array: #6.258([1, 2, 3])
	// d90102 = tag 258, 83 = array(3), 01 02 03 = 1, 2, 3
	input, _ := hex.DecodeString("d9010283010203")
	var tag RawTag
	err := Decode(input, &tag)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if tag.Number != CborTagSet {
		t.Errorf("tag.Number = %d, want %d", tag.Number, CborTagSet)
	}

	// Decode the content as an array
	var arr []int
	err = Decode(tag.Content, &arr)
	if err != nil {
		t.Fatalf("Decode content error = %v", err)
	}
	if len(arr) != 3 {
		t.Errorf("len(arr) = %d, want 3", len(arr))
	}
}

func TestDecodeMapPairs(t *testing.T) {
	tests := []struct {
		name      string
		inputHex  string
		wantLen   int
		wantErr   bool
		checkKeys []int // expected integer keys (if applicable)
	}{
		{
			name:      "simple map {1: 2}",
			inputHex:  "a10102", // {1: 2}
			wantLen:   1,
			wantErr:   false,
			checkKeys: []int{1},
		},
		{
			name:      "map with multiple entries {1: 2, 3: 4}",
			inputHex:  "a201020304", // {1: 2, 3: 4}
			wantLen:   2,
			wantErr:   false,
			checkKeys: []int{1, 3},
		},
		{
			name:      "empty map",
			inputHex:  "a0", // {}
			wantLen:   0,
			wantErr:   false,
			checkKeys: nil,
		},
		{
			name:     "not a map (array)",
			inputHex: "83010203", // [1, 2, 3]
			wantLen:  0,
			wantErr:  true,
		},
		{
			name: "map with 1-byte length (24 entries)",
			// b818 = map with 1-byte length (24), followed by 24 key-value pairs
			// Keys 0-23 (each 1 byte), values all 0 (each 1 byte)
			inputHex: "b818" +
				"0000010002000300040005000600070008000900" + // keys 0-9, values 0
				"0a000b000c000d000e000f001000110012001300" + // keys 10-19, values 0
				"1400150016001700", // keys 20-23, values 0
			wantLen:   24,
			wantErr:   false,
			checkKeys: nil, // too many to check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := hex.DecodeString(tt.inputHex)
			if err != nil {
				t.Fatalf("failed to decode hex: %v", err)
			}

			pairs, gotErr := DecodeMapPairs(input)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("DecodeMapPairs() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}
			if gotErr != nil {
				return
			}
			if len(pairs) != tt.wantLen {
				t.Errorf("DecodeMapPairs() len = %d, want %d", len(pairs), tt.wantLen)
			}

			// Verify keys can be decoded
			for i, expectedKey := range tt.checkKeys {
				if i >= len(pairs) {
					break
				}
				var key int
				if err := Decode(pairs[i].KeyRaw, &key); err != nil {
					t.Errorf("Failed to decode key %d: %v", i, err)
				} else if key != expectedKey {
					t.Errorf("Key %d = %d, want %d", i, key, expectedKey)
				}
			}
		})
	}
}

func TestDecodeMapToRaw(t *testing.T) {
	// Test map with uint64 keys: {0: "hello", 1: 42}
	// a2 = map(2), 00 = 0, 65 68656c6c6f = "hello", 01 = 1, 182a = 42
	input, _ := hex.DecodeString("a2006568656c6c6f01182a")
	result, err := DecodeMapToRaw(input)
	if err != nil {
		t.Fatalf("DecodeMapToRaw() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}

	// Check key 0 contains "hello"
	if raw, ok := result[0]; ok {
		var str string
		if err := Decode(raw, &str); err != nil {
			t.Errorf("Failed to decode value for key 0: %v", err)
		} else if str != "hello" {
			t.Errorf("Value for key 0 = %q, want %q", str, "hello")
		}
	} else {
		t.Error("Missing key 0")
	}

	// Check key 1 contains 42
	if raw, ok := result[1]; ok {
		var num int
		if err := Decode(raw, &num); err != nil {
			t.Errorf("Failed to decode value for key 1: %v", err)
		} else if num != 42 {
			t.Errorf("Value for key 1 = %d, want %d", num, 42)
		}
	} else {
		t.Error("Missing key 1")
	}
}
