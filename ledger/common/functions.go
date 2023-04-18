package common

import (
	"encoding/hex"

	"golang.org/x/crypto/blake2b"
)

func (h Hash) String() string {
	return hex.EncodeToString([]byte(h[:]))
}

func GenerateBlockHeaderHash(data []byte, prefix []byte) string {
	tmpHash, _ := blake2b.New256(nil)
	if prefix != nil {
		tmpHash.Write(prefix)
	}
	tmpHash.Write(data)
	return hex.EncodeToString(tmpHash.Sum(nil))
}
