// Code generated by go-bindata. DO NOT EDIT.
// sources:
// bindata/aes-host.js
// bindata/index.html

package firstboot


import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}


type asset struct {
	bytes []byte
	info  fileInfoEx
}

type fileInfoEx interface {
	os.FileInfo
	MD5Checksum() string
}

type bindataFileInfo struct {
	name        string
	size        int64
	mode        os.FileMode
	modTime     time.Time
	md5checksum string
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) MD5Checksum() string {
	return fi.md5checksum
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _bindataAeshostjs = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xe4\x57\x5b\x6f\xdb\x3a\x12\x7e\x96\x7e\xc5\x54\x2f\xb1\x51\x47\xea\xf6" +
	"\xd1\x96\x0d\x04\x85\x77\x5b\xc0\xc1\x06\x4e\xb3\xdb\x7d\x6a\x69\x69\x64\x71\x4b\x91\x32\x2f\x4e\x8c\xd4\xff\x7d" +
	"\x41\x52\xf2\x45\xb6\x9b\x74\xd1\x83\xf3\x70\x80\x00\x91\x34\xc3\x6f\xee\xdf\xd0\xb4\xaa\x85\xd4\xf0\x2f\x83\x50" +
	"\x48\x51\xc1\x55\xa9\x75\xad\x86\x49\x92\xe5\x3c\xfe\xaf\xca\x91\xd1\xb5\x8c\x39\xea\x84\xd7\x55\xb2\x36\x98\xe4" +
	"\x54\x69\xfb\x10\xa3\xaa\xe2\x85\x14\x8f\x0a\x65\x5c\x51\xab\x7d\x35\x0a\x43\x7c\x72\x80\x39\x16\xc4\x30\x07\x1c" +
	"\xe3\x93\x46\x9e\xf7\x9e\xc3\x40\x63\x55\x33\xa2\x71\x08\xdf\xd2\x9c\xae\x27\x61\x5a\x08\x59\xc1\xfa\x5a\xf0\xa1" +
	"\x32\x8b\x8a\xea\xb8\x96\xb8\x46\xae\xc7\x91\xe0\xf7\xee\x4b\x34\x09\x83\x20\x2d\x28\xb2\x5c\xa1\xb6\x2f\x41\xca" +
	"\x70\x89\x3c\x9f\x7c\xe2\x54\x53\xc2\xe0\xf3\xec\x1e\x14\x6a\x53\xa7\x49\x23\xf1\x5a\x64\x81\xcc\x3d\x06\xa9\xaa" +
	"\x09\x9f\x7c\x14\x4a\x73\x52\x21\x68\x01\x12\x57\x06\x95\x06\x02\x19\x4a\x4d\x0b\x9a\x11\x8d\x50\x08\x39\x4c\x13" +
	"\xa7\xec\xcf\x39\x2f\x03\xf7\x48\x79\x6d\x34\xe8\x4d\x8d\xe3\x48\xe3\x93\x8e\xc0\x62\x8d\xa3\xb2\x41\x8d\x60\x7d" +
	"\x5d\x89\x1c\xd9\x70\x4d\x98\x39\x14\x78\x80\xc0\x85\xe9\x50\x6c\x74\xad\x33\x1f\x4a\xc2\x97\x18\x41\xd2\xda\xa9" +
	"\x61\x7d\x4d\x8b\x71\xf4\xa6\x3c\xd2\xc8\xa3\xc9\xbf\x11\x88\xd1\xe2\xba\xa0\x8c\x61\x0e\xba\xa4\xca\x57\x4d\x97" +
	"\x08\x0f\xf3\x59\x63\x67\x23\x0c\x18\x65\x15\x04\x90\x2c\x43\xa5\xbc\xea\x23\x2e\xa0\x26\x4b\x8c\x01\xfe\x8e\xc8" +
	"\xa0\x90\xe8\x52\x91\x39\x7c\xa0\x7a\x00\xba\x14\x66\x59\xc6\x69\x52\x37\xf1\x27\x6d\x02\xd2\x64\x9f\xce\xd3\xcc" +
	"\xde\x7c\xb8\x9d\x42\x2d\xc5\x9a\xe6\xd8\xc9\xe0\x61\xda\x8c\x64\x6d\xd6\x5a\xe5\x93\xac\xed\x04\xe1\x99\xa4\xdd" +
	"\x35\xc2\xe3\xa4\x5d\xf4\xed\xd0\x76\x56\x62\xf6\x7d\x21\x9e\x5a\x07\xb4\x50\x5f\xc9\x52\xe2\x69\xdd\xb4\x50\x37" +
	"\xad\x60\x41\x79\x3e\xcc\xa9\x22\x0b\x86\xb9\x13\xcd\x71\x05\x3f\x7e\x80\x16\xaa\x75\x06\xde\x8c\xc7\xbb\xe0\x1b" +
	"\xd9\x54\xda\xcf\xc0\x0d\x63\xbb\xd2\xda\x66\x6a\x6a\xeb\x71\xa2\x5d\xc9\x27\x33\x41\x72\xca\x97\xa0\x6a\xca\x39" +
	"\x4a\x58\x0a\x54\x50\xa2\xc4\x33\xa5\x68\x81\x90\x29\x6c\xd1\xa6\x52\x1e\xa0\xa5\xab\xc9\xf3\xf3\x91\x8b\xdb\x6d" +
	"\x9a\xac\x26\x90\x5b\x58\x2e\x34\x90\xba\x46\x22\xfd\x20\x14\xe8\x1e\xd6\x84\xd1\x1c\x8e\x2b\x09\x3b\x24\x1b\x90" +
	"\x07\x89\x21\x5d\x18\xad\x05\xf7\x73\x9b\x31\x9a\x7d\x3f\x53\x9b\xc9\x1c\xb5\xdc\xa4\x89\xd7\x9d\xbc\x10\xc7\xde" +
	"\xf7\x4f\x50\x92\x35\x82\x2b\x8d\x6b\x61\xfb\x57\x22\x7c\x46\x59\x29\x10\x05\xdc\xa3\x5c\xd3\x0c\x81\x68\x48\x49" +
	"\x5b\xa2\x52\xa2\x4f\xc4\xc3\x7c\x16\x35\x2e\x3f\xcc\x67\xce\x65\x32\xf9\xf5\x86\x9e\x56\x84\xb2\xcb\x8d\x8c\x56" +
	"\xdc\x76\x52\xf3\xd2\xe9\x22\xff\xf5\xb4\x89\x4d\x9d\x13\x8d\xff\xb9\xb9\x9d\x9d\x69\xdf\x34\xd9\x13\x9d\x7d\xad" +
	"\x25\xda\x60\x36\xa4\x62\x2e\x14\xfb\x1e\x76\x7c\xf1\xc4\x19\x41\x63\xf7\xa6\xae\xd9\xc6\x43\xa7\x89\xa5\x57\xfb" +
	"\x60\x83\x7e\x7e\x06\x61\xb4\x3d\x67\x91\x3c\xfd\xba\x7f\xdf\x06\x61\x90\x13\x4d\x86\x50\x18\x9e\x69\x2a\x78\xaf" +
	"\x0f\xcf\x61\x10\x48\xd4\x46\x72\xf7\x18\xb4\x64\x34\x84\x47\xca\x73\xf1\x18\x33\x91\x11\xab\x1b\xb7\x92\x81\x55" +
	"\xdb\x77\x4e\xd4\x6e\x12\x92\x55\x78\xad\x34\x59\x52\xbe\xbc\x5e\xbf\x7b\x1f\x93\x9a\xc6\x0c\xb5\x42\x9e\xc9\x4d" +
	"\xad\x63\x21\x97\x49\x4e\x25\x66\x5a\xc8\x4d\x34\x80\x24\xe9\x1c\x7e\xc5\x21\x6b\xbb\x9d\xdb\x21\x14\x84\x29\xef" +
	"\x8f\xab\xc2\x10\xa2\x68\x10\x1e\x46\xd1\x50\xea\x4e\xb3\x39\x3e\xc7\xd5\xd0\xcd\x6c\x8b\x37\x95\xf2\xf8\xc3\xdd" +
	"\x3e\xbe\x9d\xd1\x87\xf9\xac\x55\xb2\x5f\x6c\xb1\x8e\x81\xec\x97\xbd\x0f\xbe\x08\x2d\xc0\x76\x14\x06\xdb\x41\x18" +
	"\x64\xa2\xaa\x8d\xb6\x2e\x3d\xdb\xd7\x05\x16\x42\xe2\xad\x30\x5c\x9f\x94\xc5\x92\x79\xdc\x9d\xb7\x9e\xb5\xd6\x6f" +
	"\xc0\x2a\xd4\xa5\xc8\xd5\xd0\xa9\x77\x57\xcd\x01\x9e\x5b\xb5\x1e\xd4\xa3\x76\xf2\x03\x63\xd0\xd2\xe0\x68\x27\xdf" +
	"\x37\x6f\xcf\xda\x72\xc6\x82\xae\x2b\x17\x0c\x30\xd4\x60\x24\x83\x31\x70\x7c\xb4\xeb\xaa\x77\xa5\x85\xba\x36\x92" +
	"\x5d\x0d\xba\x4d\xe5\xc0\x03\x23\x59\xac\x90\xc8\xac\xbc\x23\x92\x54\x2a\x56\xa8\x7b\x57\x19\x69\xce\x38\x8f\xda" +
	"\x86\xeb\x8f\xc2\xd6\xc8\x8e\x8a\xc7\xc7\x2a\x23\xdb\x59\x19\xa9\xb5\x91\x6e\xd3\x3b\x5a\xc9\x98\x50\x46\xa2\x82" +
	"\x05\x32\xf1\xb8\xc3\x90\xb8\x6a\x1c\xfd\x72\x3b\xfb\xa8\x75\x3d\xf7\xf7\x05\x1f\x76\x20\x71\x15\x8b\x1a\x79\x2f" +
	"\xfa\xc7\xf4\x73\x34\xb0\x71\xc5\x5a\xdc\x6b\x49\xf9\xb2\xd7\x3f\xd0\xe1\x4c\x10\x9b\xc6\x5e\x1f\xc6\x13\x9f\x87" +
	"\x80\x16\xd0\xb3\x42\xa5\x89\x36\x0a\xc6\x63\x78\xff\xee\x5d\x93\xa4\x26\xcf\x0d\x7b\x8d\xad\x1f\xb1\x44\x55\x0b" +
	"\xae\x7c\x19\xf6\x0a\x96\x91\xfd\x86\xf1\x82\x2d\x58\x22\x3d\x0f\xb3\xd7\xea\x1c\x3f\xc5\xdf\x86\x87\x5a\x77\xfb" +
	"\x5c\xee\xd2\x78\xa4\x30\xf7\x89\x6a\xf1\xb7\x07\xa1\xa3\x94\x42\x76\x62\xbf\xe4\xd6\xb1\x57\xd1\x97\x8a\x1d\x24" +
	"\x1d\x1c\x52\x34\xfa\x0d\x9e\xd9\xdc\x1f\x2a\x34\x3b\xba\xdf\x71\x6f\x8e\xab\x98\x2c\x84\x6c\xeb\xbd\x0d\x4f\x80" +
	"\x25\xae\x46\x87\x5f\x1d\xf5\xc0\xd8\x33\xca\x2e\x0b\xca\xde\x7c\xfb\x2f\x0c\x90\xc4\x42\xa2\x2a\xef\x5d\x3f\x9c" +
	"\x8c\xfb\xb9\xc1\xf1\xad\xf3\x6b\x73\x63\x67\xbb\x9d\x9a\x76\xce\x0f\xa6\xe6\x4f\xee\xf8\x66\x35\x5d\xec\xc8\x6d" +
	"\xc7\x02\xf2\xae\x91\x24\x01\x89\x99\x91\x0a\x07\xee\x56\x2b\xb8\xa6\xdc\x10\xc6\x36\xd0\x24\xd8\x0d\x7c\x63\xc8" +
	"\xbb\xb2\x2f\xfa\x51\x0d\xda\xaa\x9f\x29\x63\xc3\x78\xfe\xb7\xc9\xeb\x99\xce\xae\x80\x5f\x2f\x97\x2d\xd1\x99\x92" +
	"\x5d\x3c\x63\xb7\xe5\x57\x62\x74\x29\x24\xd5\x9b\x33\x14\xf9\xf3\x93\x6e\x59\xb6\xa7\xdc\xcb\xff\xd3\x1f\x77\xff" +
	"\xbc\xff\x2d\x0d\xf2\xb7\x0b\x0d\xe2\x6f\x38\xf6\xb2\x6c\xc7\x28\x8e\xe3\xe8\x90\xd9\xce\xd5\xf1\x1c\x35\x36\x70" +
	"\x6f\xc7\x10\x4d\x1d\x4f\x91\x43\xd4\x21\x44\x6f\x5f\xd1\x88\x17\x19\xee\x65\xf4\x0e\xc5\x4d\xf7\x0c\x77\xb1\xeb" +
	"\xf6\xdc\xf1\x2a\x8e\xf8\x0b\xb5\xdc\x1f\x43\x49\xee\xd6\xfd\xc2\x8a\x6c\x2e\x7b\x3f\x59\x34\xad\xc6\xb9\x4d\xd3" +
	"\xc8\x2e\xac\x9a\x3d\x76\xbb\x6b\x4e\x9b\x62\x3b\x08\xb7\xfd\x51\xf8\xbf\x00\x00\x00\xff\xff\x54\x2c\x48\x89\xc7" +
	"\x11\x00\x00")

func bindataAeshostjsBytes() ([]byte, error) {
	return bindataRead(
		_bindataAeshostjs,
		"aes-host.js",
	)
}



func bindataAeshostjs() (*asset, error) {
	bytes, err := bindataAeshostjsBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "aes-host.js",
		size: 4551,
		md5checksum: "",
		mode: os.FileMode(420),
		modTime: time.Unix(1, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}

var _bindataIndexhtml = []byte(
	"\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x53\xdd\x6a\xdb\x4c\x10\xbd\x56\x9e\x62\x3e\x7d\x04\x27\xa5\x92\x9c" +
	"\x92\x42\x90\x65\x41\x68\x03\xbd\x6b\x21\xa5\x50\x4a\x2e\x46\xda\xb1\xb5\xcd\xfe\xb1\x3b\xf2\x4f\x4d\xde\xbd\x48" +
	"\x2b\x87\x84\xd6\x50\xdd\xec\x68\xe6\x9c\xa3\xd1\xd1\x51\xf5\xdf\xc7\xcf\x1f\xbe\x7e\xff\x72\x07\x1d\x6b\x55\x9f" +
	"\x55\xc3\x01\x0a\xcd\x7a\x99\x92\x49\xeb\xb3\xa4\xea\x08\x45\x7d\x96\x24\x95\x26\x46\x68\x3b\xf4\x81\x78\x99\xf6" +
	"\xbc\xca\x6e\xd2\x71\xc0\x92\x15\xd5\xb7\xba\xc1\x10\x50\x58\x0f\x77\x62\x4d\x70\xcf\xd8\x3e\x42\x06\x2b\xe9\x03" +
	"\x43\x63\x2d\x57\x45\x44\x0e\x9c\xc0\xfb\x58\x25\x2b\x49\x4a\x04\x62\x78\x79\x1d\x40\xc8\xe0\x14\xee\x4b\x60\x6c" +
	"\x14\x2d\xe0\xe9\x15\xb6\x06\x85\x0d\xa9\xbf\x62\x33\x6f\xb7\xa7\xf0\x35\xbc\xf9\x13\xdf\x92\x52\x03\x61\x60\x0c" +
	"\xe3\xc6\xee\xb2\x20\x7f\x49\xb3\x2e\xa1\xb1\x5e\x90\xcf\x1a\xbb\x3b\x25\x29\x8d\xeb\xf9\x07\xef\x1d\x2d\x53\xa6" +
	"\x1d\xa7\x0f\x70\x00\x2d\x4d\xb6\x95\x82\xbb\x12\xae\xe7\xa4\xff\x85\xdb\x7b\x95\x3e\xc0\x49\xee\x20\x8d\x9e\x10" +
	"\x0e\xb0\xb2\x86\xb3\x15\x6a\xa9\xf6\x25\x68\x6b\x6c\x70\xd8\x1e\x2d\xfa\x1f\x9d\x83\xc3\x50\x25\x1a\x77\x47\xa1" +
	"\x1b\x69\x16\x53\xcf\xaf\xa5\x29\x01\x7b\xb6\x63\x67\x24\x39\x4f\x13\xa7\xc1\xf6\x71\xed\x6d\x6f\x44\x09\x8d\xc2" +
	"\xf6\x31\xd2\x5a\xab\xac\x2f\x61\xdb\x49\xa6\xd8\x99\x84\x5b\x54\xed\xc5\xd5\x7c\x7e\x0e\x19\x5c\x93\xbe\x3c\x7f" +
	"\xfd\x98\x39\xbc\x23\x1d\x5b\x0e\x85\x18\x2d\x9d\xe7\xef\x8f\xbd\xe8\x6e\x09\xc1\x2a\x29\xe0\xca\xed\x60\xed\x89" +
	"\xcc\xf3\x62\x55\x71\xcc\x49\x55\x4c\x39\xac\x1a\x2b\xf6\x63\x84\x84\xdc\x80\x14\xcb\x14\x9d\x1b\x73\x98\x54\x48" +
	"\x21\xeb\x6c\xe0\xba\x2a\x9e\xcb\x51\x45\xc8\x4d\x4c\x5d\xeb\xa5\x63\x88\x86\x6b\x2b\x7a\x45\x91\x2a\xb5\xb3\x9e" +
	"\xe1\x5b\x4f\xb0\xf2\x56\xc3\xac\x63\x76\xa1\x2c\x8a\x56\x98\xfc\x67\x10\xa4\xe4\xc6\xe7\x86\xb8\x30\x4e\x17\x9b" +
	"\x9e\x0a\x21\x03\x0f\x45\x4e\x41\xe7\x8d\xb7\xdb\x40\x3e\xd7\x72\x40\xcf\x16\x2f\x24\x6f\xef\xee\x3f\xd9\xc0\x93" +
	"\x6c\xfe\xbc\x58\xc4\x0d\xc0\x0d\x7a\xd8\x68\x58\x82\xa1\xed\xb0\xc1\xc5\x61\xb2\x5c\x3b\x6b\xc8\x70\x28\xa7\x6f" +
	"\x93\xcc\x8e\xe4\x59\x79\xd4\x7d\x3b\x4e\x9e\xe2\x41\xaa\x84\xd9\x10\x80\xd9\x78\xff\x74\xb9\x88\x1e\x8e\x6f\x3d" +
	"\x9a\x18\xcd\xab\x8a\xf8\xb7\xff\x0e\x00\x00\xff\xff\xc5\x63\xff\x44\xfe\x03\x00\x00")

func bindataIndexhtmlBytes() ([]byte, error) {
	return bindataRead(
		_bindataIndexhtml,
		"index.html",
	)
}



func bindataIndexhtml() (*asset, error) {
	bytes, err := bindataIndexhtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{
		name: "index.html",
		size: 1022,
		md5checksum: "",
		mode: os.FileMode(420),
		modTime: time.Unix(1, 0),
	}

	a := &asset{bytes: bytes, info: info}

	return a, nil
}


//
// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
//
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
// nolint: deadcode
//
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

//
// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or could not be loaded.
//
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

//
// AssetNames returns the names of the assets.
// nolint: deadcode
//
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

//
// _bindata is a table, holding each asset generator, mapped to its name.
//
var _bindata = map[string]func() (*asset, error){
	"aes-host.js": bindataAeshostjs,
	"index.html":  bindataIndexhtml,
}

//
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
//
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, &os.PathError{
					Op: "open",
					Path: name,
					Err: os.ErrNotExist,
				}
			}
		}
	}
	if node.Func != nil {
		return nil, &os.PathError{
			Op: "open",
			Path: name,
			Err: os.ErrNotExist,
		}
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}


type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{Func: nil, Children: map[string]*bintree{
	"aes-host.js": {Func: bindataAeshostjs, Children: map[string]*bintree{}},
	"index.html": {Func: bindataIndexhtml, Children: map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	return os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
