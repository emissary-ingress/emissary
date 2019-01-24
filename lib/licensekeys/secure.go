package licensekeys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

func ParseKey(licenseKey string) (jwt.MapClaims, *jwt.Token, error) {
	var claims jwt.MapClaims
	token, err := jwt.ParseWithClaims(licenseKey, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("1234"), nil
	})
	return claims, token, err
}

func PhoneHome(claims jwt.MapClaims, component, version string) error {
	id := fmt.Sprintf("%v", claims["id"])
	space, err := uuid.Parse("a4b394d6-02f4-11e9-87ca-f8344185863f")
	if err != nil {
		panic(err)
	}
	install_id := uuid.NewSHA1(space, []byte(id))
	data := make(map[string]interface{})
	data["application"] = "ambassador-pro"
	data["install_id"] = install_id.String()
	data["version"] = version
	data["metadata"] = map[string]string{
		"id":        id,
		"component": component,
	}
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	_, err = http.Post("https://kubernaut.io/scout", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	return nil
}
