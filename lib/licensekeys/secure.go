package licensekeys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"

	"github.com/datawire/apro/lib/jwtsupport"
)

func ParseKey(licenseKey string) (jwt.MapClaims, error) {
	// these details should match the details in apictl-key
	jwtParser := &jwt.Parser{ValidMethods: []string{"HS256"}} // HS256 is symmetric
	privateKey := []byte("1234")

	var claims jwt.MapClaims
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(licenseKey, &claims, func(token *jwt.Token) (interface{}, error) {
		return privateKey, nil
	}))
	return claims, err
}

func PhoneHome(claims jwt.MapClaims, component, version string) error {
	id := fmt.Sprintf("%v", claims["id"])
	space, err := uuid.Parse("a4b394d6-02f4-11e9-87ca-f8344185863f")
	if err != nil {
		panic(err)
	}
	data := map[string]interface{}{
		"application": "ambassador-pro",
		"install_id":  uuid.NewSHA1(space, []byte(id)).String(),
		"version":     version,
		"metadata": map[string]string{
			"id":        id,
			"component": component,
		},
	}
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	resp, err := http.Post("https://kubernaut.io/scout", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
