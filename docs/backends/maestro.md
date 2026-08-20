# Maestro

```go
import "github.com/Salvionied/apollo/v2/backend/maestro"

// network ID 1 → "mainnet"; network ID 0 → "preprod"
ctx, err := maestro.NewMaestroChainContext(1, "your_maestro_project_id")
if err != nil {
    return err
}

// Preview (network ID is still 0; the name is not implied by the ID)
ctx, err = maestro.NewMaestroChainContextWithNetwork(0, "your_maestro_project_id", "preview")
```

`NewMaestroChainContextWithNetwork` accepts only `mainnet`, `preprod`, and
`preview`. Anything else is rejected: the Maestro SDK interpolates the name
into the API hostname.

## Capabilities

Maestro reports `AllCapabilities` except `CapabilityGenesisParams`.

## Behaviour worth knowing

- Evaluation reports are validated before they are applied.
- Object-shaped `additional_utxos` are sent with a direct request; the SDK's
  `[]string` type cannot represent them.
