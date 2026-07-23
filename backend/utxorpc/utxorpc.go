package utxorpc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/babbage"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/conway"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	cardano "github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	query "github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	syncpb "github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync"
	sdk "github.com/utxorpc/go-sdk"

	"github.com/Salvionied/apollo/v2/backend"
)

// UtxoRpcChainContext implements backend.ChainContext using the UTxO RPC protocol.
type UtxoRpcChainContext struct {
	client    *sdk.UtxorpcClient
	networkId uint8
}

// Capabilities reports the UTxO RPC operations supported by this client.
func (u *UtxoRpcChainContext) Capabilities() backend.CapabilitySet {
	return backend.CapabilitySet(backend.AllCapabilities) &^
		backend.CapabilitySet(backend.CapabilityGenesisParams|backend.CapabilityCurrentEpoch|
			backend.CapabilityEvaluateTxAdditionalUtxos|backend.CapabilityScriptCbor)
}

// EvaluationError reports deterministic Cardano transaction-evaluation errors
// returned by the UTxO RPC provider.
type EvaluationError struct {
	Messages []string
}

func (e *EvaluationError) Error() string {
	return strings.Join(e.Messages, "; ")
}

// NewUtxoRpcChainContext creates a new UTxO RPC chain context.
func NewUtxoRpcChainContext(baseUrl string, networkId uint8, headers map[string]string) *UtxoRpcChainContext {
	opts := []sdk.ClientOption{
		sdk.WithBaseUrl(baseUrl),
	}
	if len(headers) > 0 {
		opts = append(opts, sdk.WithHeaders(headers))
	}
	client := sdk.NewClient(opts...)
	return &UtxoRpcChainContext{
		client:    client,
		networkId: networkId,
	}
}

func bigIntToInt64(bi *cardano.BigInt) int64 {
	if bi == nil {
		return 0
	}
	oneof := bi.GetBigInt()
	if oneof == nil {
		return 0
	}
	switch v := oneof.(type) {
	case *cardano.BigInt_Int:
		if v == nil {
			return 0
		}
		return v.Int
	case *cardano.BigInt_BigUInt:
		if v == nil {
			return 0
		}
		n := new(big.Int).SetBytes(v.BigUInt)
		if !n.IsInt64() {
			return math.MaxInt64
		}
		return n.Int64()
	case *cardano.BigInt_BigNInt:
		if v == nil {
			return 0
		}
		n := new(big.Int).SetBytes(v.BigNInt)
		n.Neg(n)
		if !n.IsInt64() {
			return math.MinInt64
		}
		return n.Int64()
	default:
		return 0
	}
}

func bigIntToString(bi *cardano.BigInt) string {
	if bi == nil {
		return "0"
	}
	oneof := bi.GetBigInt()
	if oneof == nil {
		return "0"
	}
	switch v := oneof.(type) {
	case *cardano.BigInt_Int:
		if v == nil {
			return "0"
		}
		return strconv.FormatInt(v.Int, 10)
	case *cardano.BigInt_BigUInt:
		if v == nil {
			return "0"
		}
		return new(big.Int).SetBytes(v.BigUInt).String()
	case *cardano.BigInt_BigNInt:
		if v == nil {
			return "0"
		}
		n := new(big.Int).SetBytes(v.BigNInt)
		n.Neg(n)
		return n.String()
	default:
		return "0"
	}
}

func (u *UtxoRpcChainContext) ProtocolParams() (backend.ProtocolParameters, error) {
	req := connect.NewRequest(&query.ReadParamsRequest{})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.ReadParams(req)
	if err != nil {
		return backend.ProtocolParameters{}, err
	}

	params := resp.Msg.GetValues().GetCardano()
	if params == nil {
		return backend.ProtocolParameters{}, errors.New("no cardano params in response")
	}

	maxBlockSize, err := backend.BoundedIntFromUint64(params.GetMaxBlockBodySize(), "max block body size")
	if err != nil {
		return backend.ProtocolParameters{}, err
	}
	maxTxSize, err := backend.BoundedIntFromUint64(params.GetMaxTxSize(), "max tx size")
	if err != nil {
		return backend.ProtocolParameters{}, err
	}
	maxBlockHeaderSize, err := backend.BoundedIntFromUint64(params.GetMaxBlockHeaderSize(), "max block header size")
	if err != nil {
		return backend.ProtocolParameters{}, err
	}
	collateralPercent, err := backend.BoundedIntFromUint64(params.GetCollateralPercentage(), "collateral percentage")
	if err != nil {
		return backend.ProtocolParameters{}, err
	}
	maxCollateralInputs, err := backend.BoundedIntFromUint64(params.GetMaxCollateralInputs(), "max collateral inputs")
	if err != nil {
		return backend.ProtocolParameters{}, err
	}

	pp := backend.ProtocolParameters{
		MinFeeCoefficient:   bigIntToInt64(params.GetMinFeeCoefficient()),
		MinFeeConstant:      bigIntToInt64(params.GetMinFeeConstant()),
		MaxBlockSize:        maxBlockSize,
		MaxTxSize:           maxTxSize,
		MaxBlockHeaderSize:  maxBlockHeaderSize,
		CoinsPerUtxoByte:    bigIntToString(params.GetCoinsPerUtxoByte()),
		MaxValSize:          strconv.FormatUint(params.GetMaxValueSize(), 10),
		CollateralPercent:   collateralPercent,
		MaxCollateralInputs: maxCollateralInputs,
		KeyDeposits:         bigIntToString(params.GetStakeKeyDeposit()),
		PoolDeposits:        bigIntToString(params.GetPoolDeposit()),
	}

	if txEx := params.GetMaxExecutionUnitsPerTransaction(); txEx != nil {
		pp.MaxTxExMem = strconv.FormatUint(txEx.GetMemory(), 10)
		pp.MaxTxExSteps = strconv.FormatUint(txEx.GetSteps(), 10)
	}
	if blockEx := params.GetMaxExecutionUnitsPerBlock(); blockEx != nil {
		pp.MaxBlockExMem = strconv.FormatUint(blockEx.GetMemory(), 10)
		pp.MaxBlockExSteps = strconv.FormatUint(blockEx.GetSteps(), 10)
	}

	prices := params.GetPrices()
	if prices != nil {
		if prices.GetMemory() != nil && prices.GetMemory().GetDenominator() != 0 {
			pp.PriceMem = float64(prices.GetMemory().GetNumerator()) / float64(prices.GetMemory().GetDenominator())
		}
		if prices.GetSteps() != nil && prices.GetSteps().GetDenominator() != 0 {
			pp.PriceStep = float64(prices.GetSteps().GetNumerator()) / float64(prices.GetSteps().GetDenominator())
		}
	}

	// Conway reference-script base price (lovelace per byte for the first tier),
	// exposed as a rational number.
	if refCost := params.GetMinFeeScriptRefCostPerByte(); refCost != nil && refCost.GetDenominator() != 0 {
		pp.MinFeeRefScriptCostPerByte = float64(refCost.GetNumerator()) / float64(refCost.GetDenominator())
	}

	// Parse cost models from UTxO RPC protobuf response.
	// Keys match ComputeScriptDataHash expectations: "PlutusV1", "PlutusV2", "PlutusV3".
	if cm := params.GetCostModels(); cm != nil {
		pp.CostModels = make(map[string][]int64)
		if v1 := cm.GetPlutusV1(); v1 != nil {
			pp.CostModels["PlutusV1"] = append([]int64(nil), v1.GetValues()...)
		}
		if v2 := cm.GetPlutusV2(); v2 != nil {
			pp.CostModels["PlutusV2"] = append([]int64(nil), v2.GetValues()...)
		}
		if v3 := cm.GetPlutusV3(); v3 != nil {
			pp.CostModels["PlutusV3"] = append([]int64(nil), v3.GetValues()...)
		}
	}

	return pp, nil
}

func (u *UtxoRpcChainContext) GenesisParams() (backend.GenesisParameters, error) {
	return backend.GenesisParameters{}, backend.NewUnsupportedError("UTxO RPC", backend.CapabilityGenesisParams)
}

func (u *UtxoRpcChainContext) NetworkId() uint8 {
	return u.networkId
}

func (u *UtxoRpcChainContext) CurrentEpoch() (uint64, error) {
	return 0, backend.NewUnsupportedError("UTxO RPC", backend.CapabilityCurrentEpoch)
}

func (u *UtxoRpcChainContext) MaxTxFee() (uint64, error) {
	pp, err := u.ProtocolParams()
	if err != nil {
		return 0, err
	}
	return backend.ComputeMaxTxFee(pp)
}

func (u *UtxoRpcChainContext) Tip() (uint64, error) {
	req := connect.NewRequest(&syncpb.ReadTipRequest{})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.ReadTip(req)
	if err != nil {
		return 0, err
	}
	tip := resp.Msg.GetTip()
	if tip == nil {
		return 0, errors.New("no tip in response")
	}
	return tip.GetSlot(), nil
}

func (u *UtxoRpcChainContext) Utxos(address common.Address) ([]common.Utxo, error) {
	addrBytes, err := address.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get address bytes: %w", err)
	}

	req := connect.NewRequest(&query.SearchUtxosRequest{
		Predicate: &query.UtxoPredicate{
			Match: &query.AnyUtxoPattern{
				UtxoPattern: &query.AnyUtxoPattern_Cardano{
					Cardano: &cardano.TxOutputPattern{
						Address: &cardano.AddressPattern{
							ExactAddress: addrBytes,
						},
					},
				},
			},
		},
	})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.SearchUtxos(req)
	if err != nil {
		return nil, err
	}

	var utxos []common.Utxo
	for _, item := range resp.Msg.GetItems() {
		utxo, err := utxoFromRpc(item)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UTxO from RPC: %w", err)
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

func (u *UtxoRpcChainContext) SubmitTx(txCbor []byte) (common.Blake2b256, error) {
	req := connect.NewRequest(&submit.SubmitTxRequest{
		Tx: &submit.AnyChainTx{
			Type: &submit.AnyChainTx_Raw{Raw: txCbor},
		},
	})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.SubmitTx(req)
	if err != nil {
		return common.Blake2b256{}, err
	}
	ref := resp.Msg.GetRef()
	if len(ref) == 0 {
		return common.Blake2b256{}, errors.New("no tx ref in submit response")
	}
	if len(ref) != common.Blake2b256Size {
		return common.Blake2b256{}, fmt.Errorf("invalid tx ref length: expected %d bytes, got %d", common.Blake2b256Size, len(ref))
	}
	var result common.Blake2b256
	copy(result[:], ref)
	return result, nil
}

// EvaluateTx evaluates the scripts in a transaction. UTxO RPC has no wire
// field for additional/resolved UTxOs, so off-chain or chained inputs cannot
// be evaluated by this backend.
func (u *UtxoRpcChainContext) EvaluateTx(txCbor []byte, additionalUtxos []common.Utxo) (map[common.RedeemerKey]common.ExUnits, error) {
	if len(additionalUtxos) > 0 {
		return nil, backend.NewUnsupportedError("UTxO RPC", backend.CapabilityEvaluateTxAdditionalUtxos)
	}
	expected, err := expectedRedeemerKeys(txCbor)
	if err != nil {
		return nil, err
	}
	req := connect.NewRequest(&submit.EvalTxRequest{
		Tx: &submit.AnyChainTx{
			Type: &submit.AnyChainTx_Raw{Raw: txCbor},
		},
	})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.EvalTx(req)
	if err != nil {
		return nil, fmt.Errorf("evaluate transaction: %w", err)
	}
	return evalTxResponseToExpectedExUnits(resp.Msg, expected)
}

// evalTxResponseToExUnits converts an EvalTxResponse into a redeemer ExUnits
// map. A missing report, missing cardano report, or zero evaluation results
// is an error: returning an empty map with a nil error would let callers
// silently keep zero execution budgets for their redeemers.
func evalTxResponseToExUnits(msg *submit.EvalTxResponse) (map[common.RedeemerKey]common.ExUnits, error) {
	return parseEvalTxResponse(msg, utxorpcPurposeToRedeemerTag)
}

func evalTxResponseToExpectedExUnits(
	msg *submit.EvalTxResponse,
	expected map[common.RedeemerKey]struct{},
) (map[common.RedeemerKey]common.ExUnits, error) {
	standard, standardErr := parseEvalTxResponse(msg, utxorpcPurposeToRedeemerTag)
	var evaluationErr *EvaluationError
	if errors.As(standardErr, &evaluationErr) {
		return nil, standardErr
	}
	fallback, fallbackErr := parseEvalTxResponse(msg, utxorpcZeroBasedPurposeToRedeemerTag)

	standardMatches := standardErr == nil && sameRedeemerKeySet(standard, expected)
	fallbackMatches := fallbackErr == nil && sameRedeemerKeySet(fallback, expected)
	switch {
	case standardMatches && fallbackMatches:
		if !sameRedeemerKeySet(standard, redeemerKeySet(fallback)) {
			return nil, fmt.Errorf(
				"ambiguous redeemer purpose encoding: expected keys %s; standard observed keys %s; zero-based observed keys %s",
				formatRedeemerKeys(redeemerKeySet(expected)),
				formatRedeemerKeys(redeemerKeySet(standard)),
				formatRedeemerKeys(redeemerKeySet(fallback)),
			)
		}
		return standard, nil
	case standardMatches:
		return standard, nil
	case fallbackMatches:
		return fallback, nil
	default:
		return nil, fmt.Errorf(
			"redeemer purpose encoding does not match transaction: expected keys %s; standard observed keys %s (%v); zero-based observed keys %s (%v)",
			formatRedeemerKeys(redeemerKeySet(expected)),
			formatRedeemerKeys(redeemerKeySet(standard)),
			standardErr,
			formatRedeemerKeys(redeemerKeySet(fallback)),
			fallbackErr,
		)
	}
}

func expectedRedeemerKeys(txCbor []byte) (map[common.RedeemerKey]struct{}, error) {
	var tx conway.ConwayTransaction
	if _, err := cbor.Decode(txCbor, &tx); err != nil {
		return nil, fmt.Errorf("failed to decode submitted transaction: %w", err)
	}
	expected := make(map[common.RedeemerKey]struct{}, len(tx.WitnessSet.WsRedeemers.Redeemers))
	for key := range tx.WitnessSet.WsRedeemers.Redeemers {
		expected[key] = struct{}{}
	}
	return expected, nil
}

func parseEvalTxResponse(
	msg *submit.EvalTxResponse,
	mapPurpose func(cardano.RedeemerPurpose) (common.RedeemerTag, error),
) (map[common.RedeemerKey]common.ExUnits, error) {
	if msg == nil {
		return nil, errors.New("empty evaluate response")
	}
	report := msg.GetReport()
	if report == nil {
		return nil, errors.New("no evaluation report in response")
	}
	cardanoReport := report.GetCardano()
	if cardanoReport == nil {
		return nil, errors.New("no cardano evaluation report in response")
	}
	if responseErrors := cardanoReport.GetErrors(); len(responseErrors) > 0 {
		messages := make([]string, 0, len(responseErrors))
		for _, responseError := range responseErrors {
			message := strings.TrimSpace(responseError.GetMsg())
			if message != "" {
				messages = append(messages, message)
			}
		}
		if len(messages) == 0 {
			messages = []string{"script evaluation failed without an error message"}
		}
		return nil, &EvaluationError{Messages: messages}
	}
	result := make(map[common.RedeemerKey]common.ExUnits)
	for _, redeemer := range cardanoReport.GetRedeemers() {
		tag, err := mapPurpose(redeemer.GetPurpose())
		if err != nil {
			return result, fmt.Errorf("failed to map redeemer purpose: %w", err)
		}
		key := common.RedeemerKey{
			Tag:   tag,
			Index: redeemer.GetIndex(),
		}
		if _, exists := result[key]; exists {
			return result, fmt.Errorf("duplicate evaluation report for redeemer %d:%d", tag, redeemer.GetIndex())
		}
		eu := redeemer.GetExUnits()
		if eu == nil {
			return result, fmt.Errorf("no ExUnits in evaluation report for redeemer %d:%d", tag, redeemer.GetIndex())
		}
		mem := eu.GetMemory()
		steps := eu.GetSteps()
		if mem > math.MaxInt64 || steps > math.MaxInt64 {
			return result, fmt.Errorf("ExUnits overflow: memory=%d steps=%d", mem, steps)
		}
		result[key] = common.ExUnits{
			Memory: int64(mem),
			Steps:  int64(steps),
		}
	}
	if len(result) == 0 {
		return nil, errors.New("script evaluation returned no results")
	}
	return result, nil
}

func redeemerKeySet[V any](values map[common.RedeemerKey]V) map[common.RedeemerKey]struct{} {
	keys := make(map[common.RedeemerKey]struct{}, len(values))
	for key := range values {
		keys[key] = struct{}{}
	}
	return keys
}

func sameRedeemerKeySet[V any](actual map[common.RedeemerKey]V, expected map[common.RedeemerKey]struct{}) bool {
	if len(actual) != len(expected) {
		return false
	}
	for key := range actual {
		if _, ok := expected[key]; !ok {
			return false
		}
	}
	return true
}

func formatRedeemerKeys(keys map[common.RedeemerKey]struct{}) string {
	sorted := make([]common.RedeemerKey, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return common.CompareRedeemerKeys(sorted[i], sorted[j]) < 0
	})
	formatted := make([]string, 0, len(sorted))
	for _, key := range sorted {
		formatted = append(formatted, fmt.Sprintf("%d:%d", key.Tag, key.Index))
	}
	return "[" + strings.Join(formatted, ", ") + "]"
}

func (u *UtxoRpcChainContext) UtxoByRef(txHash common.Blake2b256, index uint32) (*common.Utxo, error) {
	req := connect.NewRequest(&query.ReadUtxosRequest{
		Keys: []*query.TxoRef{
			{
				Hash:  txHash.Bytes(),
				Index: index,
			},
		},
	})
	u.client.AddHeadersToRequest(req)
	resp, err := u.client.ReadUtxos(req)
	if err != nil {
		return nil, err
	}
	items := resp.Msg.GetItems()
	if len(items) == 0 {
		return nil, errors.New("utxo not found")
	}
	utxo, err := utxoFromRpc(items[0])
	if err != nil {
		return nil, err
	}
	return &utxo, nil
}

func (u *UtxoRpcChainContext) ScriptCbor(_ common.Blake2b224) ([]byte, error) {
	return nil, backend.NewUnsupportedError("UTxO RPC", backend.CapabilityScriptCbor)
}

func utxoFromRpc(item *query.AnyUtxoData) (common.Utxo, error) {
	nativeBytes := item.GetNativeBytes()
	if len(nativeBytes) == 0 {
		ref := item.GetTxoRef()
		return common.Utxo{}, fmt.Errorf("no native bytes for utxo %s#%d",
			hex.EncodeToString(ref.GetHash()), ref.GetIndex())
	}

	// Parse the CBOR-encoded transaction output
	output, err := babbage.NewBabbageTransactionOutputFromCbor(nativeBytes)
	if err != nil {
		return common.Utxo{}, fmt.Errorf("failed to parse utxo CBOR: %w", err)
	}

	ref := item.GetTxoRef()
	refHash := ref.GetHash()
	if len(refHash) != common.Blake2b256Size {
		return common.Utxo{}, fmt.Errorf("invalid tx hash length: expected %d bytes, got %d", common.Blake2b256Size, len(refHash))
	}
	var txId common.Blake2b256
	copy(txId[:], refHash)

	input := shelley.ShelleyTransactionInput{
		TxId:        txId,
		OutputIndex: ref.GetIndex(),
	}
	return common.Utxo{
		Id:     input,
		Output: output,
	}, nil
}

// utxorpcPurposeToRedeemerTag maps UTxO RPC redeemer purpose enum to gouroboros RedeemerTag.
// UTxO RPC uses 1-based enum (SPEND=1, MINT=2, ...) while gouroboros uses 0-based (Spend=0, Mint=1, ...).
func utxorpcPurposeToRedeemerTag(purpose cardano.RedeemerPurpose) (common.RedeemerTag, error) {
	switch purpose {
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND:
		return common.RedeemerTagSpend, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT:
		return common.RedeemerTagMint, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_CERT:
		return common.RedeemerTagCert, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_REWARD:
		return common.RedeemerTagReward, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_VOTE:
		return common.RedeemerTagVoting, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_PROPOSE:
		return common.RedeemerTagProposing, nil
	default:
		return 0, fmt.Errorf("unsupported redeemer purpose: %d", purpose)
	}
}

func utxorpcZeroBasedPurposeToRedeemerTag(purpose cardano.RedeemerPurpose) (common.RedeemerTag, error) {
	switch purpose {
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_UNSPECIFIED:
		return common.RedeemerTagSpend, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_SPEND:
		return common.RedeemerTagMint, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_MINT:
		return common.RedeemerTagCert, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_CERT:
		return common.RedeemerTagReward, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_REWARD:
		return common.RedeemerTagVoting, nil
	case cardano.RedeemerPurpose_REDEEMER_PURPOSE_VOTE:
		return common.RedeemerTagProposing, nil
	default:
		return 0, fmt.Errorf("unsupported zero-based redeemer purpose: %d", purpose)
	}
}
