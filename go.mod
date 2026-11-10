module github.com/janearc/slayer

go 1.27.3

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
)

require (
	github.com/creack/pty v1.1.24
	github.com/janearc/celestial v0.0.5-review.20261104
	github.com/janearc/celestial-data v0.0.0-review.20261104
	github.com/janearc/libtheme-css v0.4.6
	github.com/janearc/puzzlemouth v0.1.4
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	go.uber.org/mock v0.6.0
	golang.org/x/image v0.46.0
	golang.org/x/mod v0.41.0
	golang.org/x/sys v0.49.0
	golang.org/x/term v0.47.0
	gopkg.in/yaml.v3 v3.0.1
	pgregory.net/rapid v1.3.0
)

// Celestial and celestial-data review candidates come from the review proxy.
// Publish those dependencies and pin releases before releasing Slayer.

tool go.uber.org/mock/mockgen
