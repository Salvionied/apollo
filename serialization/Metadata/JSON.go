package Metadata

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const metadataStringMaxBytes = 64

var (
	maxMetadataInteger = new(big.Int).SetUint64(^uint64(0))
	minMetadataInteger = new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 64))
	maxInt64Metadata   = big.NewInt(int64(^uint64(0) >> 1))
	minInt64Metadata   = big.NewInt(-int64(^uint64(0)>>1) - 1)
)

// JSONSchema selects the Cardano transaction metadata JSON mapping.
type JSONSchema int

const (
	// JSONNoSchema is the cardano-cli no-schema JSON mapping used by
	// --metadata-json-file. JSON objects become metadata maps, strings
	// beginning with a valid lowercase 0x hex prefix become bytes.
	JSONNoSchema JSONSchema = iota

	// JSONDetailedSchema is the tagged mapping with int, bytes, string,
	// list and map fields.
	JSONDetailedSchema
)

// NegativeInteger represents a metadata integer below math.MinInt64.
// Its value is the CBOR major type 1 argument, so it encodes as -1-n.
type NegativeInteger uint64

// MarshalCBOR encodes a large negative metadata integer.
func (n NegativeInteger) MarshalCBOR() ([]byte, error) {
	return appendCBORType(nil, 1, uint64(n)), nil
}

// String returns the decimal representation of the negative integer.
func (n NegativeInteger) String() string {
	value := new(big.Int).SetUint64(uint64(n))
	value.Add(value, big.NewInt(1))
	value.Neg(value)
	return value.String()
}

// MetadatumMapEntry is one key/value pair in a Cardano metadata map.
type MetadatumMapEntry struct {
	Key   any
	Value any
}

// MetadatumMap preserves arbitrary metadata map keys, including bytes.
type MetadatumMap []MetadatumMapEntry

// MarshalCBOR encodes a Cardano metadata map with deterministic key ordering.
func (m MetadatumMap) MarshalCBOR() ([]byte, error) {
	enc, err := cbor.EncOptions{Sort: cbor.SortLengthFirst}.EncMode()
	if err != nil {
		return nil, err
	}
	type encodedPair struct {
		key   []byte
		value []byte
	}
	encoded := make([]encodedPair, 0, len(m))
	for _, entry := range m {
		keyBytes, err := enc.Marshal(entry.Key)
		if err != nil {
			return nil, fmt.Errorf("metadata map key marshal failed: %w", err)
		}
		valueBytes, err := enc.Marshal(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("metadata map value marshal failed: %w", err)
		}
		encoded = append(encoded, encodedPair{key: keyBytes, value: valueBytes})
	}
	sort.Slice(encoded, func(i, j int) bool {
		if len(encoded[i].key) != len(encoded[j].key) {
			return len(encoded[i].key) < len(encoded[j].key)
		}
		return bytes.Compare(encoded[i].key, encoded[j].key) < 0
	})
	out := appendCBORType(nil, 5, uint64(len(encoded)))
	for _, pair := range encoded {
		out = append(out, pair.key...)
		out = append(out, pair.value...)
	}
	return out, nil
}

// ShelleyMaryMetadataFromJSON parses cardano-cli-style metadata JSON using
// the no-schema mapping and returns Shelley Mary metadata.
func ShelleyMaryMetadataFromJSON(jsonData []byte) (ShelleyMaryMetadata, error) {
	return ShelleyMaryMetadataFromJSONWithSchema(jsonData, JSONNoSchema)
}

// ShelleyMaryMetadataFromJSONWithSchema parses metadata JSON using the
// requested Cardano metadata JSON mapping.
func ShelleyMaryMetadataFromJSONWithSchema(
	jsonData []byte,
	schema JSONSchema,
) (ShelleyMaryMetadata, error) {
	var raw any
	dec := json.NewDecoder(bytes.NewReader(jsonData))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return ShelleyMaryMetadata{}, fmt.Errorf("decode metadata JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return ShelleyMaryMetadata{}, errors.New(
				"decode metadata JSON: multiple JSON values",
			)
		}
		return ShelleyMaryMetadata{}, fmt.Errorf("decode metadata JSON: %w", err)
	}
	metadata, err := parseTopLevelMetadata(raw, schema)
	if err != nil {
		return ShelleyMaryMetadata{}, err
	}
	return ShelleyMaryMetadata{Metadata: metadata}, nil
}

func parseTopLevelMetadata(raw any, schema JSONSchema) (Metadata, error) {
	object, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("metadata JSON top level must be an object")
	}
	metadata := make(Metadata, len(object))
	for key, value := range object {
		label, err := parseTopLevelMetadataLabel(key)
		if err != nil {
			return nil, err
		}
		path := fmt.Sprintf("metadata[%d]", label)
		var parsed any
		switch schema {
		case JSONNoSchema:
			parsed, err = parseNoSchemaMetadataValue(value, path)
		case JSONDetailedSchema:
			parsed, err = parseDetailedMetadataValue(value, path)
		default:
			err = fmt.Errorf("unsupported metadata JSON schema %d", schema)
		}
		if err != nil {
			return nil, err
		}
		metadata[label] = parsed
	}
	return metadata, nil
}

func parseNoSchemaMetadataValue(value any, path string) (any, error) {
	switch v := value.(type) {
	case nil:
		return nil, fmt.Errorf("%s: null metadata values are not allowed", path)
	case bool:
		return nil, fmt.Errorf("%s: boolean metadata values are not allowed", path)
	case json.Number:
		return parseJSONMetadataNumber(v, path)
	case string:
		return parseNoSchemaString(v, path)
	case []any:
		items := make([]any, 0, len(v))
		for idx, item := range v {
			parsed, err := parseNoSchemaMetadataValue(
				item,
				fmt.Sprintf("%s[%d]", path, idx),
			)
			if err != nil {
				return nil, err
			}
			items = append(items, parsed)
		}
		return items, nil
	case map[string]any:
		entries := make(MetadatumMap, 0, len(v))
		for key, item := range v {
			parsedKey, err := parseNoSchemaMapKey(
				key,
				fmt.Sprintf("%s key %q", path, key),
			)
			if err != nil {
				return nil, err
			}
			parsedValue, err := parseNoSchemaMetadataValue(
				item,
				fmt.Sprintf("%s.%q", path, key),
			)
			if err != nil {
				return nil, err
			}
			entries = append(
				entries,
				MetadatumMapEntry{Key: parsedKey, Value: parsedValue},
			)
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("%s: unsupported JSON metadata value", path)
	}
}

func parseDetailedMetadataValue(value any, path string) (any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"%s: detailed metadata value must be an object",
			path,
		)
	}
	if len(object) != 1 {
		return nil, fmt.Errorf(
			"%s: detailed metadata value must have exactly one field",
			path,
		)
	}
	for key, typedValue := range object {
		switch key {
		case "int":
			number, ok := typedValue.(json.Number)
			if !ok {
				return nil, fmt.Errorf("%s.int: expected JSON number", path)
			}
			return parseJSONMetadataNumber(number, path+".int")
		case "bytes":
			text, ok := typedValue.(string)
			if !ok {
				return nil, fmt.Errorf("%s.bytes: expected hex string", path)
			}
			return parseDetailedBytes(text, path+".bytes")
		case "string":
			text, ok := typedValue.(string)
			if !ok {
				return nil, fmt.Errorf("%s.string: expected string", path)
			}
			if err := validateMetadataText(text, path+".string"); err != nil {
				return nil, err
			}
			return text, nil
		case "list":
			list, ok := typedValue.([]any)
			if !ok {
				return nil, fmt.Errorf("%s.list: expected array", path)
			}
			items := make([]any, 0, len(list))
			for idx, item := range list {
				parsed, err := parseDetailedMetadataValue(
					item,
					fmt.Sprintf("%s.list[%d]", path, idx),
				)
				if err != nil {
					return nil, err
				}
				items = append(items, parsed)
			}
			return items, nil
		case "map":
			list, ok := typedValue.([]any)
			if !ok {
				return nil, fmt.Errorf("%s.map: expected array", path)
			}
			entries := make(MetadatumMap, 0, len(list))
			for idx, item := range list {
				entry, err := parseDetailedMapEntry(
					item,
					fmt.Sprintf("%s.map[%d]", path, idx),
				)
				if err != nil {
					return nil, err
				}
				entries = append(entries, entry)
			}
			return entries, nil
		default:
			return nil, fmt.Errorf(
				"%s: unknown detailed metadata field %q",
				path,
				key,
			)
		}
	}
	return nil, fmt.Errorf("%s: empty detailed metadata object", path)
}

func parseDetailedMapEntry(value any, path string) (MetadatumMapEntry, error) {
	object, ok := value.(map[string]any)
	if !ok || len(object) != 2 {
		return MetadatumMapEntry{}, fmt.Errorf(
			"%s: map entry must be an object with k and v fields",
			path,
		)
	}
	keyValue, hasKey := object["k"]
	valueValue, hasValue := object["v"]
	if !hasKey || !hasValue {
		return MetadatumMapEntry{}, fmt.Errorf(
			"%s: map entry must contain k and v fields",
			path,
		)
	}
	key, err := parseDetailedMetadataValue(keyValue, path+".k")
	if err != nil {
		return MetadatumMapEntry{}, err
	}
	parsedValue, err := parseDetailedMetadataValue(valueValue, path+".v")
	if err != nil {
		return MetadatumMapEntry{}, err
	}
	return MetadatumMapEntry{Key: key, Value: parsedValue}, nil
}

func parseTopLevelMetadataLabel(label string) (int, error) {
	parsed, ok := parseUnsignedIntegerText(label)
	if !ok {
		return 0, fmt.Errorf(
			"metadata top-level key %q must be an unsigned integer without leading zeroes",
			label,
		)
	}
	if parsed.Cmp(maxMetadataInteger) > 0 {
		return 0, fmt.Errorf(
			"metadata top-level key %q exceeds uint64 range",
			label,
		)
	}
	maxInt := uint64(^uint(0) >> 1)
	if parsed.Uint64() > maxInt {
		return 0, fmt.Errorf(
			"metadata top-level key %q exceeds this platform's int range",
			label,
		)
	}
	return int(parsed.Uint64()), nil
}

func parseNoSchemaMapKey(key string, path string) (any, error) {
	if parsed, ok := parseSignedIntegerText(key); ok {
		return metadataIntegerFromBig(parsed, path)
	}
	if bytesValue, ok, err := parseNoSchemaBytes(key, path); ok || err != nil {
		return bytesValue, err
	}
	if err := validateMetadataText(key, path); err != nil {
		return nil, err
	}
	return key, nil
}

func parseNoSchemaString(value string, path string) (any, error) {
	if bytesValue, ok, err := parseNoSchemaBytes(value, path); ok || err != nil {
		return bytesValue, err
	}
	if err := validateMetadataText(value, path); err != nil {
		return nil, err
	}
	return value, nil
}

func parseNoSchemaBytes(value string, path string) ([]byte, bool, error) {
	if !strings.HasPrefix(value, "0x") {
		return nil, false, nil
	}
	hexValue := value[2:]
	for _, ch := range hexValue {
		if ch >= 'A' && ch <= 'F' {
			return nil, false, nil
		}
	}
	decoded, err := hex.DecodeString(hexValue)
	if err != nil {
		return nil, false, nil
	}
	if err := validateMetadataBytes(decoded, path); err != nil {
		return nil, true, err
	}
	return decoded, true, nil
}

func parseDetailedBytes(value string, path string) ([]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s: expected hex-encoded bytes: %w", path, err)
	}
	if err := validateMetadataBytes(decoded, path); err != nil {
		return nil, err
	}
	return decoded, nil
}

func parseJSONMetadataNumber(number json.Number, path string) (any, error) {
	parsed, ok := parseSignedIntegerText(number.String())
	if !ok {
		return nil, fmt.Errorf("%s: metadata number must be an integer", path)
	}
	return metadataIntegerFromBig(parsed, path)
}

func metadataIntegerFromBig(value *big.Int, path string) (any, error) {
	if value.Cmp(minMetadataInteger) < 0 ||
		value.Cmp(maxMetadataInteger) > 0 {
		return nil, fmt.Errorf(
			"%s: metadata integer %s is outside the supported range",
			path,
			value.String(),
		)
	}
	if value.Sign() >= 0 {
		if value.Cmp(maxInt64Metadata) <= 0 {
			return value.Int64(), nil
		}
		return value.Uint64(), nil
	}
	if value.Cmp(minInt64Metadata) >= 0 {
		return value.Int64(), nil
	}
	abs := new(big.Int).Neg(value)
	offset := new(big.Int).Sub(abs, big.NewInt(1))
	return NegativeInteger(offset.Uint64()), nil
}

func parseUnsignedIntegerText(value string) (*big.Int, bool) {
	if value == "" {
		return nil, false
	}
	if len(value) > 1 && value[0] == '0' {
		return nil, false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return nil, false
		}
	}
	parsed := new(big.Int)
	if _, ok := parsed.SetString(value, 10); !ok {
		return nil, false
	}
	return parsed, true
}

func parseSignedIntegerText(value string) (*big.Int, bool) {
	if strings.HasPrefix(value, "-") {
		parsed, ok := parseUnsignedIntegerText(value[1:])
		if !ok {
			return nil, false
		}
		parsed.Neg(parsed)
		return parsed, true
	}
	return parseUnsignedIntegerText(value)
}

func validateMetadataText(value string, path string) error {
	if len(value) > metadataStringMaxBytes {
		return fmt.Errorf(
			"%s: metadata text value exceeds %d UTF-8 bytes",
			path,
			metadataStringMaxBytes,
		)
	}
	return nil
}

func validateMetadataBytes(value []byte, path string) error {
	if len(value) > metadataStringMaxBytes {
		return fmt.Errorf(
			"%s: metadata bytes value exceeds %d bytes",
			path,
			metadataStringMaxBytes,
		)
	}
	return nil
}

func appendCBORType(dst []byte, major byte, value uint64) []byte {
	prefix := major << 5
	switch {
	case value < 24:
		return append(dst, prefix|byte(value))
	case value <= 0xff:
		return append(dst, prefix|24, byte(value))
	case value <= 0xffff:
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], uint16(value))
		dst = append(dst, prefix|25)
		return append(dst, buf[:]...)
	case value <= 0xffffffff:
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(value))
		dst = append(dst, prefix|26)
		return append(dst, buf[:]...)
	default:
		var buf [8]byte
		binary.BigEndian.PutUint64(buf[:], value)
		dst = append(dst, prefix|27)
		return append(dst, buf[:]...)
	}
}
