package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	_ "github.com/lyft/protoc-gen-validate/tests/harness/cases/gogo"
	_ "github.com/lyft/protoc-gen-validate/tests/harness/cases/other_package/gogo"
	harness "github.com/lyft/protoc-gen-validate/tests/harness/gogo"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	checkErr(err)

	tc := new(harness.TestCase)
	checkErr(proto.Unmarshal(b, tc))

	da := new(types.DynamicAny)
	checkErr(types.UnmarshalAny(tc.Message, da))

	msg := da.Message.(interface {
		Validate() error
	})
	checkValid(msg.Validate())

}

func checkValid(err error) {
	if err == nil {
		resp(&harness.TestResult{Valid: true})
	} else {
		resp(&harness.TestResult{Reason: err.Error()})
	}
}

func checkErr(err error) {
	if err == nil {
		return
	}

	resp(&harness.TestResult{
		Error:  true,
		Reason: err.Error(),
	})
}

func resp(result *harness.TestResult) {
	if b, err := proto.Marshal(result); err != nil {
		log.Fatalf("could not marshal response: %v", err)
	} else if _, err = os.Stdout.Write(b); err != nil {
		log.Fatalf("could not write response: %v", err)
	}

	os.Exit(0)
}
