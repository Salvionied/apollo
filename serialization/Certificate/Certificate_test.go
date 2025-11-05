package Certificate

import (
	"reflect"
	"testing"

	"github.com/Salvionied/apollo/serialization"
	RelayPkg "github.com/Salvionied/apollo/serialization/Relay"
	"github.com/fxamacker/cbor/v2"
)

func mustMarshalCert(t *testing.T, cert CertificateInterface) []byte {
	t.Helper()
	bz, err := cert.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	return bz
}

func roundTrip(t *testing.T, cert CertificateInterface) CertificateInterface {
	t.Helper()
	bz := mustMarshalCert(t, cert)
	decoded, err := UnmarshalCert(bz)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	// Re-marshal and compare bytes for stability and semantic equality
	bz2, err := decoded.MarshalCBOR()
	if err != nil {
		t.Fatalf("re-marshal error: %v", err)
	}
	// Compare via decoded canonical arrays (avoid differences in CBOR head encodings)
	var v1, v2 any
	if err := cbor.Unmarshal(bz, &v1); err != nil {
		t.Fatalf("unmarshal compare v1: %v", err)
	}
	if err := cbor.Unmarshal(bz2, &v2); err != nil {
		t.Fatalf("unmarshal compare v2: %v", err)
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("round-trip mismatch:\norig=%#v\ndeco=%#v", v1, v2)
	}
	return decoded
}

// DeepEqual helper using reflect; placed here to avoid importing reflect throughout
// but use central serialization if available; fallback minimal here.

func TestStakeRegistrationRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{1, 2, 3}}}
	cert := StakeRegistration{Stake: cred}
	_ = roundTrip(t, cert)
}

func TestPoolRegistrationWithRelaysRoundTrip(t *testing.T) {
	// Build sample PoolParams with multiple relay variants
	op := serialization.PubKeyHash{}
	op[0] = 0xAA
	params := PoolParams{
		Operator:      op,
		VrfKeyHash:    []byte{0x01, 0x02, 0x03},
		Pledge:        1_000,
		Cost:          5,
		Margin:        UnitInterval{Num: 1, Den: 2},
		RewardAccount: []byte{0xAB, 0xCD},
		PoolOwners:    []serialization.PubKeyHash{op},
		Relays: RelayPkg.Relays{
			RelayPkg.SingleHostAddr{Port: uint16Ptr(3000), Ipv4: []byte{1, 2, 3, 4}, Ipv6: nil},
			RelayPkg.SingleHostName{Port: uint16Ptr(6000), DnsName: "relay.example.com"},
			RelayPkg.MultiHostName{DnsName: "pool.example.net"},
		},
		PoolMetadata: &struct {
			_    struct{} `cbor:",toarray"`
			Url  string
			Hash []byte
		}{Url: "https://meta.example", Hash: []byte{0x10, 0x20}},
	}
	cert := PoolRegistration{Params: params}
	_ = roundTrip(t, cert)
}

func TestRegDRepCertAnchorsRoundTrip(t *testing.T) {
	// With nil anchor
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xDE, 0xAD}}}
	certNil := RegDRepCert{Cred: cred, Coin: 42, Anchor: nil}
	_ = roundTrip(t, certNil)

	// With anchor present
	anch := &Anchor{Url: "https://anchor", DataHash: []byte{0x01, 0x02}}
	certWith := RegDRepCert{Cred: cred, Coin: 42, Anchor: anch}
	_ = roundTrip(t, certWith)
}

func TestStakeVoteDelegCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x01}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x11
	drep := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0xBE, 0xEF}}}
	cert := StakeVoteDelegCert{Stake: cred, PoolKeyHash: pool, Drep: drep}
	_ = roundTrip(t, cert)
}

func TestPoolRegistrationVariants(t *testing.T) {
	ownerA := serialization.PubKeyHash{}
	ownerA[0] = 0xA1
	ownerB := serialization.PubKeyHash{}
	ownerB[0] = 0xB2

	cases := []PoolParams{
		{
			Operator:      ownerA,
			VrfKeyHash:    []byte{0x01},
			Pledge:        0,
			Cost:          0,
			Margin:        UnitInterval{Num: 0, Den: 1},
			RewardAccount: nil,
			PoolOwners:    nil,
			Relays:        RelayPkg.Relays{},
			PoolMetadata:  nil,
		},
		{
			Operator:      ownerB,
			VrfKeyHash:    []byte{0xFF, 0xEE},
			Pledge:        123,
			Cost:          456,
			Margin:        UnitInterval{Num: 1, Den: 1},
			RewardAccount: []byte{},
			PoolOwners:    []serialization.PubKeyHash{ownerA, ownerB},
			Relays: RelayPkg.Relays{
				RelayPkg.SingleHostAddr{Port: nil, Ipv4: nil, Ipv6: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
			},
			PoolMetadata: &struct {
				_    struct{} `cbor:",toarray"`
				Url  string
				Hash []byte
			}{Url: "", Hash: []byte{}},
		},
	}

	for i, params := range cases {
		cert := PoolRegistration{Params: params}
		_ = roundTrip(t, cert)
		_ = i // keep i referenced for clarity; subtests not needed here
	}
}

func TestCoinsAndEpochEdges(t *testing.T) {
	cred := Credential{Code: 1, Hash: serialization.ConstrainedBytes{Payload: []byte{0xCA, 0xFE}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x99

	// coin edges
	_ = roundTrip(t, RegCert{Stake: cred, Coin: 0})
	_ = roundTrip(t, UnregCert{Stake: cred, Coin: 0})
	_ = roundTrip(t, RegCert{Stake: cred, Coin: 1 << 40})
	_ = roundTrip(t, UnregCert{Stake: cred, Coin: 1 << 40})

	// epoch edges
	_ = roundTrip(t, PoolRetirement{PoolKeyHash: pool, EpochNo: 0})
	_ = roundTrip(t, PoolRetirement{PoolKeyHash: pool, EpochNo: 1 << 40})
}

func TestDrepVariants(t *testing.T) {
	stake := Credential{Code: 2, Hash: serialization.ConstrainedBytes{Payload: []byte{0xDE}}}
	// drep with bytes cred
	drepBytes := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0x01}}}
	_ = roundTrip(t, VoteDelegCert{Stake: stake, Drep: drepBytes})
	// drep with nil cred (should still round-trip as structure allows nil pointer)
	drepNil := Drep{Code: 3, Credential: nil}
	_ = roundTrip(t, VoteDelegCert{Stake: stake, Drep: drepNil})
}

func TestStakeVoteRegDelegCombinations(t *testing.T) {
	stake := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x01
	drep := Drep{Code: 2, Credential: &serialization.ConstrainedBytes{Payload: []byte{0xAA, 0xBB}}}

	_ = roundTrip(t, StakeVoteDelegCert{Stake: stake, PoolKeyHash: pool, Drep: drep})
	_ = roundTrip(t, StakeRegDelegCert{Stake: stake, PoolKeyHash: pool, Coin: 1})
	_ = roundTrip(t, VoteRegDelegCert{Stake: stake, Drep: drep, Coin: 2})
	_ = roundTrip(t, StakeVoteRegDelegCert{Stake: stake, PoolKeyHash: pool, Drep: drep, Coin: 3})
}

// Negative/expect-error tests

func TestUnmarshalCert_InvalidEmptyArray(t *testing.T) {
	// CBOR for empty array []
	bz, _ := cbor.Marshal([]any{})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for empty array")
	}
}

func TestUnmarshalCert_InvalidKindType(t *testing.T) {
	// kind is string -> invalid
	bz, _ := cbor.Marshal([]any{"not-an-int"})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for invalid kind type")
	}
}

func TestUnmarshalCert_NegativeKind(t *testing.T) {
	// kind is negative int64 -> invalid per ReadKind
	bz, _ := cbor.Marshal([]any{int64(-1)})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for negative kind")
	}
}

func TestUnmarshalCert_KindOutOfRange(t *testing.T) {
	// kind is very large uint64 -> out of int range
	bz, _ := cbor.Marshal([]any{^uint64(0)})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for out-of-range kind")
	}
}

func TestUnmarshalCert_WrongLengths(t *testing.T) {
	// StakeRegistration requires 2 elements
	bz, _ := cbor.Marshal([]any{uint64(0)})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for stake_registration wrong length")
	}
	// StakeDelegation requires 3 elements
	wrong, _ := cbor.Marshal([]any{uint64(2), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for stake_delegation wrong length")
	}
	// PoolRegistration requires 2
	wrong, _ = cbor.Marshal([]any{uint64(3)})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for pool_registration wrong length")
	}
	// PoolRetirement requires 3
	wrong, _ = cbor.Marshal([]any{uint64(4), serialization.PubKeyHash{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for pool_retirement wrong length")
	}
	// RegCert requires 3
	wrong, _ = cbor.Marshal([]any{uint64(7), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for reg_cert wrong length")
	}
	// UnregCert requires 3
	wrong, _ = cbor.Marshal([]any{uint64(8), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for unreg_cert wrong length")
	}
	// VoteDelegCert requires 3
	wrong, _ = cbor.Marshal([]any{uint64(9), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for vote_deleg_cert wrong length")
	}
	// StakeVoteDelegCert requires 4
	wrong, _ = cbor.Marshal([]any{uint64(10), Credential{}, serialization.PubKeyHash{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for stake_vote_deleg_cert wrong length")
	}
	// StakeRegDelegCert requires 4
	wrong, _ = cbor.Marshal([]any{uint64(11), Credential{}, serialization.PubKeyHash{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for stake_reg_deleg_cert wrong length")
	}
	// VoteRegDelegCert requires 4
	wrong, _ = cbor.Marshal([]any{uint64(12), Credential{}, Drep{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for vote_reg_deleg_cert wrong length")
	}
	// StakeVoteRegDelegCert requires 5
	wrong, _ = cbor.Marshal([]any{uint64(13), Credential{}, serialization.PubKeyHash{}, Drep{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for stake_vote_reg_deleg_cert wrong length")
	}
	// AuthCommitteeHotCert requires 3
	wrong, _ = cbor.Marshal([]any{uint64(14), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for auth_committee_hot_cert wrong length")
	}
	// ResignCommitteeColdCert requires >=2
	wrong, _ = cbor.Marshal([]any{uint64(15)})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for resign_committee_cold_cert wrong length")
	}
	// RegDRepCert requires >=3
	wrong, _ = cbor.Marshal([]any{uint64(16), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for reg_drep_cert wrong length")
	}
	// UnregDRepCert requires 3
	wrong, _ = cbor.Marshal([]any{uint64(17), Credential{}})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for unreg_drep_cert wrong length")
	}
	// UpdateDRepCert requires >=2
	wrong, _ = cbor.Marshal([]any{uint64(18)})
	if _, err := UnmarshalCert(wrong); err == nil {
		t.Fatalf("expected error for update_drep_cert wrong length")
	}
}

func TestUnmarshalCert_WrongTypes(t *testing.T) {
	// pool_retirement expects (4, pool_keyhash(bytes[28]), epoch(uint))
	// give wrong types
	bz, _ := cbor.Marshal([]any{uint64(4), "not-bytes", "not-epoch"})
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for pool_retirement wrong types")
	}

	// reg_drep_cert expects (16, drep_credential, coin, anchor?/nil)
	// give anchor wrong type
	bz2, _ := cbor.Marshal([]any{uint64(16), Credential{}, int64(1), "bad-anchor"})
	if _, err := UnmarshalCert(bz2); err == nil {
		t.Fatalf("expected error for reg_drep wrong anchor type")
	}

	// stake_vote_deleg_cert expects (10, stake_credential, pool_keyhash, drep)
	// supply wrong pool key type (string)
	bz3, _ := cbor.Marshal([]any{uint64(10), Credential{}, "bad-pool", Drep{}})
	if _, err := UnmarshalCert(bz3); err == nil {
		t.Fatalf("expected error for stake_vote_deleg_cert wrong pool key type")
	}
}

func TestUnmarshalCert_PoolRegistration_InvalidRelays(t *testing.T) {
	// Build pool params with an invalid relay item encoded manually
	// Invalid: relay kind 0 but ipv4 length wrong (3 bytes)
	invalidRelay := []any{uint64(0), nil, []byte{1, 2, 3}, nil}
	params := []any{
		uint64(3), // pool_registration kind
		[]any{
			// PoolParams as array per toarray tag, fields in order
			serialization.PubKeyHash{},       // Operator
			[]byte{0x01},                     // VrfKeyHash
			int64(1),                         // Pledge
			int64(1),                         // Cost
			[]any{int64(0), int64(1)},        // UnitInterval {Num, Den}
			[]byte{0xAA},                     // RewardAccount
			[]any{},                          // PoolOwners (empty)
			[]any{invalidRelay},              // Relays array with one invalid relay
			[]any{"https://x", []byte{0x01}}, // PoolMetadata
		},
	}
	bz, _ := cbor.Marshal(params)
	if _, err := UnmarshalCert(bz); err == nil {
		t.Fatalf("expected error for pool_registration with invalid relay")
	}
}

func TestCertificates_InvalidItem(t *testing.T) {
	// certificates array containing an item with bad kind type
	arr := []any{
		[]any{"bad-kind"},
	}
	bz, _ := cbor.Marshal(arr)
	var cs Certificates
	if err := cs.UnmarshalCBOR(bz); err == nil {
		t.Fatalf("expected error when certificates contain invalid item")
	}
	// certificates array containing item with wrong length for known kind
	arr = []any{
		[]any{uint64(7), []any{int64(0), []byte{}}}, // malformed inner array (won't map to Credential properly)
	}
	bz, _ = cbor.Marshal(arr)
	if err := cs.UnmarshalCBOR(bz); err == nil {
		t.Fatalf("expected error for certificates item with wrong structure")
	}
}

func TestStakeDeregistrationRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x04}}}
	cert := StakeDeregistration{Stake: cred}
	_ = roundTrip(t, cert)
}

func TestStakeDelegationRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x05}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x21
	cert := StakeDelegation{Stake: cred, PoolKeyHash: pool}
	_ = roundTrip(t, cert)
}

func TestPoolRetirementRoundTrip(t *testing.T) {
	pool := serialization.PubKeyHash{}
	pool[0] = 0x33
	cert := PoolRetirement{PoolKeyHash: pool, EpochNo: 123456789}
	_ = roundTrip(t, cert)
}

func TestRegAndUnregCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x06}}}
	_ = roundTrip(t, RegCert{Stake: cred, Coin: 100})
	_ = roundTrip(t, UnregCert{Stake: cred, Coin: 50})
}

func TestVoteDelegCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x07}}}
	drep := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0x01, 0x02, 0x03}}}
	cert := VoteDelegCert{Stake: cred, Drep: drep}
	_ = roundTrip(t, cert)
}

func TestStakeRegDelegCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x08}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x44
	cert := StakeRegDelegCert{Stake: cred, PoolKeyHash: pool, Coin: 500}
	_ = roundTrip(t, cert)
}

func TestVoteRegDelegCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x09}}}
	drep := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0x0A}}}
	cert := VoteRegDelegCert{Stake: cred, Drep: drep, Coin: 250}
	_ = roundTrip(t, cert)
}

func TestStakeVoteRegDelegCertRoundTrip(t *testing.T) {
	cred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0x0B}}}
	pool := serialization.PubKeyHash{}
	pool[0] = 0x55
	drep := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0x0C}}}
	cert := StakeVoteRegDelegCert{Stake: cred, PoolKeyHash: pool, Drep: drep, Coin: 750}
	_ = roundTrip(t, cert)
}

func TestAuthCommitteeHotCertRoundTrip(t *testing.T) {
	cold := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xAA}}}
	hot := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xBB}}}
	cert := AuthCommitteeHotCert{Cold: cold, Hot: hot}
	_ = roundTrip(t, cert)
}

func TestResignCommitteeColdCertAnchorsRoundTrip(t *testing.T) {
	cold := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xCC}}}
	// nil anchor
	_ = roundTrip(t, ResignCommitteeColdCert{Cold: cold, Anchor: nil})
	// with anchor
	anch := &Anchor{Url: "https://resign", DataHash: []byte{0xAA, 0xBB}}
	_ = roundTrip(t, ResignCommitteeColdCert{Cold: cold, Anchor: anch})
}

func TestUnregDRepCertRoundTrip(t *testing.T) {
	drepCred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xDD}}}
	cert := UnregDRepCert{Cred: drepCred, Coin: 5}
	_ = roundTrip(t, cert)
}

func TestUpdateDRepCertAnchorsRoundTrip(t *testing.T) {
	drepCred := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{0xEE}}}
	// nil anchor
	_ = roundTrip(t, UpdateDRepCert{Cred: drepCred, Anchor: nil})
	// with anchor
	anch := &Anchor{Url: "https://update", DataHash: []byte{0x01}}
	_ = roundTrip(t, UpdateDRepCert{Cred: drepCred, Anchor: anch})
}

// Helpers
func uint16Ptr(v uint16) *uint16 { return &v }

// ---- Certificates (collection) tests ----

func TestCertificatesCollection_RoundTrip_AllKinds(t *testing.T) {
	// Build one of each kind to ensure the collection codec works end-to-end
	owner := serialization.PubKeyHash{}
	owner[0] = 0x42
	pool := serialization.PubKeyHash{}
	pool[0] = 0x24

	relays := RelayPkg.Relays{
		RelayPkg.SingleHostAddr{Port: uint16Ptr(1234), Ipv4: []byte{9, 8, 7, 6}},
		RelayPkg.SingleHostName{Port: nil, DnsName: "mix.example"},
	}
	params := PoolParams{
		Operator:      owner,
		VrfKeyHash:    []byte{0xAA},
		Pledge:        10,
		Cost:          20,
		Margin:        UnitInterval{Num: 1, Den: 10},
		RewardAccount: []byte{0x01},
		PoolOwners:    []serialization.PubKeyHash{owner},
		Relays:        relays,
		PoolMetadata: &struct {
			_    struct{} `cbor:",toarray"`
			Url  string
			Hash []byte
		}{Url: "https://pool.meta", Hash: []byte{0xCA}},
	}

	stake := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{1}}}
	credA := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{2}}}
	credB := Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{3}}}
	drep := Drep{Code: 0, Credential: &serialization.ConstrainedBytes{Payload: []byte{0xAA}}}
	anch := &Anchor{Url: "https://anch", DataHash: []byte{0x01}}

	all := Certificates{
		StakeRegistration{Stake: stake},                                             // 0
		StakeDeregistration{Stake: credA},                                           // 1
		StakeDelegation{Stake: credA, PoolKeyHash: pool},                            // 2
		PoolRegistration{Params: params},                                            // 3
		PoolRetirement{PoolKeyHash: pool, EpochNo: 99},                              // 4
		RegCert{Stake: stake, Coin: 7},                                              // 7
		UnregCert{Stake: stake, Coin: 3},                                            // 8
		VoteDelegCert{Stake: stake, Drep: drep},                                     // 9
		StakeVoteDelegCert{Stake: credB, PoolKeyHash: pool, Drep: drep},             // 10
		StakeRegDelegCert{Stake: stake, PoolKeyHash: pool, Coin: 1},                 // 11
		VoteRegDelegCert{Stake: stake, Drep: drep, Coin: 2},                         // 12
		StakeVoteRegDelegCert{Stake: stake, PoolKeyHash: pool, Drep: drep, Coin: 3}, // 13
		AuthCommitteeHotCert{Cold: credA, Hot: credB},                               // 14
		ResignCommitteeColdCert{Cold: credA, Anchor: anch},                          // 15
		RegDRepCert{Cred: credA, Coin: 5, Anchor: anch},                             // 16
		UnregDRepCert{Cred: credB, Coin: 6},                                         // 17
		UpdateDRepCert{Cred: credA, Anchor: nil},                                    // 18
	}

	// round trip collection
	bz, err := all.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal certificates: %v", err)
	}
	var got Certificates
	if err := (&got).UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal certificates: %v", err)
	}

	// Re-marshal and compare canonical decoded structures
	bz2, err := got.MarshalCBOR()
	if err != nil {
		t.Fatalf("re-marshal certificates: %v", err)
	}
	var v1, v2 any
	if err := cbor.Unmarshal(bz, &v1); err != nil {
		t.Fatalf("unmarshal v1: %v", err)
	}
	if err := cbor.Unmarshal(bz2, &v2); err != nil {
		t.Fatalf("unmarshal v2: %v", err)
	}
	if !reflect.DeepEqual(v1, v2) {
		t.Fatalf("certificates round-trip mismatch:\norig=%#v\nrt=%#v", v1, v2)
	}
}

func TestCertificatesCollection_EmptyAndMixed(t *testing.T) {
	// Empty
	empty := Certificates{}
	bz, err := empty.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	var got Certificates
	if err := (&got).UnmarshalCBOR(bz); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}

	// Mixed order small set
	pool := serialization.PubKeyHash{}
	pool[0] = 7
	mixed := Certificates{
		UnregDRepCert{Cred: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{1}}}, Coin: 0},
		StakeRegistration{Stake: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{2}}}},
		PoolRetirement{PoolKeyHash: pool, EpochNo: 1},
	}
	bz2, err := mixed.MarshalCBOR()
	if err != nil {
		t.Fatalf("marshal mixed: %v", err)
	}
	var got2 Certificates
	if err := (&got2).UnmarshalCBOR(bz2); err != nil {
		t.Fatalf("unmarshal mixed: %v", err)
	}
}

// ---- Fuzz tests (non-panicking guarantees and round-trip where parseable) ----

func FuzzUnmarshalCert(f *testing.F) {
	// Seed with a few valid encodings
	seed := []CertificateInterface{
		StakeRegistration{Stake: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{1}}}},
		PoolRegistration{Params: PoolParams{Operator: serialization.PubKeyHash{}, Relays: RelayPkg.Relays{}}},
		RegDRepCert{Cred: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{2}}}, Coin: 1, Anchor: nil},
	}
	for _, s := range seed {
		if bz, err := s.MarshalCBOR(); err == nil {
			f.Add(bz)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		cert, err := UnmarshalCert(data)
		if err != nil {
			return // reject invalid inputs; only assert no panic
		}
		// round-trip stability: re-encode and re-decode
		bz2, err := cert.MarshalCBOR()
		if err != nil {
			t.Fatalf("marshal after parse: %v", err)
		}
		cert2, err := UnmarshalCert(bz2)
		if err != nil {
			t.Fatalf("unmarshal re-encoded: %v", err)
		}
		// Compare semantic equivalence: same kind and re-encode to same bytes
		if cert.Kind() != cert2.Kind() {
			t.Fatalf("kind mismatch after round-trip: %d vs %d", cert.Kind(), cert2.Kind())
		}
		bz3, err := cert2.MarshalCBOR()
		if err != nil {
			t.Fatalf("marshal re-decoded: %v", err)
		}
		// Compare canonical CBOR structures (may differ from input due to normalization)
		var v2, v3 any
		if err := cbor.Unmarshal(bz2, &v2); err != nil {
			t.Fatalf("unmarshal bz2: %v", err)
		}
		if err := cbor.Unmarshal(bz3, &v3); err != nil {
			t.Fatalf("unmarshal bz3: %v", err)
		}
		if !reflect.DeepEqual(v2, v3) {
			t.Fatalf("round-trip not idempotent: %#v vs %#v", v2, v3)
		}
	})
}

func FuzzCertificatesUnmarshal(f *testing.F) {
	// Seeds: empty, one, multi
	empty := Certificates{}
	if bz, err := empty.MarshalCBOR(); err == nil {
		f.Add(bz)
	}
	one := Certificates{StakeRegistration{Stake: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{3}}}}}
	if bz, err := one.MarshalCBOR(); err == nil {
		f.Add(bz)
	}
	multi := Certificates{
		StakeRegistration{Stake: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{1}}}},
		UnregDRepCert{Cred: Credential{Code: 0, Hash: serialization.ConstrainedBytes{Payload: []byte{2}}}, Coin: 0},
	}
	if bz, err := multi.MarshalCBOR(); err == nil {
		f.Add(bz)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var cs Certificates
		if err := cs.UnmarshalCBOR(data); err != nil {
			return
		}
		// round-trip: re-encode and re-decode
		bz2, err := cs.MarshalCBOR()
		if err != nil {
			t.Fatalf("marshal after parse: %v", err)
		}
		var cs2 Certificates
		if err := cs2.UnmarshalCBOR(bz2); err != nil {
			t.Fatalf("unmarshal re-encoded: %v", err)
		}
		// Compare semantic equivalence: same length and kinds
		if len(cs) != len(cs2) {
			t.Fatalf("length mismatch after round-trip: %d vs %d", len(cs), len(cs2))
		}
		for i := range cs {
			if cs[i].Kind() != cs2[i].Kind() {
				t.Fatalf("cert[%d] kind mismatch: %d vs %d", i, cs[i].Kind(), cs2[i].Kind())
			}
		}
		// Verify idempotency: re-encode should produce same bytes
		bz3, err := cs2.MarshalCBOR()
		if err != nil {
			t.Fatalf("marshal re-decoded: %v", err)
		}
		var v2, v3 any
		if err := cbor.Unmarshal(bz2, &v2); err != nil {
			t.Fatalf("unmarshal bz2: %v", err)
		}
		if err := cbor.Unmarshal(bz3, &v3); err != nil {
			t.Fatalf("unmarshal bz3: %v", err)
		}
		if !reflect.DeepEqual(v2, v3) {
			t.Fatalf("round-trip not idempotent: %#v vs %#v", v2, v3)
		}
	})
}
