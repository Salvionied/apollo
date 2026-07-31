package apollo

import (
	"bytes"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// Every address in this file is synthetic: the payment and staking
// credentials are fixed byte patterns, so no real wallet, no real user and no
// real funds are involved. They exercise textual address decoding only.

// syntheticAddress builds an address of the given type and network from fixed
// credential bytes.
func syntheticAddress(t *testing.T, addrType, network uint8) common.Address {
	t.Helper()
	payment := bytes.Repeat([]byte{0xaa}, common.AddressHashSize)
	stake := bytes.Repeat([]byte{0xbb}, common.AddressHashSize)
	switch addrType {
	case common.AddressTypeKeyKey,
		common.AddressTypeScriptKey,
		common.AddressTypeKeyScript,
		common.AddressTypeScriptScript:
	case common.AddressTypeKeyNone, common.AddressTypeScriptNone:
		stake = nil
	case common.AddressTypeNoneKey, common.AddressTypeNoneScript:
		payment = nil
	default:
		t.Fatalf("unsupported synthetic address type %d", addrType)
	}
	addr, err := common.NewAddressFromParts(addrType, network, payment, stake)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

// mainnetSweepAddress is the synthetic mainnet base address the corruption
// sweep mutates. Mainnet matters: every character of an "addr1..." string is
// also a legal base58 character, which is what makes the bech32-to-base58
// fallthrough dangerous there and harmless on "addr_test1..." addresses.
func mainnetSweepAddress(t *testing.T) string {
	t.Helper()
	addr := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	return addr.String()
}

// addressCorruptionAlphabet is the bech32 data charset plus characters that
// are illegal in bech32 but legal in base58 ("b", "i", "o") and the bech32
// separator ("1"), which moves the human-readable part when substituted in.
const addressCorruptionAlphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l1bio"

// addressCorruptions returns every single-character corruption of addr that
// leaves the human-readable part intact: substitution, insertion, deletion,
// doubling and truncation. The HRP is preserved on purpose, because that is
// what a realistic typo looks like and what routes the string into the
// dangerous decode path.
func addressCorruptions(t *testing.T, addr string) []string {
	t.Helper()
	sep := strings.IndexByte(addr, '1')
	if sep < 0 {
		t.Fatalf("address %q has no bech32 separator", addr)
	}
	start := sep + 1
	capacity := 8 * len(addr) * len(addressCorruptionAlphabet)
	seen := make(map[string]struct{}, capacity)
	out := make([]string, 0, capacity)
	add := func(candidate string) {
		if candidate == addr {
			return
		}
		if _, dup := seen[candidate]; dup {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	for i := start; i < len(addr); i++ {
		// substitute
		for _, c := range addressCorruptionAlphabet {
			add(addr[:i] + string(c) + addr[i+1:])
		}
		// delete
		add(addr[:i] + addr[i+1:])
		// double
		add(addr[:i] + addr[i:i+1] + addr[i:])
		// truncate
		add(addr[:i])
	}
	// insert, including at the very end
	for i := start; i <= len(addr); i++ {
		for _, c := range addressCorruptionAlphabet {
			add(addr[:i] + string(c) + addr[i:])
		}
	}
	return out
}

// addressEntryPoints are the builder entry points that turn caller-supplied
// address text into an address Apollo will pay, spend from, or send change to.
func addressEntryPoints() []struct {
	name  string
	parse func(string) error
} {
	return []struct {
		name  string
		parse func(string) error
	}{
		{
			"PayToAddressBech32",
			func(text string) error {
				_, err := New(setupFixedContext()).
					PayToAddressBech32(text, 2_000_000)
				return err
			},
		},
		{
			"SetChangeAddressBech32",
			func(text string) error {
				_, err := New(setupFixedContext()).
					SetChangeAddressBech32(text)
				return err
			},
		},
		{
			"AddInputAddressFromBech32",
			func(text string) error {
				_, err := New(setupFixedContext()).
					AddInputAddressFromBech32(text)
				return err
			},
		},
		{
			"NewPayment",
			func(text string) error {
				_, err := NewPayment(text, 2_000_000, nil)
				return err
			},
		},
	}
}

// TestAddressEntryPointsRejectCorruptedMainnetAddress sweeps single-character
// corruptions of a mainnet address through every entry point. A bech32
// checksum failure must never be retried as base58: on master these inputs
// silently decode into unrelated bytes and Apollo builds a payment to a
// different recipient.
func TestAddressEntryPointsRejectCorruptedMainnetAddress(t *testing.T) {
	valid := mainnetSweepAddress(t)
	corruptions := addressCorruptions(t, valid)
	if len(corruptions) < 6000 {
		t.Fatalf("sweep is too small: %d corruptions", len(corruptions))
	}
	for _, entry := range addressEntryPoints() {
		accepted := 0
		var samples []string
		for _, corrupted := range corruptions {
			if err := entry.parse(corrupted); err == nil {
				accepted++
				if len(samples) < 3 {
					samples = append(samples, corrupted)
				}
			}
		}
		if accepted != 0 {
			t.Errorf(
				"%s accepted %d of %d corrupted mainnet addresses; e.g. %v",
				entry.name,
				accepted,
				len(corruptions),
				samples,
			)
		} else {
			t.Logf(
				"%s rejected all %d corrupted mainnet addresses",
				entry.name,
				len(corruptions),
			)
		}
	}
}

// TestAddressEntryPointsAcceptValidAddresses guards against the validation
// being too strict: every canonical address form must still round-trip.
func TestAddressEntryPointsAcceptValidAddresses(t *testing.T) {
	types := []uint8{
		common.AddressTypeKeyKey,
		common.AddressTypeScriptKey,
		common.AddressTypeKeyScript,
		common.AddressTypeScriptScript,
		common.AddressTypeKeyNone,
		common.AddressTypeScriptNone,
		common.AddressTypeNoneKey,
		common.AddressTypeNoneScript,
	}
	networks := []uint8{
		common.AddressNetworkTestnet,
		common.AddressNetworkMainnet,
	}
	valid := make([]string, 0, len(types)*len(networks)+1)
	for _, network := range networks {
		for _, addrType := range types {
			valid = append(
				valid,
				syntheticAddress(t, addrType, network).String(),
			)
		}
	}
	byron, err := common.NewByronAddressFromParts(
		common.ByronAddressTypePubkey,
		bytes.Repeat([]byte{0xaa}, common.AddressHashSize),
		common.ByronAddressAttributes{},
	)
	if err != nil {
		t.Fatal(err)
	}
	valid = append(valid, byron.String())

	for _, entry := range addressEntryPoints() {
		for _, text := range valid {
			if err := entry.parse(text); err != nil {
				t.Errorf("%s rejected valid address %q: %v",
					entry.name, text, err)
			}
		}
	}
}

// TestPayToAddressBech32PreservesReceiver verifies the accepted address is
// the one the caller asked for, byte for byte.
func TestPayToAddressBech32PreservesReceiver(t *testing.T) {
	want := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	a, err := New(setupFixedContext()).
		PayToAddressBech32(want.String(), 2_000_000)
	if err != nil {
		t.Fatal(err)
	}
	payment, ok := a.payments[0].(*Payment)
	if !ok {
		t.Fatalf("unexpected payment type %T", a.payments[0])
	}
	got, err := payment.Receiver.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := want.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, wantBytes) {
		t.Fatalf("receiver = %x, want %x", got, wantBytes)
	}
}

// TestAddressEntryPointsRejectTrailingPayloadBytes covers the case that came
// closest to being fatal in the wild: a base-address payload with a single
// extra byte, which the ledger's lenient Byron-compatibility path can accept.
func TestAddressEntryPointsRejectTrailingPayloadBytes(t *testing.T) {
	base := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	raw, err := base.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1+2*common.AddressHashSize {
		t.Fatalf("unexpected base address length %d", len(raw))
	}
	withTrailing, err := common.NewAddressFromBytes(append(raw, 0xcc))
	if err != nil {
		t.Fatal(err)
	}
	text := withTrailing.String()
	for _, entry := range addressEntryPoints() {
		if err := entry.parse(text); err == nil {
			t.Errorf("%s accepted a 58-byte address payload %q",
				entry.name, text)
		}
	}
}

// TestAddressEntryPointsAcceptPointerAddress guards the variable-length
// pointer address form against the payload-length validation.
func TestAddressEntryPointsAcceptPointerAddress(t *testing.T) {
	pointers := [][]byte{
		{0, 0, 0},
		{0x81, 0x00, 0x7f, 0x01},
	}
	for _, pointer := range pointers {
		addr, err := common.NewAddressFromParts(
			common.AddressTypeKeyPointer,
			common.AddressNetworkMainnet,
			bytes.Repeat([]byte{0xaa}, common.AddressHashSize),
			pointer,
		)
		if err != nil {
			t.Fatal(err)
		}
		text := addr.String()
		for _, entry := range addressEntryPoints() {
			if err := entry.parse(text); err != nil {
				t.Errorf("%s rejected pointer address %q: %v",
					entry.name, text, err)
			}
		}
	}
}

// TestAddressEntryPointsRejectPointerAddressTrailingBytes covers trailing
// bytes after a pointer address's variable-length staking payload.
func TestAddressEntryPointsRejectPointerAddressTrailingBytes(t *testing.T) {
	addr, err := common.NewAddressFromParts(
		common.AddressTypeKeyPointer,
		common.AddressNetworkMainnet,
		bytes.Repeat([]byte{0xaa}, common.AddressHashSize),
		[]byte{0, 0, 0, 0xcc},
	)
	if err != nil {
		t.Fatal(err)
	}
	text := addr.String()
	for _, entry := range addressEntryPoints() {
		if err := entry.parse(text); err == nil {
			t.Errorf("%s accepted trailing bytes in pointer address %q",
				entry.name, text)
		}
	}
}

// TestAddressEntryPointsRejectUnknownAddressType covers the header type
// nibbles 9-13, which Cardano does not define.
func TestAddressEntryPointsRejectUnknownAddressType(t *testing.T) {
	for addrType := uint8(9); addrType <= 13; addrType++ {
		raw := make([]byte, 1+2*common.AddressHashSize)
		raw[0] = (addrType << 4) | common.AddressNetworkMainnet
		for i := range common.AddressHashSize {
			raw[1+i] = 0xaa
			raw[1+common.AddressHashSize+i] = 0xbb
		}
		addr, err := common.NewAddressFromBytes(raw)
		if err != nil {
			t.Fatal(err)
		}
		text := addr.String()
		for _, entry := range addressEntryPoints() {
			if err := entry.parse(text); err == nil {
				t.Errorf("%s accepted unknown address type %d in %q",
					entry.name, addrType, text)
			}
		}
	}
}

// TestAddressEntryPointsRejectUnknownNetworkId covers network nibbles 2-15,
// which fit in the header byte but do not exist on Cardano.
func TestAddressEntryPointsRejectUnknownNetworkId(t *testing.T) {
	for network := uint8(2); network < 16; network++ {
		raw := make([]byte, 1+2*common.AddressHashSize)
		raw[0] = (common.AddressTypeKeyKey << 4) | network
		for i := range common.AddressHashSize {
			raw[1+i] = 0xaa
			raw[1+common.AddressHashSize+i] = 0xbb
		}
		addr, err := common.NewAddressFromBytes(raw)
		if err != nil {
			t.Fatal(err)
		}
		text := addr.String()
		for _, entry := range addressEntryPoints() {
			if err := entry.parse(text); err == nil {
				t.Errorf("%s accepted network id %d in %q",
					entry.name, network, text)
			}
		}
	}
}

// TestAddressEntryPointsRejectMalformedText covers inputs that are not
// addresses at all. None of them may panic.
func TestAddressEntryPointsRejectMalformedText(t *testing.T) {
	valid := mainnetSweepAddress(t)
	malformed := []string{
		"",
		" ",
		"addr1",
		"addr_test1",
		"stake1",
		"not-a-valid-address",
		"invalid",
		" " + valid,
		valid + " ",
		strings.ToUpper(valid[:5]) + valid[5:], // mixed case
		"addr_test1" + valid[5:],               // testnet prefix, mainnet body
		"addr1" + valid[10:],                   // mainnet prefix, short body
		"DdzFF",
	}
	for _, entry := range addressEntryPoints() {
		for _, text := range malformed {
			if err := entry.parse(text); err == nil {
				t.Errorf("%s accepted malformed address %q", entry.name, text)
			}
		}
	}
}

// TestAddressEntryPointsAcceptUppercaseBech32 locks in existing behavior:
// bech32 permits an all-uppercase encoding, and it decodes to the same
// address, so it must keep working.
func TestAddressEntryPointsAcceptUppercaseBech32(t *testing.T) {
	upper := strings.ToUpper(mainnetSweepAddress(t))
	for _, entry := range addressEntryPoints() {
		if err := entry.parse(upper); err != nil {
			t.Errorf("%s rejected uppercase bech32 %q: %v",
				entry.name, upper, err)
		}
	}
}
