package Relay

import (
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// ReadKind extracts the kind discriminator from a CBOR-decoded value.
// fxamacker/cbor decodes positive integers as uint64 by default, so we
// must accept both uint64 and int64 here.
func ReadKind(v any) (int, error) {
	switch n := v.(type) {
	case uint64:
		if n > uint64(^uint(0)>>1) { // guard: larger than max int
			return 0, errors.New("kind out of range")
		}
		return int(n), nil
	case int64:
		if n < 0 {
			return 0, errors.New("kind must be non-negative")
		}
		if n > int64(^uint(0)>>1) {
			return 0, errors.New("kind out of range")
		}
		return int(n), nil
	default:
		return 0, errors.New("invalid type for kind; expected integer")
	}
}

type RelayInterface interface {
	Kind() int
	MarshalCBOR() ([]byte, error)
}

type SingleHostAddr struct {
	Port *uint16
	Ipv4 []byte
	Ipv6 []byte
}

func (v SingleHostAddr) Kind() int { return 0 }
func (v SingleHostAddr) MarshalCBOR() ([]byte, error) {
	if v.Ipv4 != nil && len(v.Ipv4) != 4 {
		return nil, fmt.Errorf("ipv4 must be 4 bytes when set, got %d", len(v.Ipv4))
	}
	if v.Ipv6 != nil && len(v.Ipv6) != 16 {
		return nil, fmt.Errorf("ipv6 must be 16 bytes when set, got %d", len(v.Ipv6))
	}
	return cbor.Marshal([]any{v.Kind(), v.Port, v.Ipv4, v.Ipv6})
}
func (v *SingleHostAddr) UnmarshalCBOR(data []byte) error {
	var arr []any
	if err := cbor.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) != 4 {
		return fmt.Errorf("expected array of length 4, got %d", len(arr))
	}
	// kind discriminator
	k, err := ReadKind(arr[0])
	if err != nil {
		return err
	}
	if k != 0 {
		return fmt.Errorf("unexpected kind %d for SingleHostAddr", k)
	}

	if arr[1] == nil {
		v.Port = nil
	} else {
		switch p := arr[1].(type) {
		case uint64:
			if p > 65535 {
				return fmt.Errorf("port out of range: %d", p)
			}
			pv := uint16(p)
			v.Port = &pv
		default:
			return errors.New("invalid type for port; expected unsigned integer or null")
		}
	}

	// ipv4
	if arr[2] == nil {
		v.Ipv4 = nil
	} else {
		b, ok := arr[2].([]byte)
		if !ok {
			return errors.New("invalid type for ipv4; expected byte string or null")
		}
		if len(b) != 4 {
			return fmt.Errorf("ipv4 must be 4 bytes, got %d", len(b))
		}
		v.Ipv4 = append([]byte(nil), b...)
	}

	// ipv6
	if arr[3] == nil {
		v.Ipv6 = nil
	} else {
		b, ok := arr[3].([]byte)
		if !ok {
			return errors.New("invalid type for ipv6; expected byte string or null")
		}
		if len(b) != 16 {
			return fmt.Errorf("ipv6 must be 16 bytes, got %d", len(b))
		}
		v.Ipv6 = append([]byte(nil), b...)
	}
	return nil
}

type SingleHostName struct {
	Port    *uint16
	DnsName string
}

func (v SingleHostName) Kind() int { return 1 }
func (v SingleHostName) MarshalCBOR() ([]byte, error) {
	if len(v.DnsName) > 128 {
		return nil, fmt.Errorf("dns name too long: %d", len(v.DnsName))
	}
	return cbor.Marshal([]any{v.Kind(), v.Port, v.DnsName})
}
func (v *SingleHostName) UnmarshalCBOR(data []byte) error {
	var arr []any
	if err := cbor.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) != 3 {
		return fmt.Errorf("expected array of length 3, got %d", len(arr))
	}
	// kind discriminator
	k, err := ReadKind(arr[0])
	if err != nil {
		return err
	}
	if k != 1 {
		return fmt.Errorf("unexpected kind %d for SingleHostName", k)
	}
	// port
	if arr[1] == nil {
		v.Port = nil
	} else {
		switch p := arr[1].(type) {
		case uint64:
			if p > 65535 {
				return fmt.Errorf("port out of range: %d", p)
			}
			pv := uint16(p)
			v.Port = &pv
		default:
			return errors.New("invalid type for port; expected unsigned integer or null")
		}
	}

	// dns name
	switch d := arr[2].(type) {
	case string:
		if len(d) > 128 {
			return fmt.Errorf("dns name too long: %d", len(d))
		}
		v.DnsName = d
	default:
		return errors.New("invalid type for dns name; expected string")
	}
	return nil
}

type MultiHostName struct {
	DnsName string
}

func (v MultiHostName) Kind() int { return 2 }
func (v MultiHostName) MarshalCBOR() ([]byte, error) {
	if len(v.DnsName) > 128 {
		return nil, fmt.Errorf("dns name too long: %d", len(v.DnsName))
	}
	return cbor.Marshal([]any{v.Kind(), v.DnsName})
}
func (v *MultiHostName) UnmarshalCBOR(data []byte) error {
	var arr []any
	if err := cbor.Unmarshal(data, &arr); err != nil {
		return err
	}
	if len(arr) != 2 {
		return fmt.Errorf("expected array of length 2, got %d", len(arr))
	}
	// kind discriminator
	k, err := ReadKind(arr[0])
	if err != nil {
		return err
	}
	if k != 2 {
		return fmt.Errorf("unexpected kind %d for MultiHostName", k)
	}

	// dns name
	switch d := arr[1].(type) {
	case string:
		if len(d) > 128 {
			return fmt.Errorf("dns name too long: %d", len(d))
		}
		v.DnsName = d
	default:
		return errors.New("invalid type for dns name; expected string")
	}
	return nil
}

// UnmarshalRelay unmarshals CBOR data into the appropriate Relay type based on the kind discriminator.
// It returns a RelayInterface and an error.
func UnmarshalRelay(data []byte) (RelayInterface, error) {
	// First, unmarshal to get the kind discriminator
	var arr []any
	if err := cbor.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, errors.New("empty array, cannot determine kind")
	}

	// Extract kind
	kind, err := ReadKind(arr[0])
	if err != nil {
		return nil, err
	}

	// Dispatch based on kind
	switch kind {
	case 0: // single_host_addr = (0, port/ nil, ipv4/ nil, ipv6/ nil)
		var result SingleHostAddr
		if err := result.UnmarshalCBOR(data); err != nil {
			return nil, err
		}
		return result, nil
	case 1: // single_host_name = (1, port/ nil, dns_name)
		var result SingleHostName
		if err := result.UnmarshalCBOR(data); err != nil {
			return nil, err
		}
		return result, nil
	case 2: // multi_host_name = (2, dns_name)
		var result MultiHostName
		if err := result.UnmarshalCBOR(data); err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unknown relay kind: %d", kind)
	}
}

type Relays []RelayInterface

func (v Relays) MarshalCBOR() ([]byte, error) {
	arr := make([][]byte, 0, len(v))
	for _, relay := range v {
		bz, err := relay.MarshalCBOR()
		if err != nil {
			return nil, err
		}
		arr = append(arr, bz)
	}
	var out []any
	for _, e := range arr {
		var v any
		if err := cbor.Unmarshal(e, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return cbor.Marshal(out)
}

func (v *Relays) UnmarshalCBOR(data []byte) error {
	var arr []any
	if err := cbor.Unmarshal(data, &arr); err != nil {
		return err
	}
	res := make(Relays, 0, len(arr))
	for _, item := range arr {
		marshaledRelay, err := cbor.Marshal(item)
		if err != nil {
			return err
		}
		relay, err := UnmarshalRelay(marshaledRelay)
		if err != nil {
			return err
		}
		res = append(res, relay)
	}
	*v = res
	return nil
}
