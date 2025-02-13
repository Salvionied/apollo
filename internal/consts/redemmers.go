package consts

import (
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
)

var (
	INDEX_ONE_MINT_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.MINT,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          122,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_TWO_MINT_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.MINT,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          123,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_ONE_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          122,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_TWO_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          123,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_THREE_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          124,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_FOUR_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          125,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_FIVE_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          126,
			Value:          PlutusData.PlutusDefArray{},
		},
	}

	INDEX_SIX_SPEND_REDEEMER *Redeemer.Redeemer = &Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 0,
		Data: PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusArray,
			TagNr:          126,
			Value:          PlutusData.PlutusDefArray{},
		},
	}
)
