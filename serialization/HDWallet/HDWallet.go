package HDWallet

import (
	"crypto/sha512"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/SundaeSwap-finance/apollo/crypto/bip32"

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

func NewHDWalletFromSeed(seed string) *HDWallet {
	seed_converted, err := hex.DecodeString(seed)
	if err != nil {
		panic(err)
	}
	seed_modified := tweak_bits(seed_converted)
	privKey, err := bip32.NewXPrv(seed_modified)
	if err != nil {
		panic(err)
	}
	return &HDWallet{
		RootXprivKey: privKey,
		XPrivKey:     privKey,
		Path:         "m",
		Seed:         seed_converted,
		Mnemonic:     "",
		Passphrase:   "",
		Entropy:      "",
	}

}

func GenerateSeed(mnemonic string, passphrase string) string {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, passphrase)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(seed)
}

func generateSeedFromEntropy(passphrase string, entropy []byte) string {
	res := pbkdf2.Key([]byte(passphrase), entropy, 4096, 96, sha512.New)
	return hex.EncodeToString(res)
}

func NewHDWalletFromMnemonic(mnemonic string, passphrase string) *HDWallet {
	mnemo := norm.NFKD.String(mnemonic)
	entropy, error := bip39.EntropyFromMnemonic(mnemonic)
	if error != nil {
		panic(error)
	}
	if !bip39.IsMnemonicValid(mnemo) {
		panic("Invalid Mnemonic")
	}
	seed := generateSeedFromEntropy(passphrase, entropy)
	wallet := NewHDWalletFromSeed(seed)
	wallet.Seed = []byte(seed)
	wallet.Mnemonic = mnemonic
	wallet.Passphrase = passphrase

	wallet.Entropy = hex.EncodeToString(entropy)
	return wallet
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

func (hd *HDWallet) DerivePath(path string) *HDWallet {
	if path[:2] != "m/" {
		panic("Invalid path")
	}
	derived_wallet := hd.copy()
	for _, index := range strings.Split(strings.TrimLeft(path, "m/"), "/") {
		if strings.HasSuffix(index, "'") {
			ind_val, err := strconv.Atoi(string(index[:len(index)-1]))

			if err != nil {
				panic(err)
			}
			derived_wallet = derived_wallet.Derive(
				uint32(ind_val), true,
			)
		} else {
			ind_val, err := strconv.Atoi(index)
			if err != nil {
				panic(err)
			}
			derived_wallet = derived_wallet.Derive(
				uint32(ind_val), false,
			)
		}
	}
	return derived_wallet
}

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

func GenerateMnemonic() string {
	entropy, error := bip39.NewEntropy(256)
	if error != nil {
		panic(error)
	}
	mnemo, err := bip39.NewMnemonic(entropy)
	if err != nil {
		panic(err)
	}
	return mnemo

}

func IsMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}
