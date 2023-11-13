Plutus Struct tags:

plutusConstr: int -> Defines the constructor - for no constructor
plutusType: Bytes || Int || Map || IndefList || DefList || StringBytes




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

    plutusData, err := plutus.MarshalPlutus(d)
```
Unmarshaling
```
    plutusData = PlutusData.PlutusData{...}
    d = Datum{...}
    err := plutus.UnmarshalPlutus(d)

```
