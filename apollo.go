package apollo

import (
	"github.com/salvionied/apollo/apollotypes"
	"github.com/salvionied/apollo/serialization"
	"github.com/salvionied/apollo/serialization/Address"
	"github.com/salvionied/apollo/serialization/HDWallet"
	"github.com/salvionied/apollo/serialization/Key"
	"github.com/salvionied/apollo/serialization/Metadata"
	"github.com/salvionied/apollo/serialization/PlutusData"
	"github.com/salvionied/apollo/serialization/UTxO"
	"github.com/salvionied/apollo/txBuilding/Backend/BlockFrostChainContext"
)

type Apollo struct {
	wallet  apollotypes.Wallet
	backend apollotypes.Backend
	network Network
}

func (a *Apollo) SwitchBackend(backend apollotypes.Backend) *Apollo {
	a.backend = backend
	return a
}

func (a *Apollo) SwitchNetwork(network Network) *Apollo {
	a.network = network
	return a
}

func (a *Apollo) NewTx(utils any) *builder {
	return &builder{Apollo: a}
}

func (a *Apollo) FromTx(tx any) *builder {
	return &builder{Apollo: a}
}

func (a *Apollo) UtxosAt(address Address.Address) []UTxO.UTxO {
	return a.backend.Utxos(address)
}

func (a *Apollo) UtxosByAsset(asset string) []UTxO.UTxO {
	return nil
}

func (a *Apollo) CheckTx(txHash serialization.TransactionId, checkInterval int) chan bool {
	//TODO
	return nil
}

func (a *Apollo) DatumOf(utxo UTxO.UTxO) *PlutusData.PlutusData {
	//TODO
	return nil
}

func (a *Apollo) MetadataOf(asset string) *Metadata.Metadata {
	//TODO
	return nil
}

func (a *Apollo) SetWalletFromPrivateKeyHex(privateKey string) *Apollo {
	return a
}

func (a *Apollo) SetWalletFromPrivateKeyBytes(privateKey []byte) *Apollo {
	return a
}

func (a *Apollo) SetWalletFromMnemonic(mnemonic string) *Apollo {
	paymentPath := "m/1852'/1815'/0'/0/0"
	stakingPath := "m/1852'/1815'/0'/2/0"
	hdWall := HDWallet.NewHDWalletFromMnemonic(mnemonic, "")
	paymentKeyPath := hdWall.DerivePath(paymentPath)
	verificationKey_bytes := paymentKeyPath.XPrivKey.PublicKey()
	signingKey_bytes := paymentKeyPath.XPrivKey.Bytes()
	stakingKeyPath := hdWall.DerivePath(stakingPath)
	stakeVerificationKey_bytes := stakingKeyPath.XPrivKey.PublicKey()
	stakeSigningKey_bytes := stakingKeyPath.XPrivKey.Bytes()
	//stake := stakingKeyPath.RootXprivKey.Bytes()
	signingKey := Key.SigningKey{Payload: signingKey_bytes}
	verificationKey := Key.VerificationKey{Payload: verificationKey_bytes}
	stakeSigningKey := Key.StakeSigningKey{Payload: stakeSigningKey_bytes}
	stakeVerificationKey := Key.StakeVerificationKey{Payload: stakeVerificationKey_bytes}
	stakeVerKey := Key.VerificationKey{Payload: stakeVerificationKey_bytes}
	skh, _ := stakeVerKey.Hash()
	vkh, _ := verificationKey.Hash()

	addr := Address.Address{StakingPart: skh[:], PaymentPart: vkh[:], Network: 1, AddressType: Address.KEY_KEY, HeaderByte: 0b00000001, Hrp: "addr"}
	wallet := apollotypes.GenericWallet{SigningKey: signingKey, VerificationKey: verificationKey, Address: addr, StakeSigningKey: stakeSigningKey, StakeVerificationKey: stakeVerificationKey}
	a.wallet = &wallet
	return a

}

func New(cc apollotypes.Backend, network Network) *Apollo {
	return &Apollo{
		backend: cc,
		network: network,
	}
}

func NewBlockfrostBackend(projectId string, network Network) apollotypes.Backend {
	switch network {
	case MAINNET:
		bfc := BlockFrostChainContext.NewBlockfrostChainContext(projectId, int(MAINNET), BLOCKFROST_BASE_URL_MAINNET)
		return &bfc
	case TESTNET:
		bfc := BlockFrostChainContext.NewBlockfrostChainContext(projectId, int(TESTNET), BLOCKFROST_BASE_URL_TESTNET)
		return &bfc
	case PREVIEW:
		bfc := BlockFrostChainContext.NewBlockfrostChainContext(projectId, int(TESTNET), BLOCKFROST_BASE_URL_PREVIEW)
		return &bfc
	case PREPROD:
		bfc := BlockFrostChainContext.NewBlockfrostChainContext(projectId, int(TESTNET), BLOCKFROST_BASE_URL_PREPROD)
		return &bfc
	}
	return nil
}
