# the mock of finish's titled, made by uber's mockgen: a title of any
# kind, so the kinds a job cannot hold can be tried.
go tool mockgen -source=titles.go -destination=mocks/titled.go \
	-package mocks -mock_names titled=Titled
