package Relay_test

import (
	"testing"

	"github.com/Salvionied/apollo/serialization/Relay"
	"github.com/fxamacker/cbor/v2"
)

func u16ptr(v uint16) *uint16 { return &v }

func TestSingleHostAddr_RoundTripAndErrors(t *testing.T) {
	success := []Relay.SingleHostAddr{
		{},
		{Port: u16ptr(0)},
		{Port: u16ptr(65535)},
		{Ipv4: []byte{1, 2, 3, 4}},
		{Ipv6: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
		{Port: u16ptr(8080), Ipv4: []byte{192, 168, 0, 1}},
		{Port: u16ptr(8080), Ipv6: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}},
		{Ipv4: []byte{10, 0, 0, 1}, Ipv6: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}},
		{Port: u16ptr(6000), Ipv4: []byte{8, 8, 8, 8}, Ipv6: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99}},
	}
	for i, v := range success {
		t.Run("roundtrip_success_#"+string(rune('A'+i)), func(t *testing.T) {
			data, err := v.MarshalCBOR()
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			var got Relay.SingleHostAddr
			if err := got.UnmarshalCBOR(data); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			// compare
			if (v.Port == nil) != (got.Port == nil) {
				t.Fatalf("port nil mismatch: want %v got %v", v.Port, got.Port)
			}
			if v.Port != nil && got.Port != nil && *v.Port != *got.Port {
				t.Fatalf("port mismatch: want %d got %d", *v.Port, *got.Port)
			}
			if string(v.Ipv4) != string(got.Ipv4) {
				t.Fatalf("ipv4 mismatch: want %v got %v", v.Ipv4, got.Ipv4)
			}
			if string(v.Ipv6) != string(got.Ipv6) {
				t.Fatalf("ipv6 mismatch: want %v got %v", v.Ipv6, got.Ipv6)
			}
		})
	}

	t.Run("marshal_error_ipv4_len", func(t *testing.T) {
		_, err := (Relay.SingleHostAddr{Ipv4: []byte{1, 2, 3}}).MarshalCBOR()
		if err == nil {
			t.Fatal("expected error for ipv4 length != 4")
		}
	})
	t.Run("marshal_error_ipv6_len", func(t *testing.T) {
		_, err := (Relay.SingleHostAddr{Ipv6: make([]byte, 15)}).MarshalCBOR()
		if err == nil {
			t.Fatal("expected error for ipv6 length != 16")
		}
	})

	t.Run("unmarshal_error_wrong_len", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, nil})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for wrong array length")
		}
	})

	t.Run("unmarshal_error_kind_mismatch", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, nil, nil, nil})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for unexpected kind")
		}
	})

	t.Run("unmarshal_error_port_out_of_range", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, uint64(70000), nil, nil})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for port out of range")
		}
	})

	t.Run("unmarshal_error_port_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, "not-a-number", nil, nil})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for port wrong type")
		}
	})

	t.Run("unmarshal_error_ipv4_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, nil, "not-bytes", nil})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for ipv4 wrong type")
		}
	})

	t.Run("unmarshal_error_ipv6_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, nil, nil, "not-bytes"})
		var v Relay.SingleHostAddr
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for ipv6 wrong type")
		}
	})
}

func TestSingleHostName_RoundTripAndErrors(t *testing.T) {
	success := []Relay.SingleHostName{
		{DnsName: "example.com"},
		{Port: u16ptr(0), DnsName: "a"},
		{Port: u16ptr(65535), DnsName: "x.y.z"},
		{DnsName: string(make([]byte, 128))},
	}
	for i, v := range success {
		t.Run("roundtrip_success_#"+string(rune('A'+i)), func(t *testing.T) {
			data, err := v.MarshalCBOR()
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			var got Relay.SingleHostName
			if err := got.UnmarshalCBOR(data); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if (v.Port == nil) != (got.Port == nil) {
				t.Fatalf("port nil mismatch")
			}
			if v.Port != nil && got.Port != nil && *v.Port != *got.Port {
				t.Fatalf("port mismatch: want %d got %d", *v.Port, *got.Port)
			}
			if v.DnsName != got.DnsName {
				t.Fatalf("dns mismatch: want %q got %q", v.DnsName, got.DnsName)
			}
		})
	}

	t.Run("marshal_error_dns_too_long", func(t *testing.T) {
		_, err := (Relay.SingleHostName{DnsName: string(make([]byte, 129))}).MarshalCBOR()
		if err == nil {
			t.Fatal("expected error for dns too long")
		}
	})
	t.Run("unmarshal_error_wrong_len", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, nil})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for wrong array length")
		}
	})
	t.Run("unmarshal_error_kind_mismatch", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{0, nil, "example.com"})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for unexpected kind")
		}
	})
	t.Run("unmarshal_error_port_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, "bad-port", "example.com"})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for port wrong type")
		}
	})
	t.Run("unmarshal_error_port_out_of_range", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, uint64(70000), "example.com"})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for port out of range")
		}
	})
	t.Run("unmarshal_error_dns_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, nil, []byte{1}})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for dns wrong type")
		}
	})
	t.Run("unmarshal_error_dns_too_long", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, nil, string(make([]byte, 129))})
		var v Relay.SingleHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for dns too long")
		}
	})
}

func TestMultiHostName_RoundTripAndErrors(t *testing.T) {
	success := []Relay.MultiHostName{
		{DnsName: "example.com"},
		{DnsName: string(make([]byte, 128))},
	}
	for i, v := range success {
		t.Run("roundtrip_success_#"+string(rune('A'+i)), func(t *testing.T) {
			data, err := v.MarshalCBOR()
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			var got Relay.MultiHostName
			if err := got.UnmarshalCBOR(data); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}
			if v.DnsName != got.DnsName {
				t.Fatalf("dns mismatch: want %q got %q", v.DnsName, got.DnsName)
			}
		})
	}

	t.Run("marshal_error_dns_too_long", func(t *testing.T) {
		_, err := (Relay.MultiHostName{DnsName: string(make([]byte, 129))}).MarshalCBOR()
		if err == nil {
			t.Fatal("expected error for dns too long")
		}
	})
	t.Run("unmarshal_error_wrong_len", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{2})
		var v Relay.MultiHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for wrong array length")
		}
	})
	t.Run("unmarshal_error_kind_mismatch", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{1, "example.com"})
		var v Relay.MultiHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for unexpected kind")
		}
	})
	t.Run("unmarshal_error_dns_wrong_type", func(t *testing.T) {
		bad, _ := cbor.Marshal([]any{2, []byte{1}})
		var v Relay.MultiHostName
		if err := v.UnmarshalCBOR(bad); err == nil {
			t.Fatal("expected error for dns wrong type")
		}
	})
}

func TestUnmarshalRelay_Dispatch(t *testing.T) {
	// single_host_addr
	aData, _ := cbor.Marshal([]any{0, nil, []byte{1, 2, 3, 4}, nil})
	a, err := Relay.UnmarshalRelay(aData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := a.(Relay.SingleHostAddr); !ok {
		t.Fatalf("expected SingleHostAddr, got %T", a)
	}

	// single_host_name
	sData, _ := cbor.Marshal([]any{1, nil, "example.com"})
	s, err := Relay.UnmarshalRelay(sData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := s.(Relay.SingleHostName); !ok {
		t.Fatalf("expected SingleHostName, got %T", s)
	}

	// multi_host_name
	mData, _ := cbor.Marshal([]any{2, "example.com"})
	m, err := Relay.UnmarshalRelay(mData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := m.(Relay.MultiHostName); !ok {
		t.Fatalf("expected MultiHostName, got %T", m)
	}

	// invalid kind
	badKind, _ := cbor.Marshal([]any{3, nil})
	if _, err := Relay.UnmarshalRelay(badKind); err == nil {
		t.Fatal("expected error for unknown relay kind")
	}

	// invalid array (empty)
	empty, _ := cbor.Marshal([]any{})
	if _, err := Relay.UnmarshalRelay(empty); err == nil {
		t.Fatal("expected error for empty array")
	}

	// invalid kind type
	wrongKindType, _ := cbor.Marshal([]any{"not-int", nil})
	if _, err := Relay.UnmarshalRelay(wrongKindType); err == nil {
		t.Fatal("expected error for invalid relay kind type")
	}
}

func TestRelays_MarshalUnmarshal(t *testing.T) {
	relays := Relay.Relays{
		Relay.SingleHostAddr{Ipv4: []byte{1, 2, 3, 4}},
		Relay.SingleHostName{Port: u16ptr(8080), DnsName: "example.com"},
		Relay.MultiHostName{DnsName: "pool.example"},
	}
	data, err := relays.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var got Relay.Relays
	if err := got.UnmarshalCBOR(data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(got) != len(relays) {
		t.Fatalf("length mismatch: want %d got %d", len(relays), len(got))
	}
	// basic sanity of types
	if _, ok := got[0].(Relay.SingleHostAddr); !ok {
		t.Fatalf("index 0 type mismatch")
	}
	if _, ok := got[1].(Relay.SingleHostName); !ok {
		t.Fatalf("index 1 type mismatch")
	}
	if _, ok := got[2].(Relay.MultiHostName); !ok {
		t.Fatalf("index 2 type mismatch")
	}

	// empty slice
	var empty Relay.Relays
	data, err = empty.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal empty error: %v", err)
	}
	var gotEmpty Relay.Relays
	if err := gotEmpty.UnmarshalCBOR(data); err != nil {
		t.Fatalf("unmarshal empty error: %v", err)
	}
	if len(gotEmpty) != 0 {
		t.Fatalf("expected empty relays after roundtrip")
	}
}
