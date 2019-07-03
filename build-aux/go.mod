module github.com/datawire/build-aux

go 1.12

require (
	github.com/datawire/teleproxy v0.0.0-00010101000000-000000000000 // indirect
	github.com/golangci/golangci-lint v0.0.0-00010101000000-000000000000 // indirect
)

// Pin versions of external commands
replace (
	github.com/datawire/teleproxy => github.com/datawire/teleproxy v0.3.16
	github.com/golangci/golangci-lint => github.com/golangci/golangci-lint v1.17.1
)
