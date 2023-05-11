package byron

import (
	"github.com/salvionied/apollo/ledger/common"

	"github.com/Salvionied/cbor/v2"
)

type ByronEbbBlock struct {
	_        struct{} `cbor:",toarray"`
	Id       uint
	EbbBlock EbbBlock
}

func (ebb *ByronEbbBlock) Hash() string {
	return ebb.EbbBlock.Header.Hash()
}

type ByronMainBlock struct {
	_         struct{} `cbor:",toarray"`
	Id        uint
	MainBlock MainBlock
}

func (main *ByronMainBlock) Hash() string {
	return main.MainBlock.Header.Hash()
}

type MainBlock struct {
	_      struct{}    `cbor:",toarray"`
	Header BlockHead   `cbor:"header"`
	Body   BlockBody   `cbor:"body"`
	Extra  interface{} `cbor:"extra"`
}

type EbbBlock struct {
	Header EbbHead `cbor:"header"`
}

type BlockHead struct {
	_             struct{}       `cbor:",toarray"`
	ProtocolMagic uint32         `cbor:"protocolMagic"`
	PrevBlock     common.BlockId `cbor:"prevBlock"`
	BodyProof     BlockProof     `cbor:"bodyProof"`
	ConsensusData BlockCons      `cbor:"consensusData"`
	ExtraData     BlockHeadEx    `cbor:"extraData"`
}

func (bh *BlockHead) Hash() string {
	marshaled, _ := cbor.Marshal(bh)
	return common.GenerateBlockHeaderHash(marshaled, []byte{0x82, common.BLOCK_TYPE_BYRON_MAIN})
}

type SlotId struct {
	_     struct{} `cbor:",toarray"`
	Epoch common.EpochId
	Slot  uint64
}

type BlockSig struct {
	_  struct{} `cbor:",toarray"`
	Id uint8
	//TODO: Implement this
	// blocksig = [0, signature]
	//      / [1, lwdlgsig]
	//      / [2, dlgsig]
	Val interface{}
}

type BlockCons struct {
	_          struct{} `cbor:",toarray"`
	SlotId     SlotId   `cbor:"slotId"`
	Pubkey     common.PubKey
	Difficulty []uint64
	BlockSig   BlockSig
}

type BlockBody struct {
	_          struct{}        `cbor:",toarray"`
	TxPayload  []TxWithWitness `cbor:"txPayload"`
	SscPayload Ssc             `cbor:"sscPayload"`
	DlgPayload Dlg             `cbor:"dlgPayload,omitempty"`
	UpdPayload Upd             `cbor:"updPayload"`
}

type EbbHead struct {
	ProtocolMagic uint32         `cbor:"protocolMagic"`
	PrevBlock     common.BlockId `cbor:"prevBlock"`
	BodyProof     common.Hash    `cbor:"bodyProof"`
	ExtraData     interface{}    `cbor:"extraData"`
}

func (eh *EbbHead) Hash() string {
	marshaled, _ := cbor.Marshal(eh)
	return common.GenerateBlockHeaderHash(marshaled, []byte{0x82, common.BLOCK_TYPE_BYRON_EBB})
}

type BlockProof struct {
	_        struct{}    `cbor:",toarray"`
	TxProof  TxProof     `cbor:"txProof"`
	SscProof SscProof    `cbor:"sscProof"`
	DlgProof common.Hash `cbor:"dlgProof"`
	UpdProof common.Hash `cbor:"updProof"`
}

type Bver struct {
	_     struct{} `cbor:",toarray"`
	Major uint16
	Minor uint16
	Last  uint8
}

type SoftwareVersion struct {
	_    struct{} `cbor:",toarray"`
	Text string
	Num  uint32
}

type BlockHeadEx struct {
	_               struct{}        `cbor:",toarray"`
	BlockVersion    Bver            `cbor:"blockVersion"`
	SoftwareVersion SoftwareVersion `cbor:"softwareVersion"`
	Attributes      interface{}     `cbor:"attributes"`
	ExtraProof      common.Hash     `cbor:"extraProof"`
}

type SscProof [3]any

type TxProof struct {
	_  struct{} `cbor:",toarray"`
	Id uint8
	F  common.Hash
	FF common.Hash
}

type Ssc interface{}

//TODO Implement Shared See Computation

type Dlg interface{}

// type Dlg struct {
// 	_           struct{}         `cbor:",toarray"`
// 	Epoch       common.EpochId   `cbor:"epoch"`
// 	Issuer      common.PubKey    `cbor:"issuer"`
// 	Delegate    common.PubKey    `cbor:"delegate"`
// 	Certificate common.Signature `cbor:"certificate"`
// }

type TxFeePool struct {
	Val [2]uint64 `cbor:"0,keyasint"`
}

type SoftForkRule struct {
	_   struct{} `cbor:",toarray"`
	F   uint64
	FF  uint64
	FFF uint64
}

type BverMod struct {
	ScriptVersion     []uint16         `cbor:"scriptVersion,omitempty"`
	SlotDuration      []uint64         `cbor:"slotDuration,omitempty"`
	MaxBlockSize      []uint64         `cbor:"maxBlockSize,omitempty"`
	MaxHeaderSize     []uint64         `cbor:"maxHeaderSize,omitempty"`
	MaxTxSize         []uint64         `cbor:"maxTxSize,omitempty"`
	MaxProposalSize   []uint64         `cbor:"maxProposalSize,omitempty"`
	MpcThd            []uint64         `cbor:"mpcThd,omitempty"`
	HeavyDelThd       []uint64         `cbor:"heavyDelThd,omitempty"`
	UpdateVoteThd     []uint64         `cbor:"updateVoteThd,omitempty"`
	UpdateProposalThd []uint64         `cbor:"updateProposalThd,omitempty"`
	UpdateImplicit    []uint64         `cbor:"updateImplicit,omitempty"`
	SoftforkRule      []SoftForkRule   `cbor:"softforkRule,omitempty"`
	TxFeePolicy       []TxFeePool      `cbor:"txFeePolicy,omitempty"`
	UnlockStakeEpoch  []common.EpochId `cbor:"unlockStakeEpoch,omitempty"`
}

type UpProp struct {
	BlockVersion    Bver             `cbor:"blockVersion"`
	BlockVersionMod BverMod          `cbor:"blockVersionMod"`
	SoftwareVersion SoftwareVersion  `cbor:"softwareVersion"`
	Data            interface{}      //TOOD Implement Data
	Attributes      interface{}      `cbor:"attributes"`
	From            common.PubKey    `cbor:"from"`
	Signature       common.Signature `cbor:"signature"`
}

type UpVote struct {
	Voter      common.PubKey    `cbor:"voter"`
	ProposalId common.Updid     `cbor:"proposalId"`
	Vote       bool             `cbor:"vote"`
	Signature  common.Signature `cbor:"signature"`
}
type Upd struct {
	_        struct{} `cbor:",toarray"`
	Proposal []UpProp `cbor:"proposal,omitempty"`
	Votes    []UpVote `cbor:"votes"`
}
