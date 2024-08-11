Plutus Struct tags:

plutusConstr: int -> Defines the constructor - for no constructor
plutusType: Bytes || Int || Map || IndefList || DefList || StringBytes
PredefinedPlutusTypes: Address





Sample
```
type Datum struct {
    _ struct `plutusType:IndefList plutusConstr:1`
    Pkh []byte `plutusType:Bytes`
    Amount int64 `plutusType:Int`
}

type NestedDatum struct {
     _ struct `plutusType:IndefList plutusConstr:1`
    Pkh []byte `plutusType:Bytes`
    Amount int64 `plutusType:Int`
    Extra Datum
}

```



Usage
Marshaling
```
    d = Datum{...}

    plutusData, err := plutusencoder.MarshalPlutus(d)
```
Unmarshaling
```
    plutusData = PlutusData.PlutusData{...}
    d = Datum{...}
    err := plutusencoder.UnmarshalPlutus(d)

```
