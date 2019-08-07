package main

import (
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/cobra"
)


var (
	ID         string
	EXPIRATION int
)

func init() {
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a jwt token",
		Run: func(cmd *cobra.Command, args []string) {
			// for example, server receive token string in request header.
			tokenstring := createTokenString()
			// This is that token string.
			fmt.Println(tokenstring)
		},
	}
	create.Flags().StringVarP(&ID, "id", "i", "", "id for key")
	create.Flags().IntVarP(&EXPIRATION, "expiration", "e", 14, "expiration from now in days (can be negative for testing)")
	create.MarkFlagRequired("id")
	create.MarkFlagRequired("expiration")

	argparser.AddCommand(create)
}

type License struct {
	Id string `json:"id"`
	jwt.StandardClaims
}

func createTokenString() string {
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), &License{
		Id: ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(EXPIRATION) * 24 * 60 * 60 * time.Second).Unix(),
		},
	})
	tokenstring, err := token.SignedString([]byte("1234"))
	if err != nil {
		log.Fatalln(err)
	}
	return tokenstring
}
