package apollotypes

import (
	"encoding/hex"
	"errors"

	"github.com/Salvionied/apollo/serialization/PlutusData"
)

type AikenPlutusJSON struct {
	Preamble struct {
		Title         string `json:"title"`
		Description   string `json:"description"`
		Version       string `json:"version"`
		PlutusVersion string `json:"plutusVersion"`
		License       string `json:"license"`
	} `json:"preamble"`
	Validators []struct {
		Title string `json:"title"`
		Datum struct {
			Title  string `json:"title"`
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"datum"`
		Redeemer struct {
			Title  string `json:"title"`
			Schema struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"redeemer"`
		CompiledCode string `json:"compiledCode"`
		Hash         string `json:"hash"`
	} `json:"validators"`
	Definitions struct {
	} `json:"definitions"`
}

/*
*

	GetScript retrives a Plutus V2 script by its name from an AikenPlutusJSON object.
	It searches through the Validators and returns the script if found.

	Params:
		apj (*AikenPlutusJSON): A pointer to an AikenPlutusJSON object.
		name (string): the name of the script to retrieve.

	Returns:
		(*PlutusData.PlutusV2Script, error): A pointer to a Plutus V2 script and an error (if any).
*/
func (apj *AikenPlutusJSON) GetScript(name string) (*PlutusData.PlutusV2Script, error) {
	for _, validator := range apj.Validators {
		if validator.Title == name {
			decoded_string, err := hex.DecodeString(validator.CompiledCode)
			if err != nil {
				return nil, err
			}
			p2Script := PlutusData.PlutusV2Script(decoded_string)
			return &p2Script, nil
		}
	}
	return nil, errors.New("script not found")
}
