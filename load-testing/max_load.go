package main

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	rate := 100
	okRate := 1
	var nokRate int
	url := "http://ambassador.nkrause.k736.net:31541/http-echo/"

	// first, find the point at which the system breaks
	for {
		cmdRate := "-rate=" + strconv.Itoa(rate)
		fmt.Printf("Attacking with rate %s\n", cmdRate)
		out, e := exec.Command("./test_rate", cmdRate, "-url="+url).Output()
		if e != nil {
			log.Fatal("exec error: ", e)
		}
		stringRate := string(out)

		if strings.Compare(stringRate, "1.000000") == 0 {
			okRate = rate
			fmt.Printf("!!!  Success at %d req/sec\n", rate)
			rate *= 2
		} else {
			nokRate = rate
			fmt.Printf(":(  Failed at %d req/sec\n", rate)
			break
		}
	}

	// next, do a binary search between okRate and nokRate
	for (nokRate - okRate) > 1 {
		rate = (nokRate + okRate) / 2
		cmdRate := "-rate=" + strconv.Itoa(rate)
		fmt.Printf("Attacking with rate %s\n", cmdRate)
		out, e := exec.Command("./test_rate", cmdRate, "-url="+url).Output()
		if e != nil {
			log.Fatal("exec error: ", e)
		}
		stringRate := string(out)

		if strings.Compare(stringRate, "1.000000") == 0 {
			okRate = rate
			fmt.Printf("!!!  Success at %d req/sec\n", rate)
		} else {
			nokRate = rate
			fmt.Printf(":(  Failed at %d req/sec\n", rate)
		}
	}
	fmt.Printf("➡️  Maximum Working Rate: %d req/sec\n", okRate)
}
