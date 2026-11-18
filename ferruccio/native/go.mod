module github.com/janearc/slayer/ferruccio/native

go 1.27.3

require (
	github.com/gogpu/gg v0.52.7
	github.com/gogpu/gogpu v0.54.1
	github.com/gogpu/ui v0.1.55
	github.com/janearc/chroma v0.0.0-20261008193048-284aab1761ca
	github.com/janearc/puzzlemouth v0.1.4
	github.com/janearc/slayer v0.0.0
	github.com/janearc/slayer/ferruccio v0.0.0
	golang.org/x/image v0.46.0
)

require (
	github.com/coregx/signals v0.1.1 // indirect
	github.com/go-webgpu/goffi v0.6.8 // indirect
	github.com/go-webgpu/webgpu v0.5.5 // indirect
	github.com/gogpu/gpucontext v0.31.3 // indirect
	github.com/gogpu/gputypes v0.8.2 // indirect
	github.com/gogpu/naga v0.21.5 // indirect
	github.com/gogpu/wgpu v0.34.10 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/janearc/celestial v0.0.9 // indirect
	github.com/janearc/libtheme-css v0.4.6 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	go.etcd.io/bbolt v1.5.0 // indirect
	go.uber.org/mock v0.6.0
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/term v0.47.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/janearc/slayer/ferruccio => ..

replace github.com/janearc/slayer => ../..

replace github.com/gogpu/gogpu => ./third_party/gogpu
