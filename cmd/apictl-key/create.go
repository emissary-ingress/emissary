package main

import (
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/cobra"
)

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
	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), &License{
		Id: customerID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
		},
	})
	tokenstring, err := token.SignedString([]byte("1234"))
	if err != nil {
		log.Fatalln(err)
	}
	return tokenstring
}
