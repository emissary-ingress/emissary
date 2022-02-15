package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func Main(reader io.Reader, idx, num int) error {
	var lines []string

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	fenceposts := make([]int, len(lines)+1)
	fenceposts[0] = 0
	fenceposts[len(fenceposts)-1] = len(lines)
	for i := 1; i < len(fenceposts)-1; i++ {
		fenceposts[i] = int(float64(i*len(lines)) / float64(num))
	}

	output := lines[fenceposts[idx-1]:fenceposts[idx]]
	fmt.Println(strings.Join(output, " or "))
	return nil
}

func errUsage(err error) {
	fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
	fmt.Fprintf(os.Stderr, "Usage: %s I N <list.txt\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Where I and N are positive integers, and I is in [1,N]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Ex: %s 1 3 <list.txt\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Ex: %s 2 3 <list.txt\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Ex: %s 3 3 <list.txt\n", os.Args[0])
	os.Exit(2)
}

func main() {
	if len(os.Args) != 3 {
		errUsage(fmt.Errorf("takes exactly 2 arguments, got %d", len(os.Args)-1))
	}
	i, err := strconv.Atoi(os.Args[1])
	if err != nil {
		errUsage(err)
	}
	n, err := strconv.Atoi(os.Args[2])
	if err != nil {
		errUsage(err)
	}
	if i < 1 || n < 1 || i > n {
		errUsage(fmt.Errorf("integers out of bounds"))
	}
	if err := Main(os.Stdin, i, n); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}
