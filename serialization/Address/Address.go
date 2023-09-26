package Address

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Salvionied/apollo/crypto/bech32"
	"github.com/Salvionied/apollo/serialization"

	"github.com/Salvionied/cbor/v2"
)

const (
	BYRON          = 0b1000
	KEY_KEY        = 0b0000
	SCRIPT_KEY     = 0b0001
	KEY_SCRIPT     = 0b0010
	SCRIPT_SCRIPT  = 0b0011
	KEY_POINTER    = 0b0100
	SCRIPT_POINTER = 0b0101
	KEY_NONE       = 0b0110
	SCRIPT_NONE    = 0b0111
	NONE_KEY       = 0b1110
	NONE_SCRIPT    = 0b1111
)
const (
	MAINNET = 1
	TESTNET = 0
)

type Address struct {
	PaymentPart []byte
	StakingPart []byte
	Network     byte
	AddressType byte
	HeaderByte  byte
	Hrp         string
}

func (addr *Address) Equal(other *Address) bool {
	return addr.String() == other.String()
}

func (addr *Address) Debug() string {
	return fmt.Sprintf("{\nPaymentPart: %v\nStakingPart: %v\nNetwork: %v\nAddressType: %v\nHeaderByte: %v\nHrp: %s\n}", addr.PaymentPart, addr.StakingPart, addr.Network, addr.AddressType, addr.HeaderByte, addr.Hrp)
}

func (addr *Address) ToCbor() (string, error) {
	b, err := cbor.Marshal(addr.Bytes())
	if err != nil {
		return "", fmt.Errorf("error marshalling address to cbor, %s", err)
	}
	return hex.EncodeToString(b), nil
}
func (addr *Address) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(addr.Bytes())
}
func (addr *Address) UnmarshalCBOR(value []byte) error {
	res := make([]byte, 0)
	err := cbor.Unmarshal(value, &res)
	header := res[0]
	payload := res[1:]
	addr.PaymentPart = payload[:serialization.VERIFICATION_KEY_HASH_SIZE]
	addr.StakingPart = payload[serialization.VERIFICATION_KEY_HASH_SIZE:]
	addr.Network = (header & 0x0F)
	addr.AddressType = (header & 0xF0) >> 4
	addr.HeaderByte = header
	addr.Hrp = ComputeHrp(addr.AddressType, addr.Network)
	return err
}

func (addr Address) Bytes() []byte {
	var payment []byte
	var staking []byte
	payment = addr.PaymentPart
	if len(addr.StakingPart) == 28 {
		staking = addr.StakingPart
	} else {
		staking = make([]byte, 0)
	}
	result := make([]byte, 0)
	result = append(result, addr.HeaderByte)
	result = append(result, payment...)
	return append(result, staking...)

}

func (addr Address) String() string {
	byteaddress, err := bech32.ConvertBits(addr.Bytes(), 8, 5, true)
	if err != nil {
		return ""
	}
	result, _ := bech32.Encode(addr.Hrp, byteaddress)
	return result
}

func ComputeHrp(address_type uint8, network uint8) string {
	var prefix string
	if address_type == NONE_KEY || address_type == NONE_SCRIPT {
		prefix = "stake"
	} else {
		prefix = "addr"
	}
	var suffix string
	if network == 1 {
		suffix = ""
	} else {
		suffix = "_test"
	}
	return prefix + suffix

}

func DecodeAddress(value string) (Address, error) {
	_, data, err := bech32.Decode(value)
	if err != nil {
		return Address{}, err
	}

	decoded_value, _ := bech32.ConvertBits(data, 5, 8, false)

	header := decoded_value[0]
	payload := decoded_value[1:]
	network := (header & 0x0F)
	addr_type := (header & 0xF0) >> 4
	if !(network == 0b0000 || network == 0b0001) {
		return Address{}, errors.New("invalid network tag")
	}
	if addr_type == KEY_KEY {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], payload[serialization.VERIFICATION_KEY_HASH_SIZE:], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == SCRIPT_KEY {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], payload[serialization.VERIFICATION_KEY_HASH_SIZE:], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == KEY_SCRIPT {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], payload[serialization.VERIFICATION_KEY_HASH_SIZE:], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == KEY_NONE {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], make([]byte, 0), network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == SCRIPT_SCRIPT {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], payload[serialization.VERIFICATION_KEY_HASH_SIZE:], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == SCRIPT_NONE {
		return Address{payload[:serialization.VERIFICATION_KEY_HASH_SIZE], make([]byte, 0), network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else if addr_type == NONE_KEY {
		return Address{make([]byte, 0), payload[:serialization.VERIFICATION_KEY_HASH_SIZE], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	} else {
		return Address{make([]byte, 0), payload[:serialization.VERIFICATION_KEY_HASH_SIZE], network, addr_type, header, ComputeHrp(addr_type, network)}, nil
	}
}
