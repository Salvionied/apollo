# Contributing

Use Go 1.25.12 or newer. Before opening a pull request, run:

```bash
gofmt -w $(find . -name '*.go' -not -path './.worktrees/*')
go test -race ./...
```

Keep transaction and CBOR changes covered by deterministic tests. Backend
changes should include provider-response fixtures and should return errors
instead of panicking or silently accepting malformed data.
