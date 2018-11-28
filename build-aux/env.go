package main

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var Verbose bool

type Environ map[string]string
type Config struct {
	Unversioned []string
	Profiles    map[string]Environ
}

func die(err error, args ...interface{}) {
	if err != nil {
		if args != nil {
			panic(fmt.Errorf("%v: %v", err, args))
		} else {
			panic(err)
		}
	}
}

func main() {
	verb := os.Getenv("VERBOSE")
	if verb != "" {
		var err error
		Verbose, err = strconv.ParseBool(verb)
		if err != nil {
			fmt.Printf("warning: unable to parse VERBOSE=%s as bool\n", verb)
		}
	}

	var output = flag.String("output", "", "output file")
	var input = flag.String("input", "", "input file")
	flag.Parse()

	in, err := os.Open(*input)
	defer in.Close()

	out, err := os.Create(*output)
	die(err)
	defer out.Close()

	profile := strings.ToLower(os.Getenv("PROFILE"))
	if profile == "" {
		profile = "dev"
	}
	bytes, err := ioutil.ReadAll(in)
	die(err)

	var config Config
	err = json.Unmarshal(bytes, &config)
	die(err)

	current, ok := config.Profiles[profile]
	if !ok {
		panic("no such profile: " + profile)
	}

	combined := make(map[string]string)

	for k, v := range config.Profiles["default"] {
		combined[k] = v
	}
	for k, v := range current {
		combined[k] = v
	}

	for k, v := range combined {
		out.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}

	out.WriteString(fmt.Sprintf("HASH=%x\n", hash(config.Unversioned)))
}

func versioned(path string, excludes []string) bool {
	for _, ex := range excludes {
		m, err := filepath.Match(ex, path)
		die(err)
		if m {
			return false
		}
	}
	return true
}

func hash(unversioned []string) []byte {
	standard, err := shell("git ls-files --exclude-standard")
	die(err)
	others, err := shell("git ls-files --exclude-standard --others")
	die(err)

	files := append(standard, others...)

	h := md5.New()
	for _, file := range files {
		if strings.TrimSpace(file) == "" {
			continue
		}
		if !versioned(file, unversioned) {
			if Verbose {
				fmt.Printf("skipping %s\n", file)
			}
			continue
		}
		if Verbose {
			fmt.Printf("hashing %s\n", file)
		}
		h.Write([]byte(file))
		info, err := os.Lstat(file)
		die(err)
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(file)
			die(err)
			h.Write([]byte("link"))
			h.Write([]byte(target))
		} else {
			h.Write([]byte("file"))
			f, err := os.Open(file)
			die(err, file)
			defer f.Close()
			_, err = io.Copy(h, f)
			die(err)
		}
	}

	return h.Sum(nil)
}

func shell(command string) ([]string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	str := string(out)
	lines := strings.Split(str, "\n")
	return lines, err
}
