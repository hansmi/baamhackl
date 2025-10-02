module github.com/hansmi/baamhackl

go 1.24.0

toolchain go1.24.1

// go-yaml 1.10 fails to build on 32 bit platforms: "cannot use math.MaxInt64
// […] as int value in assignment".
//
// https://github.com/goccy/go-yaml/pull/350
exclude github.com/goccy/go-yaml v1.10.0

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/go-playground/validator/v10 v10.27.0
	github.com/goccy/go-yaml v1.18.0
	github.com/gofrs/flock v0.12.1
	github.com/google/go-cmp v0.7.0
	github.com/google/subcommands v1.2.0
	github.com/jonboulle/clockwork v0.5.0
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/prometheus/client_golang v1.23.2
	github.com/rivo/uniseg v0.4.7
	github.com/spf13/afero v1.15.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.17.0
	golang.org/x/sys v0.35.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

require (
	github.com/google/renameio/v2 v2.0.0
	golang.org/x/crypto v0.41.0 // indirect
)
