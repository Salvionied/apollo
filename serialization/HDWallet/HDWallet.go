package HDWallet

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/Salvionied/apollo/crypto/bip32"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/text/unicode/norm"
)

type HDWallet struct {
	RootXprivKey bip32.XPrv
	XPrivKey     bip32.XPrv
	Path         string
	Seed         []byte
	Mnemonic     string
	Passphrase   string
	Entropy      string
}

func tweak_bits(seed []byte) []byte {
	seed[0] &= 0b11111000
	seed[31] &= 0b00011111
	seed[31] |= 0b01000000
	return seed
}


/**
	NewHDWalletFromSeed creates a new HDWallet instance from a seed string.

	Params:
		seed (string): The input seed string.
	
	Returns:
		*HDWallet: A new HDWallet instance.
		error: An error if the new wallet fails.
*/
func NewHDWalletFromSeed(seed string) (*HDWallet, error) {
	seed_converted, err := hex.DecodeString(seed)
	if err != nil {
		return nil, err
	}
	seed_modified := tweak_bits(seed_converted)
	privKey, err := bip32.NewXPrv(seed_modified)
	if err != nil {
		return nil, err
	}
	return &HDWallet{
		RootXprivKey: privKey,
		XPrivKey:     privKey,
		Path:         "m",
		Seed:         seed_converted,
		Mnemonic:     "",
		Passphrase:   "",
		Entropy:      "",
	}, nil

}

/**
	GenerateSeed generates a seed string from a mnemonic and passphrase.

	Params:
		mnemonic (string): The mnemonic for seed generation. 
		passphrase (string): The passphrase for seed generation.

	Returns:
		string: The seed string in hexadecimal format.
		error: An error if the generation seed fails.
*/
func GenerateSeed(mnemonic string, passphrase string) (string, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, passphrase)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(seed), nil
}

func generateSeedFromEntropy(passphrase string, entropy []byte) string {
	res := pbkdf2.Key([]byte(passphrase), entropy, 4096, 96, sha512.New)
	return hex.EncodeToString(res)
}


/**
	NewHDWalletFromMnemonic creates a new HDWallet instance from a
	mnemonic and passphrase.

	Params:
		mnemonic (string): The mnemonic for wallet generation. 
		passphrase (string): The passphrase for wallet generation.
	
	Returns:
		*HDWallet: A new HDWallet instance.
		error: An error if the new wallet fails.
*/

func NewHDWalletFromMnemonic(mnemonic string, passphrase string) (*HDWallet, error) {
	mnemo := norm.NFKD.String(mnemonic)
	entropy, error := bip39.EntropyFromMnemonic(mnemonic)
	if error != nil {
		return nil, error
	}
	if !bip39.IsMnemonicValid(mnemo) {
		return nil, error
	}
	seed := generateSeedFromEntropy(passphrase, entropy)
	wallet, err := NewHDWalletFromSeed(seed)
	if err != nil {
		return nil, err
	}
	wallet.Seed = []byte(seed)
	wallet.Mnemonic = mnemonic
	wallet.Passphrase = passphrase

	wallet.Entropy = hex.EncodeToString(entropy)
	return wallet, nil
}

func (hd *HDWallet) copy() *HDWallet {
	return &HDWallet{
		RootXprivKey: hd.RootXprivKey,
		XPrivKey:     hd.XPrivKey,
		Path:         hd.Path,
		Seed:         hd.Seed,
		Mnemonic:     hd.Mnemonic,
		Passphrase:   hd.Passphrase,
		Entropy:      hd.Entropy,
	}
}


/**
	DerivePath derives a new HDWallet from the current wallet based on 
	the path.

	Params:
		path (string): The derivation path in the format "m/".

	Returns:
		*HDWallet: A new HDWallet derived based on the path.
		error: An error if the derived path fails.
*/
func (hd *HDWallet) DerivePath(path string) (*HDWallet, error) {
	if path[:2] != "m/" {
		return nil, errors.New("Invalid path")
	}
	derived_wallet := hd.copy()
	for _, index := range strings.Split(strings.TrimLeft(path, "m/"), "/") {
		if strings.HasSuffix(index, "'") {
			ind_val, err := strconv.Atoi(string(index[:len(index)-1]))

			if err != nil {
				return nil, err
			}
			derived_wallet = derived_wallet.Derive(
				uint32(ind_val), true,
			)
		} else {
			ind_val, err := strconv.Atoi(index)
			if err != nil {
				return nil, err
			}
			derived_wallet = derived_wallet.Derive(
				uint32(ind_val), false,
			)
		}
	}
	return derived_wallet, nil
}

/**
	Derive function derives a new HDWallet from the current wallet
	based on an index and a flag.

	Params:
		index (uint32): The index for derivation.
		hardened (bool): A flag indicating whether to perform a hardened derivation.

	Returns:
		*HDWallet: A new HDWallet derived based on the index and hardening flag.
*/
func (hd *HDWallet) Derive(index uint32, hardened bool) *HDWallet {
	if hardened {
		index += 1 << 31
	}
	derived_xprivkey := hd.XPrivKey.Derive(index)
	return &HDWallet{
		RootXprivKey: hd.RootXprivKey,
		XPrivKey:     derived_xprivkey,
		Path:         hd.Path + "/" + strconv.Itoa(int(index)),
		Seed:         hd.Seed,
		Mnemonic:     hd.Mnemonic,
		Passphrase:   hd.Passphrase,
		Entropy:      hd.Entropy,
	}

}


/**
	GenerateMnemonic function generate a random mnemonic.

	Returns:
		string: A random mnemonic phrase.
		error: An error if the generation of mnemonic fails.
*/
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemo, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemo, nil

}

/**
	IsMnemonic checks if a given mnemonic pgrase is valid.

	Params:
		mnemonic (string): The mnemonic phrase to validate.

	Returns:
		bool: true if the mnemonic is valid, false otherwise.
*/
func IsMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}
