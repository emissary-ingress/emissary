get:
	go get -t -d

test: *.go get
	scripts/test.sh

cover: coverage.out
	go tool cover -html=coverage.out -o coverage.html
