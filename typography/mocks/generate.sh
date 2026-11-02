# the mock of typography's Metrics, made by uber's mockgen: a font that
# kerns, for the layout's tests, since go's own fonts kern nothing.
go tool mockgen -destination mocks/metrics.go -package mocks \
	github.com/janearc/slayer/typography Metrics
