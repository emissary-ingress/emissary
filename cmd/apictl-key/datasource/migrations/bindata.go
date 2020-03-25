package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __1_initial_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8e\x31\x6b\xc3\x30\x10\x46\x77\xff\x8a\x23\x53\x02\xed\xd6\xad\x93\x9a\x5c\xa8\xa8\x2d\x07\xf9\x44\x9d\x2e\x42\x58\x47\xd1\x10\x3b\x95\xe4\xfe\xfe\x52\xbb\x35\x21\x81\x68\x13\xbc\xef\xdd\xdb\x6a\x14\x84\x80\x2d\xa1\x6a\x64\xad\x40\xee\x41\xd5\x04\xd8\xca\x86\x1a\x58\x8d\x63\xf0\x8f\x43\x4a\xe7\xd5\x73\x51\xfc\xc1\x24\x5e\x4a\x04\xc7\xc9\xfa\xe1\xe4\x42\x9f\x60\x5d\x00\x00\x04\x0f\xb7\xcf\x18\xb9\x83\x83\x96\x95\xd0\x47\x78\xc3\x23\xec\x70\x2f\x4c\x49\xf0\x6b\xb6\x9f\xdc\x73\x74\x99\xed\xf7\xd3\x7a\xf3\x30\x59\xba\xc8\x2e\x87\xa1\xb7\x39\x9c\xf8\xdf\x42\xb2\xc2\x86\x44\x75\x80\x77\x49\xaf\xd3\x17\x3e\x6a\x85\x53\xab\x32\x65\xb9\x68\xb7\x46\x6b\x54\x64\x97\xc5\x6c\x9d\x4b\xaf\xda\x08\x5b\x5a\x0c\x33\x17\xce\xd6\x79\x1f\x39\xa5\x0b\x4e\x2a\xbc\xe6\x22\x7f\x8d\x9c\x32\x47\x7b\xb1\xb8\xcb\x75\x43\x9f\x5d\x97\x6f\xee\x16\x9b\xe2\x27\x00\x00\xff\xff\xa0\xbb\xc6\x4b\x85\x01\x00\x00")

func _1_initial_up_sql() ([]byte, error) {
	return bindata_read(
		__1_initial_up_sql,
		"1_initial.up.sql",
	)
}

var __2_domains_hostname_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\xf4\x09\x71\x0d\x52\x08\x71\x74\xf2\x71\x55\x48\x4c\x2d\x8e\x4f\xc9\xcf\x4d\xcc\xcc\x2b\x56\x80\x88\x3b\xfb\xfb\x84\xfa\xfa\x29\x64\x16\xc4\x27\xa6\xa4\x14\xa5\x16\x17\x2b\xb8\x04\xf9\x07\x28\xf8\xf9\x87\x28\xf8\x85\xfa\xf8\x58\x73\x71\xe1\xd4\xef\xe2\x02\xd3\x9d\x91\x5f\x5c\x92\x97\x98\x9b\xaa\x10\xe2\x1a\x11\x62\xcd\x05\x08\x00\x00\xff\xff\x75\xbd\x19\x86\x72\x00\x00\x00")

func _2_domains_hostname_up_sql() ([]byte, error) {
	return bindata_read(
		__2_domains_hostname_up_sql,
		"2_domains_hostname.up.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"1_initial.up.sql":          _1_initial_up_sql,
	"2_domains_hostname.up.sql": _2_domains_hostname_up_sql,
}

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
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"1_initial.up.sql":          &_bintree_t{_1_initial_up_sql, map[string]*_bintree_t{}},
	"2_domains_hostname.up.sql": &_bintree_t{_2_domains_hostname_up_sql, map[string]*_bintree_t{}},
}}
