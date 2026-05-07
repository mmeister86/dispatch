# Contributing

## Development

```bash
make test
make build
./bin/dispatch version
```

## Adding Providers

Providers implement `internal/provider.Provider` and stream `provider.Chunk` values. Keep provider-specific JSON mapping inside the provider package and cover it with `httptest`.

## Adding Tools

Tools implement `internal/tools.Tool`, expose JSON schemas via `Definitions`, and execute by name with raw JSON arguments. Keep network clients small and test them with `httptest`.
