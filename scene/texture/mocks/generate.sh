# the mock of the texture package's Footprinted, made by uber's mockgen: a
# paint that is told what point and what pixel width it will be asked about
# and what it answers. Run from scene/texture.
go tool mockgen -destination mocks/footprinted.go -package mocks \
	github.com/janearc/slayer/scene/texture Footprinted
