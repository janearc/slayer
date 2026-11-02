# the mock of the hud's Blocker, made by uber's mockgen: a scene that is
# told what ray it will be asked about and what it answers.
go tool mockgen -destination mocks/blocker.go -package mocks \
	github.com/janearc/slayer/hud Blocker
