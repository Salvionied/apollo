package apollo

import (
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// compatibilityWallet deliberately implements only Wallet. This compile-time
// assertion prevents optional evaluation signing from changing that interface.
type compatibilityWallet struct{}

func (compatibilityWallet) Address() common.Address { return common.Address{} }
func (compatibilityWallet) SignTxBody(common.Blake2b256) (common.VkeyWitness, error) {
	return common.VkeyWitness{}, nil
}
func (compatibilityWallet) PubKeyHash() common.Blake2b224      { return common.Blake2b224{} }
func (compatibilityWallet) StakePubKeyHash() common.Blake2b224 { return common.Blake2b224{} }

func TestCustomWalletSatisfiesUnchangedWalletInterface(t *testing.T) {
	var _ Wallet = compatibilityWallet{}
}
