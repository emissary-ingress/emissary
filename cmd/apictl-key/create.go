package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/cobra"
)

var privKey, pubKey = func() (*rsa.PrivateKey, *rsa.PublicKey) {
	// privKeyPem was generated with `openssl genrsa`, which is (as of OpenSSL
	// 1.1.1c) assumed to have sane defaults.
	//
	// This is assumed to be:
	//  - start with the the raw RSA private key
	//  - DER-encode it with the ASN.1-schema for RSA private keys specified by PKCS#1
	//  - PEM-encode it
	privKeyPEM := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAoAt5rKRDI80mr6KFXufLLsiJf2g5cv5oau+aP2XxSZOAuNbi
Sipg/vvI6vkI5VFp8boR4ejs97n/IGAG7SfEjucVatk7HFYeWNLmLfm3pi8bCgtK
XTTZFJ8dWNQ9+/2HAOOq2AKVUinjAlAct6fIu8sggH6lKD2DFOg/XOZr+qhgvcOS
tfJXZlWuo62nyxfUw+53nLZo4U1k7HxzNJL8Y/iSORBUxEMf1YV/RKQYbMZ1jBFu
oQq74UKdFhIYn4ESt6uXOiemQy2w96Jr61bRlFV+WWxIy0lm4m5PCqeULkoO7o6F
/rUXOOCVBZzDS04FUG5EXfjAgwe+7NF46MyoiwIDAQABAoIBAHeGGmiEhF/IZovk
pBYspeFagtVT0RPAS9sQ9fFDAOAh+JASaw1YePf0sihJtAsOskCE5bYBbnfTwGU9
Ue7rNDBFBAm4Eh0nc6KCXsUXKcBCjb8Cj5bsUPLFofUlDOWUga54WK7ZvwqNnaus
iMXf2FnnaW6NJmrXBX4ntKp8q0eWKa+B5fYkPiVJ38ukYrlm9Ym/sJ0uNAl+UzEc
74+PblCjUbsUEVRCHvZkW/wt8qn+fuSYGZzMq7cFO+HwANk/nImhgypxt7/jkSrZ
ZVC9eadE0q5ewmaso8KD4Qk+RWuFAqb6NZiJY7dFFhdPEOUrOmAR2KVeBF6maMlg
ZIeU/gECgYEA0K2qoFczR2LnkdTl+lYgbqhNyn7DkTHu6pdivq1LGTBaR4yjZgSo
RTVWFwk0VeleWfeVfcUsIIjzlfLcu0SIRv89TqTImzt4L8Ma8YBUTzH2HBaAA9RZ
WJ/uPFI28Ug2qRJzvdftvFpscca6QysYytzEUzf6RSiCtRtONdR1oFsCgYEAxFaA
cIdddeUX40kb9ZtpeG6I2tOj6zpHJXnsF4yO8wSpPguMQCIA2imqlStmef+Xw1t0
1jh6jkU5FinsywVw22fEKeQt6DUtAUFcoRy/VmxsTahYOOKbTzSm4rLrlIl2flFI
k6pBzzDsaQ9SxCQ+xWXKnkKUkPZfgek7Yqvuj5ECgYEAhgEP0flNR6k+tYo2yOQn
3YectM1kfsfG+cSPN40G7bz8LHgmsauJ9y+CAjb58bVzzmhMCkDkzlvDuGYF0wf2
T0k2sFrnK7ArxNgQZEcZXOXjejQErvDdEylYjknpWFYcK8RaqO2Rj+OtQf7wu5Ng
T10ngZ0vzNtv3CcVuUGe64MCgYEAlhBzjQ65jYGzt2HKv/ewLn91lKPMlt7tQCSn
Ifypye8XGDglU2Np+VV9bxRD+B02NvfxHkb+zTz1fA5BUY9wChKOqWIhAGmcY2g8
z1u0lu65/MUd4SS6hlh88arFSruiWLvx2AN610zSdR5kKUx2udOqgTnsabwVlarZ
W/qDlTECgYEAj21WqgV4kkBV8HdNWtOkgp2h1LNmdyahFeUHMaLZHMhVdZUsVxVV
evuQEE+XHF5LfuoG52m4e/LXyB3kPf5GZNLD7agFVDJ2lZFEdx4RAFxwZpP0io5H
XVeDirHUsrZ7hC5rM/9SpLwOSjUB7Dp1ZdSzXDo2LJwtQWDI5Y/4/JY=
-----END RSA PRIVATE KEY-----`)

	privKeyDER, _ := pem.Decode(privKeyPEM)
	if privKeyDER == nil {
		panic("invalid PEM")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(privKeyDER.Bytes)
	if err != nil {
		panic(err)
	}
	return privKey, privKey.Public().(*rsa.PublicKey)
}()

func init() {
	var (
		argCustomerID   string
		argLifetimeDays int
	)
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a jwt token",
	}
	create.Flags().StringVarP(&argCustomerID, "id", "i", "", "id for key")
	create.Flags().IntVarP(&argLifetimeDays, "expiration", "e", 14, "expiration from now in days (can be negative for testing)")
	create.MarkFlagRequired("id")
	create.MarkFlagRequired("expiration")

	create.Run = func(cmd *cobra.Command, args []string) {
		expiresAt := time.Now().Add(time.Duration(argLifetimeDays) * 24 * time.Hour)
		tokenstring := createTokenString(argCustomerID, expiresAt)
		fmt.Println(tokenstring)
	}

	argparser.AddCommand(create)
}

type License struct {
	Id string `json:"id"`
	jwt.StandardClaims
}

func createTokenString(customerID string, expiresAt time.Time) string {
	token := jwt.NewWithClaims(jwt.GetSigningMethod("PS512"), &License{
		Id: customerID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
		},
	})
	tokenstring, err := token.SignedString(privKey)
	if err != nil {
		log.Fatalln(err)
	}
	return tokenstring
}

func init() {
	subcmd := &cobra.Command{
		Use:   "pubkey",
		Short: "Dump the public key that can be used to verify license keys",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("&rsa.PublicKey{\n\tN: newBigIntFromBytes(%#v),\n\tE: %#v,\n}\n", pubKey.N.Bytes(), pubKey.E)
		},
	}
	argparser.AddCommand(subcmd)
}
