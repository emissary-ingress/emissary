package main

// TODO:
// - extract and inject useful jwt fields and userinfo
// - wire in config source for policy
// - caching of clientCredentials
// - proper caching of pemCert
// - packaging
// - docs
// - testing
// - additional flows
// - maybe make domain and audience part of policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"errors"
	"log"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/joho/godotenv"
)

type Response struct {
	Message string `json:"message"`
}

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N string `json:"n"`
	E string `json:"e"`
	X5c []string `json:"x5c"`
}

// XXX: this implementation will be replaced by some dynamically
// reloadable configuration source, probably CRDs. The first return
// result specifies whether authentication is required, the second
// return result specifies which scopes are required for access.
func policy(method, host, path string) (bool, []string) {
	if path == "/backend/public/" {
		return true, []string{}
	} else if path == "/backend/private/" {
		return false, []string{}
	} else if path == "/backend/private-scoped/" {
		return false, []string{"read:messages"}
	} else {
		return false, []string{}
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Print("Error loading .env file")
	}

	common := negroni.Classic()
	common.UseFunc(func (rw http.ResponseWriter, rq *http.Request, next http.HandlerFunc) {
		client_id := rq.Header.Get("Client-Id")
		client_secret := rq.Header.Get("Client-Secret")
		if client_id != "" {
			auth, err := clientCredentials(client_id, client_secret)
			if err != nil {
				log.Println(err)
			} else {
				rq.Header.Set("Authorization", fmt.Sprintf("%s %s", auth.Type, auth.Token))
			}
		}
		next(rw, rq)
	})

	jwtMware := jwtmiddleware.New(jwtmiddleware.Options {
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// Verify 'aud' claim
			aud := os.Getenv("AUTH0_AUDIENCE")
			checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
			if !checkAud {
				return token, errors.New("Invalid audience.")
			}
			// Verify 'iss' claim
			iss := "https://" + os.Getenv("AUTH0_DOMAIN") + "/"
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer.")
			}

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
		CredentialsOptional: true,
	})

	common.UseFunc(jwtMware.HandlerWithNext)

	common.UseHandlerFunc(func (rw http.ResponseWriter, rq *http.Request) {
		public, scopes := policy(rq.Method, rq.Host, rq.URL.Path)
		if !public {
			token, _ := rq.Context().Value(jwtMware.Options.UserProperty).(*jwt.Token)
			if token == nil {
				responseJSON("unauthorized", rw, rq, http.StatusUnauthorized)
				return
			} else {
				for _, scope := range scopes {
					if !checkScope(scope, token.Raw) {
						responseJSON("forbidden", rw, rq, http.StatusForbidden)
						return
					}
				}
			}
		}
		responseJSON("allowed", rw, rq, http.StatusOK)
	})

	fmt.Println("Listening on http://localhost:8080")
	http.ListenAndServe("0.0.0.0:8080", common)
}

type AuthRequest struct {
	ClientId string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience string `json:"audience"`
	GrantType string `json:"grant_type"`
}

type AuthResponse struct {
	Token string `json:"access_token"`
	Type string `json:"token_type"`
}

func clientCredentials(client_id, client_secret string) (auth AuthResponse, err error) {
	req := AuthRequest{
		ClientId: client_id,
		ClientSecret: client_secret,
		Audience: os.Getenv("AUTH0_AUDIENCE"),
		GrantType: "client_credentials",
	}
	body, err := json.Marshal(req)
	if err != nil { return }
	resp, err := http.Post("https://" + os.Getenv("AUTH0_DOMAIN") + "/oauth/token", "application/json",
		bytes.NewReader(body))
	if err != nil { return }
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		err = json.NewDecoder(resp.Body).Decode(&auth)
	} else {
		err = fmt.Errorf("%v", resp.Status)
	}
	return
}

type CustomClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
}

func checkScope(scope string, tokenString string) bool {
	token, _ := jwt.ParseWithClaims(tokenString, &CustomClaims{}, nil)

	claims, _ := token.Claims.(*CustomClaims)

	hasScope := false
	result := strings.Split(claims.Scope, " ")
	for i := range result {
		if result[i] == scope {
			hasScope = true
		}
	}

	return hasScope
}

var pemCert = ""

func getPemCert(token *jwt.Token) (string, error) {
	var err error
	if pemCert != "" {
		return pemCert, nil
	} else {
		pemCert, err = getPemCertUncached(token)
		return pemCert, err
	}
}

func getPemCertUncached(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get("https://" + os.Getenv("AUTH0_DOMAIN") + "/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k, _ := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("Unable to find appropriate key.")
		return cert, err
	}

	return cert, nil
}

func responseJSON(message string, w http.ResponseWriter, r *http.Request, statusCode int) {
	response := Response{message}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if statusCode == http.StatusOK {
		w.Header().Set("Authorization", r.Header.Get("Authorization"))
		if r.Header.Get("Client-Secret") != "" {
			w.Header().Set("Client-Secret", "-")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jsonResponse)
}
