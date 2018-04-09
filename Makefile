all: mountpoint/ambex

vendor:
	glide install

ambex: mountpoint/ambex

mountpoint/ambex: vendor main.go
	sh -c "export GOOS=linux; go install ./..."
	cp $(HOME)/go/bin/linux_amd64/ambex mountpoint

clean:
	rm -rf vendor ambex mountpoint/ambex
