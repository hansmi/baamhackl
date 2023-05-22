module github.com/hansmi/baamhackl

go 1.19

// go-yaml 1.10 fails to build on 32 bit platforms: "cannot use math.MaxInt64
// […] as int value in assignment".
//
// https://github.com/goccy/go-yaml/pull/350
exclude github.com/goccy/go-yaml v1.10.0

require (
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/go-playground/validator/v10 v10.14.0
	github.com/goccy/go-yaml v1.11.0
	github.com/gofrs/flock v0.8.1
	github.com/google/go-cmp v0.5.9
	github.com/google/subcommands v1.2.0
	github.com/jonboulle/clockwork v0.4.0
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/prometheus/client_golang v1.15.1
	github.com/prometheus/common v0.43.0
	github.com/rivo/uniseg v0.4.4
	github.com/spf13/afero v1.9.5
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.24.0
	golang.org/x/sync v0.2.0
	golang.org/x/sys v0.8.0
)

require (
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/google/renameio/v2 v2.0.0
	github.com/mattn/go-colorable v0.1.13 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
)
