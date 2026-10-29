package backendutil

import (
	"fmt"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// ScriptRefFromHash identifies a script's language by matching its hash to
// the requested hash while preserving the script's original bytes.
func ScriptRefFromHash(scriptHash common.Blake2b224, scriptCbor []byte) (*common.ScriptRef, error) {
	var native common.NativeScript
	if _, err := cbor.DecodeExact(scriptCbor, &native); err == nil && native.Hash() == scriptHash {
		return &common.ScriptRef{
			Type:   common.ScriptRefTypeNativeScript,
			Script: native,
		}, nil
	}
	for _, candidate := range []struct {
		typ    uint
		script common.Script
	}{
		{common.ScriptRefTypePlutusV1, common.PlutusV1Script(scriptCbor)},
		{common.ScriptRefTypePlutusV2, common.PlutusV2Script(scriptCbor)},
		{common.ScriptRefTypePlutusV3, common.PlutusV3Script(scriptCbor)},
		{common.ScriptRefTypePlutusV4, common.PlutusV4Script(scriptCbor)},
	} {
		if candidate.script.Hash() == scriptHash {
			return &common.ScriptRef{Type: candidate.typ, Script: candidate.script}, nil
		}
	}
	return nil, fmt.Errorf("script bytes do not match requested hash %s", scriptHash)
}
