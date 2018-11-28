package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	flag.Parse()
	for _, arg := range flag.Args() {
		entries, err := parse(arg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(entries)
	}
}

type Entry struct {
	Key   string
	Value string
}

func parse(pattern string) (result []Entry, err error) {
	parts := strings.Split(pattern, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if len(part) == 0 {
			err = fmt.Errorf("empty path component: %s", pattern)
			return
		}

		re := regexp.MustCompile(`{([a-zA-Z]+[a-zA-Z0-9]*):([^}]+)}`)

		keyval := re.FindStringSubmatch(part)
		var key, val string
		if keyval != nil {
			if len(keyval[0]) != len(part) {
				err = fmt.Errorf("invalid pair: %s", part)
				return
			}
			key = keyval[1]
			val = keyval[2]
		} else {
			key = "generic_key"
			val = part
		}
		result = append(result, Entry{key, val})
	}

	return
}
