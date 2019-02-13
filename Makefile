example-plugin.so: FORCE
	GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o $@ .

clean:
	rm -f -- *.so

.PHONY: FORCE clean
.DELETE_ON_ERROR:
