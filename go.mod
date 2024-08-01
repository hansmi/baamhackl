module github.com/hansmi/baamhackl

go 1.21.0

toolchain go1.22.1

// go-yaml 1.10 fails to build on 32 bit platforms: "cannot use math.MaxInt64
// […] as int value in assignment".
//
// https://github.com/goccy/go-yaml/pull/350
exclude github.com/goccy/go-yaml v1.10.0

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/go-playground/validator/v10 v10.22.0
	github.com/goccy/go-yaml v1.12.0
	github.com/gofrs/flock v0.11.0
	github.com/google/go-cmp v0.6.0
	github.com/google/subcommands v1.2.0
	github.com/jonboulle/clockwork v0.4.0
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/prometheus/client_golang v1.19.1
	github.com/rivo/uniseg v0.4.7
	github.com/spf13/afero v1.11.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.7.0
	golang.org/x/sys v0.21.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/google/renameio/v2 v2.0.0
	github.com/mattn/go-colorable v0.1.13 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
)
