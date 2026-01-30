package cbor

import (
	"bytes"
	"errors"

	_cbor "github.com/fxamacker/cbor/v2"
)

// CBOR constants used in Cardano
const (
	// CborTagSet is used for sets in Cardano CDDL: set<a> = #6.258([* a])
	CborTagSet = 258

	// CBOR major type 5 (map) encoding constants
	// Maps with 0-23 entries use 0xa0 + length directly
	CborMapBase     = 0xa0 // Base byte for maps with length 0-23
	CborMap1ByteLen = 0xb8 // Map with 1-byte length following (24-255 entries)
	CborMap2ByteLen = 0xb9 // Map with 2-byte length following (256-65535 entries)
	CborMap4ByteLen = 0xba // Map with 4-byte length following (up to ~4 billion)
	CborMap8ByteLen = 0xbb // Map with 8-byte length following (up to ~18 quintillion)
)

// RawTag is an alias for the fxamacker/cbor RawTag type.
// It represents a CBOR tag with its number and raw content.
type RawTag = _cbor.RawTag

// RawMessage is an alias for raw CBOR bytes.
type RawMessage = _cbor.RawMessage

// decMode is the decode mode configured for Cardano CBOR
var decMode _cbor.DecMode

func init() {
	decOptions := _cbor.DecOptions{
		// Allow decoding unknown fields without error
		ExtraReturnErrors: _cbor.ExtraDecErrorNone,
		// Support deep nesting (Cardano blocks can be deeply nested)
		MaxNestedLevels: 256,
	}
	var err error
	decMode, err = decOptions.DecMode()
	if err != nil {
		panic("failed to create CBOR decode mode: " + err.Error())
	}
}

// Decode decodes CBOR data into the destination object.
func Decode(data []byte, dest any) error {
	return decMode.Unmarshal(data, dest)
}

// UnwrapSetTag checks if the data is wrapped in a CBOR tag 258 (Set).
// If it is, it returns the unwrapped content. Otherwise, it returns the original data.
// The second return value indicates whether the data was wrapped in a Set tag.
func UnwrapSetTag(data []byte) ([]byte, bool, error) {
	var tag RawTag
	if err := Decode(data, &tag); err == nil {
		if tag.Number == CborTagSet {
			return []byte(tag.Content), true, nil
		}
		// It's a tag, but not tag 258 - this might be an error for some use cases
		// but we return the original data to let the caller decide
	}
	// Not a tag, return original data
	return data, false, nil
}

// DecodeMapToRaw decodes a CBOR map into a map of uint64 keys to raw CBOR bytes.
// This allows preserving the original CBOR encoding for each field value,
// which is critical for data containing CBOR tags that may not decode correctly
// when re-encoded through Go interfaces.
func DecodeMapToRaw(data []byte) (map[uint64]RawMessage, error) {
	var result map[uint64]RawMessage
	if err := decMode.Unmarshal(data, &result); err != nil {
		return nil, errors.New("failed to decode map: " + err.Error())
	}
	return result, nil
}

// MapPair represents a key-value pair in a CBOR map where both key and value
// are preserved as raw CBOR bytes. This is useful for maps that may have
// complex keys (like CBOR tags) that can't be used as Go map keys.
type MapPair struct {
	KeyRaw   RawMessage
	ValueRaw RawMessage
}

// DecodeMapPairs decodes a CBOR map into a slice of key-value pairs,
// preserving raw bytes for both keys and values. This is essential for
// handling Plutus data maps where keys can be constructor-tagged values
// (cbor.Tag) which cannot be used as Go map keys.
func DecodeMapPairs(data []byte) ([]MapPair, error) {
	reader := bytes.NewReader(data)

	// Check the first byte to determine if it's a map
	firstByte, err := reader.ReadByte()
	if err != nil {
		return nil, errors.New("failed to read first byte: " + err.Error())
	}

	// Major type 5 is maps (0xa0-0xbf for definite, 0xbf for indefinite)
	majorType := firstByte >> 5
	if majorType != 5 {
		return nil, errors.New("data is not a CBOR map")
	}

	// Calculate map length
	var mapLen uint64
	additionalInfo := firstByte & 0x1f
	switch {
	case additionalInfo < 24:
		mapLen = uint64(additionalInfo)
	case additionalInfo == 24:
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		mapLen = uint64(b)
	case additionalInfo == 25:
		buf := make([]byte, 2)
		if _, err := reader.Read(buf); err != nil {
			return nil, err
		}
		mapLen = uint64(buf[0])<<8 | uint64(buf[1])
	case additionalInfo == 26:
		buf := make([]byte, 4)
		if _, err := reader.Read(buf); err != nil {
			return nil, err
		}
		mapLen = uint64(
			buf[0],
		)<<24 | uint64(
			buf[1],
		)<<16 | uint64(
			buf[2],
		)<<8 | uint64(
			buf[3],
		)
	case additionalInfo == 27:
		buf := make([]byte, 8)
		if _, err := reader.Read(buf); err != nil {
			return nil, err
		}
		mapLen = uint64(buf[0])<<56 |
			uint64(buf[1])<<48 |
			uint64(buf[2])<<40 |
			uint64(buf[3])<<32 |
			uint64(buf[4])<<24 |
			uint64(buf[5])<<16 |
			uint64(buf[6])<<8 |
			uint64(buf[7])
	case additionalInfo == 31:
		// Indefinite length - not commonly used in Cardano, but handle it
		return nil, errors.New("indefinite length maps not yet supported")
	default:
		return nil, errors.New("invalid map length encoding")
	}

	// Calculate header size from what we've already parsed (bytes consumed from reader)
	headerSize := len(data) - reader.Len()
	remaining := data[headerSize:]

	pairs := make([]MapPair, 0, mapLen)
	currentData := remaining

	for i := uint64(0); i < mapLen; i++ {
		// Decode key as RawMessage
		keyReader := bytes.NewReader(currentData)
		keyDec := decMode.NewDecoder(keyReader)
		var keyRaw RawMessage
		if err := keyDec.Decode(&keyRaw); err != nil {
			return nil, errors.New("failed to decode map key: " + err.Error())
		}
		keyBytesRead := keyDec.NumBytesRead()
		currentData = currentData[keyBytesRead:]

		// Decode value as RawMessage
		valueReader := bytes.NewReader(currentData)
		valueDec := decMode.NewDecoder(valueReader)
		var valueRaw RawMessage
		if err := valueDec.Decode(&valueRaw); err != nil {
			return nil, errors.New("failed to decode map value: " + err.Error())
		}
		valueBytesRead := valueDec.NumBytesRead()
		currentData = currentData[valueBytesRead:]

		pairs = append(pairs, MapPair{
			KeyRaw:   keyRaw,
			ValueRaw: valueRaw,
		})
	}

	return pairs, nil
}
