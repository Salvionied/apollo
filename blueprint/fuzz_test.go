package blueprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

func FuzzParseAndValidate(f *testing.F) {
	seedFiles := []string{
		"spend_record.json",
		"mint_policy.json",
		"multi_purpose.json",
		"asset_map_tuple.json",
		"wide_datum.json",
		"governance.json",
		"tx_actions.json",
		"invalid_cases.json",
	}
	for _, name := range seedFiles {
		raw, err := os.ReadFile(filepath.Join("testdata", name))
		if err == nil {
			f.Add(raw, []byte{0x01})
		}
	}

	f.Fuzz(func(_ *testing.T, rawJSON []byte, rawCBOR []byte) {
		bp, err := Parse(rawJSON)
		if err != nil {
			return
		}
		pd, err := data.Decode(rawCBOR)
		if err != nil {
			return
		}
		for i := range bp.Validators {
			v := &bp.Validators[i]
			if v.Datum != nil {
				_ = bp.ValidateDatum(v.Title, pd)
			}
			if v.Redeemer.Single != nil {
				_ = bp.ValidateData(&v.Redeemer.Single.Schema, pd)
			}
		}
	})
}
