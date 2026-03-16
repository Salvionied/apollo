package Governance

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/fxamacker/cbor/v2"
)

// ----------------------------------------------------------------
// Vote
// ----------------------------------------------------------------

// Vote represents a governance vote value.
type Vote int

const (
	VoteNo      Vote = 0
	VoteYes     Vote = 1
	VoteAbstain Vote = 2
)

// ----------------------------------------------------------------
// VoterRole
// ----------------------------------------------------------------

// VoterRole identifies the role of a voter.
type VoterRole int

const (
	ConstitutionalCommitteeKeyHash VoterRole = 0
	ConstitutionalCommitteeScript  VoterRole = 1
	DRepKeyHash                    VoterRole = 2
	DRepScript                     VoterRole = 3
	StakePoolOperator              VoterRole = 4
)

// ----------------------------------------------------------------
// Voter  -- CBOR array [role, hash]
// ----------------------------------------------------------------

// Voter identifies who is casting a vote.
type Voter struct {
	_    struct{} `cbor:",toarray"`
	Role VoterRole
	Hash serialization.ConstrainedBytes
}

// ----------------------------------------------------------------
// GovActionId  -- CBOR array [tx_hash, index]
// ----------------------------------------------------------------

// GovActionId references a specific governance action.
type GovActionId struct {
	_               struct{} `cbor:",toarray"`
	TransactionHash []byte
	GovActionIndex  uint32
}

// ----------------------------------------------------------------
// VotingProcedure  -- CBOR array [vote, anchor / null]
// ----------------------------------------------------------------

// VotingProcedure captures a single vote plus optional anchor.
type VotingProcedure struct {
	_      struct{} `cbor:",toarray"`
	Vote   Vote
	Anchor *Certificate.Anchor
}

// ----------------------------------------------------------------
// VotingProcedures  -- nested CBOR map
//
//	{ voter => { action_id => procedure } }
//
// ----------------------------------------------------------------

// ActionVote pairs an action with its voting procedure.
type ActionVote struct {
	ActionId  GovActionId
	Procedure VotingProcedure
}

// VoterVotes groups all votes cast by a single voter.
type VoterVotes struct {
	Voter Voter
	Votes []ActionVote
}

// VotingProcedures is the top-level collection.
type VotingProcedures []VoterVotes

// Add appends a vote for the given voter and action.
// If the same voter already has a vote for the same action,
// the existing procedure is replaced.
func (vp *VotingProcedures) Add(
	voter Voter,
	actionId GovActionId,
	procedure VotingProcedure,
) {
	for i := range *vp {
		v := &(*vp)[i]
		if voterEqual(v.Voter, voter) {
			for voteIndex := range v.Votes {
				if govActionIdEqual(
					v.Votes[voteIndex].ActionId,
					actionId,
				) {
					v.Votes[voteIndex].Procedure = procedure
					return
				}
			}
			v.Votes = append(
				v.Votes,
				ActionVote{
					ActionId:  actionId,
					Procedure: procedure,
				},
			)
			return
		}
	}
	*vp = append(*vp, VoterVotes{
		Voter: voter,
		Votes: []ActionVote{
			{
				ActionId:  actionId,
				Procedure: procedure,
			},
		},
	})
}

func voterEqual(a, b Voter) bool {
	return a.Role == b.Role &&
		bytes.Equal(a.Hash.Payload, b.Hash.Payload)
}

func govActionIdEqual(a, b GovActionId) bool {
	return a.GovActionIndex == b.GovActionIndex &&
		bytes.Equal(a.TransactionHash, b.TransactionHash)
}

// MarshalCBOR encodes as { voter => { action_id => proc } }.
func (vp VotingProcedures) MarshalCBOR() ([]byte, error) {
	outerRaw := make([][]byte, 0, len(vp)*2)
	for _, vv := range vp {
		voterBz, err := cbor.Marshal(vv.Voter)
		if err != nil {
			return nil, err
		}

		innerRaw := make([][]byte, 0, len(vv.Votes)*2)
		for _, av := range vv.Votes {
			aidBz, err := cbor.Marshal(av.ActionId)
			if err != nil {
				return nil, err
			}
			procBz, err := cbor.Marshal(av.Procedure)
			if err != nil {
				return nil, err
			}
			innerRaw = append(innerRaw, aidBz, procBz)
		}
		innerMap := buildCBORMapRaw(innerRaw)
		outerRaw = append(outerRaw, voterBz, innerMap)
	}
	return buildCBORMapRaw(outerRaw), nil
}

// buildCBORMapRaw builds a definite-length CBOR map from
// alternating key, value byte slices that are already CBOR.
func buildCBORMapRaw(pairs [][]byte) []byte {
	count := len(pairs) / 2
	sortedPairs := make([]rawPair, 0, count)
	for i := 0; i+1 < len(pairs); i += 2 {
		sortedPairs = append(sortedPairs, rawPair{
			key:   pairs[i],
			value: pairs[i+1],
		})
	}
	sort.Slice(sortedPairs, func(i, j int) bool {
		return compareCanonicalCBORKeys(
			sortedPairs[i].key,
			sortedPairs[j].key,
		) < 0
	})
	var buf bytes.Buffer
	writeMapHeader(&buf, count)
	for _, pair := range sortedPairs {
		buf.Write(pair.key)
		buf.Write(pair.value)
	}
	return buf.Bytes()
}

func compareCanonicalCBORKeys(a, b []byte) int {
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return bytes.Compare(a, b)
}

func writeMapHeader(buf *bytes.Buffer, count int) {
	switch {
	case count <= 23:
		buf.WriteByte(0xa0 | byte(count))
	case count <= 255:
		buf.WriteByte(0xb8)
		buf.WriteByte(byte(count))
	case count <= 65535:
		buf.WriteByte(0xb9)
		buf.WriteByte(byte(count >> 8))
		buf.WriteByte(byte(count))
	case uint64(count) <= uint64(^uint32(0)):
		buf.WriteByte(0xba)
		var encoded [4]byte
		binary.BigEndian.PutUint32(
			encoded[:],
			uint32(count),
		)
		buf.Write(encoded[:])
	default:
		buf.WriteByte(0xbb)
		var encoded [8]byte
		binary.BigEndian.PutUint64(
			encoded[:],
			uint64(count),
		)
		buf.Write(encoded[:])
	}
}

// UnmarshalCBOR decodes { voter => { action_id => procedure } }.
func (vp *VotingProcedures) UnmarshalCBOR(
	data []byte,
) error {
	outerPairs, err := decodeMapPairs(data)
	if err != nil {
		return fmt.Errorf("decode outer map: %w", err)
	}

	result := make(VotingProcedures, 0, len(outerPairs))
	for _, op := range outerPairs {
		var voter Voter
		if err := cbor.Unmarshal(op.key, &voter); err != nil {
			return fmt.Errorf("decode voter: %w", err)
		}

		innerPairs, err := decodeMapPairs(op.value)
		if err != nil {
			return fmt.Errorf(
				"decode inner map: %w",
				err,
			)
		}

		votes := make([]ActionVote, 0, len(innerPairs))
		for _, ip := range innerPairs {
			var aid GovActionId
			if err := cbor.Unmarshal(
				ip.key, &aid,
			); err != nil {
				return fmt.Errorf(
					"decode gov_action_id: %w",
					err,
				)
			}
			var proc VotingProcedure
			if err := cbor.Unmarshal(
				ip.value, &proc,
			); err != nil {
				return fmt.Errorf(
					"decode voting_procedure: %w",
					err,
				)
			}
			votes = append(votes, ActionVote{
				ActionId:  aid,
				Procedure: proc,
			})
		}
		result = append(result, VoterVotes{
			Voter: voter,
			Votes: votes,
		})
	}
	*vp = result
	return nil
}

type rawPair struct {
	key   []byte
	value []byte
}

// decodeMapPairs decodes a CBOR map into key/value byte slices.
func decodeMapPairs(data []byte) ([]rawPair, error) {
	header, err := decodeCollectionHeader(data, 5)
	if err != nil {
		return nil, err
	}
	rest := data[header.headerSize:]
	dec := cbor.NewDecoder(bytes.NewReader(rest))
	pairs := make([]rawPair, 0)

	decodePair := func() (rawPair, error) {
		var keyRaw cbor.RawMessage
		if err := dec.Decode(&keyRaw); err != nil {
			return rawPair{}, fmt.Errorf(
				"decode map key: %w",
				err,
			)
		}
		var valRaw cbor.RawMessage
		if err := dec.Decode(&valRaw); err != nil {
			return rawPair{}, fmt.Errorf(
				"decode map value: %w",
				err,
			)
		}
		return rawPair{
			key:   []byte(keyRaw),
			value: []byte(valRaw),
		}, nil
	}

	if header.indefinite {
		for {
			offset := dec.NumBytesRead()
			if offset >= len(rest) {
				return nil, errors.New(
					"indefinite CBOR map missing break",
				)
			}
			if rest[offset] == 0xff {
				if offset+1 != len(rest) {
					return nil, errors.New(
						"extraneous data after CBOR map",
					)
				}
				return pairs, nil
			}

			pair, err := decodePair()
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	}

	for i := uint64(0); i < header.length; i++ {
		pair, err := decodePair()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
	}
	if dec.NumBytesRead() != len(rest) {
		return nil, errors.New("extraneous data after CBOR map")
	}
	return pairs, nil
}

type collectionHeader struct {
	headerSize int
	indefinite bool
	length     uint64
}

func decodeCollectionHeader(
	data []byte,
	expectedMajorType byte,
) (collectionHeader, error) {
	if len(data) == 0 {
		return collectionHeader{}, errors.New("empty CBOR data")
	}

	firstByte := data[0]
	majorType := firstByte >> 5
	if majorType != expectedMajorType {
		return collectionHeader{}, fmt.Errorf(
			"not a CBOR %s",
			collectionTypeName(expectedMajorType),
		)
	}

	additionalInfo := firstByte & 0x1f
	switch {
	case additionalInfo < 24:
		return collectionHeader{
			headerSize: 1,
			length:     uint64(additionalInfo),
		}, nil
	case additionalInfo == 24:
		if len(data) < 2 {
			return collectionHeader{}, io.ErrUnexpectedEOF
		}
		return collectionHeader{
			headerSize: 2,
			length:     uint64(data[1]),
		}, nil
	case additionalInfo == 25:
		if len(data) < 3 {
			return collectionHeader{}, io.ErrUnexpectedEOF
		}
		return collectionHeader{
			headerSize: 3,
			length: uint64(
				binary.BigEndian.Uint16(data[1:3]),
			),
		}, nil
	case additionalInfo == 26:
		if len(data) < 5 {
			return collectionHeader{}, io.ErrUnexpectedEOF
		}
		return collectionHeader{
			headerSize: 5,
			length: uint64(
				binary.BigEndian.Uint32(data[1:5]),
			),
		}, nil
	case additionalInfo == 27:
		if len(data) < 9 {
			return collectionHeader{}, io.ErrUnexpectedEOF
		}
		return collectionHeader{
			headerSize: 9,
			length:     binary.BigEndian.Uint64(data[1:9]),
		}, nil
	case additionalInfo == 31:
		return collectionHeader{
			headerSize: 1,
			indefinite: true,
		}, nil
	default:
		return collectionHeader{}, fmt.Errorf(
			"invalid %s length encoding",
			collectionTypeName(expectedMajorType),
		)
	}
}

func collectionTypeName(majorType byte) string {
	switch majorType {
	case 4:
		return "array"
	case 5:
		return "map"
	default:
		return "collection"
	}
}

// ================================================================
// GovAction -- interface + concrete types
// ================================================================

// GovAction is the interface all governance action types satisfy.
type GovAction interface {
	GovActionType() int
	MarshalCBOR() ([]byte, error)
}

// UnmarshalGovAction decodes CBOR and dispatches to the correct
// concrete type based on the type tag (element 0).
func UnmarshalGovAction(data []byte) (GovAction, error) {
	// Decode as raw items to avoid issues with complex map
	// keys (e.g. credential arrays as map keys).
	items, err := decodeArrayItems(data)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("empty gov_action array")
	}

	var kindVal uint64
	if err := cbor.Unmarshal(
		items[0], &kindVal,
	); err != nil {
		return nil, fmt.Errorf(
			"decode gov_action kind: %w",
			err,
		)
	}
	kind := int(kindVal)

	switch kind {
	case 6:
		if len(items) != 1 {
			return nil, fmt.Errorf(
				"invalid InfoAction length: got %d",
				len(items),
			)
		}
		return InfoAction{}, nil
	case 3:
		return unmarshalNoConfidence(items)
	case 1:
		return unmarshalHardFork(items)
	case 2:
		return unmarshalTreasuryWd(items)
	case 0:
		return unmarshalParamChange(items)
	case 4:
		return unmarshalUpdateCommittee(items)
	case 5:
		return unmarshalNewConstitution(items)
	default:
		return nil, fmt.Errorf(
			"unknown gov_action type: %d",
			kind,
		)
	}
}

func unmarshalNoConfidence(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 2 {
		return nil, errors.New(
			"invalid NoConfidence length",
		)
	}
	prev, err := decodePrevActionRaw(items[1])
	if err != nil {
		return nil, err
	}
	return NoConfidence{PrevActionId: prev}, nil
}

func unmarshalHardFork(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 3 {
		return nil, errors.New(
			"invalid HardForkInitiation length",
		)
	}
	prev, err := decodePrevActionRaw(items[1])
	if err != nil {
		return nil, err
	}
	var pv ProtocolVersion
	if err := cbor.Unmarshal(items[2], &pv); err != nil {
		return nil, err
	}
	return HardForkInitiation{
		PrevActionId:    prev,
		ProtocolVersion: pv,
	}, nil
}

func unmarshalTreasuryWd(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 3 {
		return nil, errors.New(
			"invalid TreasuryWithdrawals length",
		)
	}
	wds, err := decodeWithdrawals(items[1])
	if err != nil {
		return nil, err
	}
	prev, err := decodePrevActionRaw(items[2])
	if err != nil {
		return nil, err
	}
	return TreasuryWithdrawals{
		Withdrawals:  wds,
		PrevActionId: prev,
	}, nil
}

func unmarshalParamChange(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 3 {
		return nil, errors.New(
			"invalid ParameterChange length",
		)
	}
	prev, err := decodePrevActionRaw(items[1])
	if err != nil {
		return nil, err
	}
	var params map[int]any
	if err := cbor.Unmarshal(
		items[2], &params,
	); err != nil {
		return nil, err
	}
	return ParameterChange{
		PrevActionId: prev,
		ParamUpdate:  params,
	}, nil
}

func unmarshalUpdateCommittee(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 5 {
		return nil, errors.New(
			"invalid UpdateCommittee length",
		)
	}
	prev, err := decodePrevActionRaw(items[1])
	if err != nil {
		return nil, err
	}
	var removed []Certificate.Credential
	if err := cbor.Unmarshal(
		items[2], &removed,
	); err != nil {
		return nil, err
	}
	added, err := decodeAddedCC(items[3])
	if err != nil {
		return nil, err
	}
	var quorum Certificate.UnitInterval
	if err := cbor.Unmarshal(
		items[4], &quorum,
	); err != nil {
		return nil, err
	}
	return UpdateCommittee{
		PrevActionId: prev,
		Removed:      removed,
		Added:        added,
		Quorum:       quorum,
	}, nil
}

func unmarshalNewConstitution(
	items [][]byte,
) (GovAction, error) {
	if len(items) != 3 {
		return nil, errors.New(
			"invalid NewConstitution length",
		)
	}
	prev, err := decodePrevActionRaw(items[1])
	if err != nil {
		return nil, err
	}
	// items[2] is a CBOR array [anchor, scripthash/null]
	constitItems, err := decodeArrayItems(items[2])
	if err != nil {
		return nil, errors.New(
			"invalid constitution payload",
		)
	}
	if len(constitItems) != 2 {
		return nil, errors.New(
			"constitution must have 2 elements",
		)
	}
	var anchor Certificate.Anchor
	if err := cbor.Unmarshal(
		constitItems[0], &anchor,
	); err != nil {
		return nil, err
	}
	// Script hash: null or byte string
	var scriptHash []byte
	var isNull bool
	if len(constitItems[1]) == 1 &&
		constitItems[1][0] == 0xf6 {
		isNull = true
	}
	if !isNull {
		if err := cbor.Unmarshal(
			constitItems[1], &scriptHash,
		); err != nil {
			return nil, errors.New(
				"invalid script hash type",
			)
		}
	}
	return NewConstitution{
		PrevActionId: prev,
		Anchor:       anchor,
		ScriptHash:   scriptHash,
	}, nil
}

// decodeArrayItems decodes a CBOR array and returns each
// element as raw CBOR bytes.
func decodeArrayItems(data []byte) ([][]byte, error) {
	header, err := decodeCollectionHeader(data, 4)
	if err != nil {
		return nil, err
	}
	rest := data[header.headerSize:]
	dec := cbor.NewDecoder(bytes.NewReader(rest))
	items := make([][]byte, 0)

	decodeItem := func() ([]byte, error) {
		var raw cbor.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf(
				"decode array item %d: %w",
				len(items),
				err,
			)
		}
		return []byte(raw), nil
	}

	if header.indefinite {
		for {
			offset := dec.NumBytesRead()
			if offset >= len(rest) {
				return nil, errors.New(
					"indefinite CBOR array missing break",
				)
			}
			if rest[offset] == 0xff {
				if offset+1 != len(rest) {
					return nil, errors.New(
						"extraneous data after CBOR array",
					)
				}
				return items, nil
			}

			item, err := decodeItem()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}

	for i := uint64(0); i < header.length; i++ {
		item, err := decodeItem()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if dec.NumBytesRead() != len(rest) {
		return nil, errors.New("extraneous data after CBOR array")
	}
	return items, nil
}

func decodePrevActionRaw(
	data []byte,
) (*GovActionId, error) {
	// Check for CBOR null (0xf6)
	if len(data) == 1 && data[0] == 0xf6 {
		return nil, nil
	}
	var aid GovActionId
	if err := cbor.Unmarshal(data, &aid); err != nil {
		return nil, err
	}
	return &aid, nil
}

func decodeWithdrawals(
	mapBytes []byte,
) ([]Withdrawal, error) {
	pairs, err := decodeMapPairs(mapBytes)
	if err != nil {
		return nil, err
	}
	result := make([]Withdrawal, 0, len(pairs))
	for _, p := range pairs {
		var acct []byte
		if err := cbor.Unmarshal(p.key, &acct); err != nil {
			return nil, fmt.Errorf(
				"decode reward account: %w",
				err,
			)
		}
		var coin int64
		if err := cbor.Unmarshal(
			p.value, &coin,
		); err != nil {
			return nil, fmt.Errorf(
				"decode withdrawal coin: %w",
				err,
			)
		}
		result = append(result, Withdrawal{
			RewardAccount: acct,
			Coin:          coin,
		})
	}
	return result, nil
}

func decodeAddedCC(
	mapBytes []byte,
) ([]AddedCommitteeMember, error) {
	pairs, err := decodeMapPairs(mapBytes)
	if err != nil {
		return nil, err
	}
	result := make([]AddedCommitteeMember, 0, len(pairs))
	for _, p := range pairs {
		var cred Certificate.Credential
		if err := cbor.Unmarshal(
			p.key, &cred,
		); err != nil {
			return nil, fmt.Errorf(
				"decode credential: %w",
				err,
			)
		}
		var epoch uint64
		if err := cbor.Unmarshal(
			p.value, &epoch,
		); err != nil {
			return nil, fmt.Errorf(
				"decode epoch: %w",
				err,
			)
		}
		result = append(result, AddedCommitteeMember{
			Credential: cred,
			Epoch:      epoch,
		})
	}
	return result, nil
}

// marshalPrevAction encodes a nullable prev action id.
func marshalPrevAction(prev *GovActionId) (any, error) {
	if prev == nil {
		return nil, nil
	}
	bz, err := cbor.Marshal(*prev)
	if err != nil {
		return nil, err
	}
	var v any
	if err := cbor.Unmarshal(bz, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// ----------------------------------------------------------------
// InfoAction  [6]
// ----------------------------------------------------------------

// InfoAction is the simplest governance action with no fields.
type InfoAction struct{}

func (a InfoAction) GovActionType() int { return 6 }

func (a InfoAction) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal([]any{a.GovActionType()})
}

// ----------------------------------------------------------------
// NoConfidence  [3, prev / null]
// ----------------------------------------------------------------

// NoConfidence expresses lack of confidence in the current
// constitutional committee.
type NoConfidence struct {
	PrevActionId *GovActionId
}

func (a NoConfidence) GovActionType() int { return 3 }

func (a NoConfidence) MarshalCBOR() ([]byte, error) {
	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}
	return cbor.Marshal([]any{a.GovActionType(), prev})
}

// ----------------------------------------------------------------
// HardForkInitiation  [1, prev / null, [major, minor]]
// ----------------------------------------------------------------

// ProtocolVersion pairs a major and minor version.
type ProtocolVersion struct {
	_     struct{} `cbor:",toarray"`
	Major uint64
	Minor uint64
}

// HardForkInitiation proposes a hard fork to a new protocol
// version.
type HardForkInitiation struct {
	PrevActionId    *GovActionId
	ProtocolVersion ProtocolVersion
}

func (a HardForkInitiation) GovActionType() int { return 1 }

func (a HardForkInitiation) MarshalCBOR() ([]byte, error) {
	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}
	return cbor.Marshal(
		[]any{
			a.GovActionType(),
			prev,
			a.ProtocolVersion,
		},
	)
}

// ----------------------------------------------------------------
// TreasuryWithdrawals
//
//	[2, {reward_account => coin}, prev / null]
//
// ----------------------------------------------------------------

// Withdrawal pairs a reward account with an amount.
type Withdrawal struct {
	RewardAccount []byte
	Coin          int64
}

// TreasuryWithdrawals proposes withdrawals from the treasury.
type TreasuryWithdrawals struct {
	Withdrawals  []Withdrawal
	PrevActionId *GovActionId
}

func (a TreasuryWithdrawals) GovActionType() int { return 2 }

func (a TreasuryWithdrawals) MarshalCBOR() (
	[]byte, error,
) {
	// Build the withdrawal map as raw CBOR
	wdRaw := make([][]byte, 0, len(a.Withdrawals)*2)
	for _, w := range a.Withdrawals {
		kBz, err := cbor.Marshal(w.RewardAccount)
		if err != nil {
			return nil, err
		}
		vBz, err := cbor.Marshal(w.Coin)
		if err != nil {
			return nil, err
		}
		wdRaw = append(wdRaw, kBz, vBz)
	}
	wdMap := buildCBORMapRaw(wdRaw)

	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}

	// Build [2, map_raw, prev] as a CBOR array manually
	// so the map stays as-is (not re-encoded).
	return marshalArrayWithRaw(
		[]any{a.GovActionType()},
		wdMap,
		[]any{prev},
	)
}

// ----------------------------------------------------------------
// ParameterChange  [0, prev / null, protocol_param_update]
// ----------------------------------------------------------------

// ParameterChange proposes changes to protocol parameters.
type ParameterChange struct {
	PrevActionId *GovActionId
	ParamUpdate  map[int]any
}

func (a ParameterChange) GovActionType() int { return 0 }

func (a ParameterChange) MarshalCBOR() ([]byte, error) {
	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}
	return cbor.Marshal(
		[]any{a.GovActionType(), prev, a.ParamUpdate},
	)
}

// ----------------------------------------------------------------
// UpdateCommittee
//
//	[4, prev/null, [*cred], {cred=>epoch}, quorum]
//
// ----------------------------------------------------------------

// AddedCommitteeMember pairs a credential with an epoch.
type AddedCommitteeMember struct {
	Credential Certificate.Credential
	Epoch      uint64
}

// UpdateCommittee proposes changes to the constitutional
// committee.
type UpdateCommittee struct {
	PrevActionId *GovActionId
	Removed      []Certificate.Credential
	Added        []AddedCommitteeMember
	Quorum       Certificate.UnitInterval
}

func (a UpdateCommittee) GovActionType() int { return 4 }

func (a UpdateCommittee) MarshalCBOR() ([]byte, error) {
	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}

	// Build added map {cred => epoch} as raw CBOR
	addedRaw := make([][]byte, 0, len(a.Added)*2)
	for _, m := range a.Added {
		credBz, err := cbor.Marshal(m.Credential)
		if err != nil {
			return nil, err
		}
		epochBz, err := cbor.Marshal(m.Epoch)
		if err != nil {
			return nil, err
		}
		addedRaw = append(addedRaw, credBz, epochBz)
	}
	addedMap := buildCBORMapRaw(addedRaw)

	// Encode: [4, prev, removed, addedMap, quorum]
	// We must embed addedMap as raw CBOR inside the array.
	removedBz, err := cbor.Marshal(a.Removed)
	if err != nil {
		return nil, err
	}
	quorumBz, err := cbor.Marshal(a.Quorum)
	if err != nil {
		return nil, err
	}
	prevBz, err := cbor.Marshal(prev)
	if err != nil {
		return nil, err
	}
	typeBz, err := cbor.Marshal(a.GovActionType())
	if err != nil {
		return nil, err
	}

	// Build the 5-element CBOR array manually
	var buf bytes.Buffer
	buf.WriteByte(0x85) // array of 5
	buf.Write(typeBz)
	buf.Write(prevBz)
	buf.Write(removedBz)
	buf.Write(addedMap)
	buf.Write(quorumBz)
	return buf.Bytes(), nil
}

// ----------------------------------------------------------------
// NewConstitution
//
//	[5, prev/null, [anchor, scripthash / null]]
//
// ----------------------------------------------------------------

// NewConstitution proposes a new constitution.
type NewConstitution struct {
	PrevActionId *GovActionId
	Anchor       Certificate.Anchor
	ScriptHash   []byte // nil means null
}

func (a NewConstitution) GovActionType() int { return 5 }

func (a NewConstitution) MarshalCBOR() ([]byte, error) {
	prev, err := marshalPrevAction(a.PrevActionId)
	if err != nil {
		return nil, err
	}
	var sh any
	if a.ScriptHash != nil {
		sh = a.ScriptHash
	}
	return cbor.Marshal([]any{
		a.GovActionType(),
		prev,
		[]any{a.Anchor, sh},
	})
}

// ================================================================
// ProposalProcedure / ProposalProcedures
// ================================================================

// ProposalProcedure encodes a single governance proposal.
type ProposalProcedure struct {
	Deposit       int64
	RewardAccount []byte
	Action        GovAction
	Anchor        Certificate.Anchor
}

// MarshalCBOR encodes as
// [deposit, reward_account, gov_action, anchor].
func (pp ProposalProcedure) MarshalCBOR() ([]byte, error) {
	if pp.Action == nil {
		return nil, errors.New("proposal action is nil")
	}
	actionBz, err := pp.Action.MarshalCBOR()
	if err != nil {
		return nil, err
	}

	depositBz, err := cbor.Marshal(pp.Deposit)
	if err != nil {
		return nil, err
	}
	rewardBz, err := cbor.Marshal(pp.RewardAccount)
	if err != nil {
		return nil, err
	}
	anchorBz, err := cbor.Marshal(pp.Anchor)
	if err != nil {
		return nil, err
	}

	// Build 4-element CBOR array manually to embed
	// raw action bytes.
	var buf bytes.Buffer
	buf.WriteByte(0x84) // array of 4
	buf.Write(depositBz)
	buf.Write(rewardBz)
	buf.Write(actionBz)
	buf.Write(anchorBz)
	return buf.Bytes(), nil
}

// UnmarshalProposalProcedure decodes a single proposal.
func UnmarshalProposalProcedure(
	data []byte,
) (ProposalProcedure, error) {
	items, err := decodeArrayItems(data)
	if err != nil {
		return ProposalProcedure{}, err
	}
	if len(items) != 4 {
		return ProposalProcedure{}, errors.New(
			"proposal_procedure must have 4 elements",
		)
	}

	var deposit int64
	if err := cbor.Unmarshal(
		items[0], &deposit,
	); err != nil {
		return ProposalProcedure{}, err
	}

	var rewardAccount []byte
	if err := cbor.Unmarshal(
		items[1], &rewardAccount,
	); err != nil {
		return ProposalProcedure{}, err
	}

	action, err := UnmarshalGovAction(items[2])
	if err != nil {
		return ProposalProcedure{}, err
	}

	var anchor Certificate.Anchor
	if err := cbor.Unmarshal(items[3], &anchor); err != nil {
		return ProposalProcedure{}, err
	}

	return ProposalProcedure{
		Deposit:       deposit,
		RewardAccount: rewardAccount,
		Action:        action,
		Anchor:        anchor,
	}, nil
}

// ProposalProcedures is a collection of proposals.
type ProposalProcedures []ProposalProcedure

// MarshalCBOR encodes as a CBOR array.
func (ps ProposalProcedures) MarshalCBOR() (
	[]byte, error,
) {
	rawItems := make([][]byte, 0, len(ps))
	for _, p := range ps {
		bz, err := p.MarshalCBOR()
		if err != nil {
			return nil, err
		}
		rawItems = append(rawItems, bz)
	}
	return buildCBORArrayRaw(rawItems), nil
}

func buildCBORArrayRaw(items [][]byte) []byte {
	count := len(items)
	var buf bytes.Buffer
	switch {
	case count <= 23:
		buf.WriteByte(0x80 | byte(count))
	case count <= 255:
		buf.WriteByte(0x98)
		buf.WriteByte(byte(count))
	default:
		buf.WriteByte(0x99)
		buf.WriteByte(byte(count >> 8))
		buf.WriteByte(byte(count))
	}
	for _, bz := range items {
		buf.Write(bz)
	}
	return buf.Bytes()
}

// UnmarshalCBOR decodes a CBOR array of proposal procedures.
func (ps *ProposalProcedures) UnmarshalCBOR(
	data []byte,
) error {
	items, err := decodeArrayItems(data)
	if err != nil {
		// Try standard CBOR decode as fallback for
		// empty arrays
		var raw []any
		if err2 := cbor.Unmarshal(data, &raw); err2 != nil {
			return err
		}
		if len(raw) == 0 {
			*ps = ProposalProcedures{}
			return nil
		}
		return err
	}
	result := make(ProposalProcedures, 0, len(items))
	for _, item := range items {
		pp, err := UnmarshalProposalProcedure(item)
		if err != nil {
			return err
		}
		result = append(result, pp)
	}
	*ps = result
	return nil
}

// marshalArrayWithRaw builds a CBOR array where some elements
// are pre-encoded CBOR bytes (the rawMiddle) and others are
// Go values to be marshaled normally. The result is:
//
//	[prefix_values..., rawMiddle, suffix_values...]
func marshalArrayWithRaw(
	prefix []any,
	rawMiddle []byte,
	suffix []any,
) ([]byte, error) {
	total := len(prefix) + 1 + len(suffix)
	var buf bytes.Buffer
	switch {
	case total <= 23:
		buf.WriteByte(0x80 | byte(total))
	case total <= 255:
		buf.WriteByte(0x98)
		buf.WriteByte(byte(total))
	default:
		buf.WriteByte(0x99)
		buf.WriteByte(byte(total >> 8))
		buf.WriteByte(byte(total))
	}

	for _, v := range prefix {
		bz, err := cbor.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(bz)
	}
	buf.Write(rawMiddle)
	for _, v := range suffix {
		bz, err := cbor.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(bz)
	}
	return buf.Bytes(), nil
}
