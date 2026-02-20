package Address

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Salvionied/apollo/constants"
	"github.com/Salvionied/apollo/crypto/bech32"
	"github.com/Salvionied/apollo/serialization"
	"github.com/fxamacker/cbor/v2"
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

func WalletAddressFromBytes(
	payment []byte,
	staking []byte,
	network constants.Network,
) *Address {
	var addr Address
	addr.PaymentPart = payment
	addr.StakingPart = staking
	if network == constants.MAINNET {
		addr.Network = MAINNET
	} else {
		addr.Network = TESTNET
	}
	if len(payment) == 0 {
		return nil
	} else if len(staking) == 0 {
		addr.AddressType = KEY_NONE
	} else {
		addr.AddressType = KEY_KEY
	}
	addr.HeaderByte = (addr.AddressType << 4) | addr.Network
	addr.Hrp = ComputeHrp(addr.AddressType, addr.Network)
	return &addr
}

/**
This function check if the current address is equal to another address.

Params:
	addr (*Address): A pointer to the current address.
	other (*Address): A pointer to the other address for comparison.

Returns:
	bool: true if the addresses are equal, false otherwise.
*/

func (addr *Address) Equal(other *Address) bool {
	return addr.String() == other.String()
}

/*
*

	Debug method returns a formatted string representation of the address for debugging

	Returns:
		string: A formatted debug string representing the address.
*/
func (addr *Address) Debug() string {
	return fmt.Sprintf(
		"{\nPaymentPart: %v\nStakingPart: %v\nNetwork: %v\nAddressType: %v\nHeaderByte: %v\nHrp: %s\n}",
		addr.PaymentPart,
		addr.StakingPart,
		addr.Network,
		addr.AddressType,
		addr.HeaderByte,
		addr.Hrp,
	)
}

/*
*

	It converts an address to its CBOR (Concise Binary Object Representation) format and returns


	it as a hexadecimal string. This function marshals the address into its binary representation


	using the CBOR encoding. In case of success, it returns the binary data encoded as a
	hexadecimal string, otherwise a fatal error is logged.

	Returns:


	string: A hexadecimal string representation of the address in CBOR format.
	error: An error if the conversion fails.
*/
func (addr *Address) ToCbor() (string, error) {
	b, err := cbor.Marshal(addr.Bytes())
	if err != nil {
		return "", fmt.Errorf("error marshalling address to cbor, %w", err)
	}
	return hex.EncodeToString(b), nil
}

/*
*

		MarshalCBOR encodes an address to its CBOR (Concise Binary Object Representation) format.

		Returns:
		   	[]byte: A slice of bytes representing the address in CBOR format.
	  		error: An error, if any, encountered during the encoding process.
*/
func (addr *Address) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(addr.Bytes())
}

/*
*

	UnmarshalCBOR decodes a CBOR (Concise Binary Object Representation) encoded address from a byte slice.

	Params:
		value ([]byte): A byte slice containing the CBOR-encoded address.

	Returns:
		error: An error, if any, encountered during the decoding process.
*/
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

/*
*

	This function returns the binary representation of the address. It
	constructs and returns the binary representation of the address containing
	the header byte, payment part, and staking part (if present).

	Returns:
		[]byte: A byte slice representing the binary data of the address.
*/
func (addr Address) Bytes() []byte {
	var payment []byte
	var staking []byte
	payment = addr.PaymentPart
	if len(addr.StakingPart) == 28 {
		staking = addr.StakingPart
	} else {
		staking = make([]byte, 0)
	}
	result := make([]byte, 0, 1+len(payment)+len(staking))
	result = append(result, addr.HeaderByte)
	result = append(result, payment...)
	return append(result, staking...)

}

/*
*

	This function returns the string representation of the address.

	Returns:
		string: A string representing the address in Bech32 format.
*/
func (addr Address) String() string {
	byteaddress, err := bech32.ConvertBits(addr.Bytes(), 8, 5, true)
	if err != nil {
		return ""
	}
	result, _ := bech32.Encode(addr.Hrp, byteaddress)
	return result
}

/*
*

	ComputeHrp computes the human-readable part (Hrp) for an address
	based on its address type and network.

	Params:
	 	address_type (uint8): The type of the address.
		network (uint8): The network identifier (1 for mainnet, 0 for testnet).

	Returns:
		string: The computed Hrp for address encoding.
*/
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

/*
*

	This function decodes a string representation of an address into its corresponding Address structure.

	Parameters:
		value (string): The string representation of the address to decode.

	Returns:
		Address: The decoded Address structure.
		error: An error, if any, encountered during the decoding process.
*/
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
	if network != 0b0000 && network != 0b0001 {
		return Address{}, errors.New("invalid network tag")
	}
	switch addr_type {
	case KEY_KEY:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			payload[serialization.VERIFICATION_KEY_HASH_SIZE:],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case SCRIPT_KEY:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			payload[serialization.VERIFICATION_KEY_HASH_SIZE:],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case KEY_SCRIPT:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			payload[serialization.VERIFICATION_KEY_HASH_SIZE:],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case KEY_NONE:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			make([]byte, 0),
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case SCRIPT_SCRIPT:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			payload[serialization.VERIFICATION_KEY_HASH_SIZE:],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case SCRIPT_NONE:
		return Address{
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			make([]byte, 0),
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	case NONE_KEY:
		return Address{
			make([]byte, 0),
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	default:
		return Address{
			make([]byte, 0),
			payload[:serialization.VERIFICATION_KEY_HASH_SIZE],
			network,
			addr_type,
			header,
			ComputeHrp(addr_type, network),
		}, nil
	}
}
