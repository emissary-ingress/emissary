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
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	jw "github.com/auth0/go-jwt-middleware"
	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/gobwas/glob"
	"github.com/joho/godotenv"
	ms "github.com/mitchellh/mapstructure"
)

func init() {
	// If version is not set, make that clear.
}

type Response struct {
	Message string `json:"message"`
}

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type Rule struct {
	Host   string
	Path   string
	Public bool
	Scopes string
}

func match(pattern, input string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		log.Print(err)
		return false
	}
	return g.Match(input)
}

func (r Rule) match(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

var rules atomic.Value

func init() {
	rules.Store(make([]Rule, 0))
	log.Printf("initialized")
}

// The first return result specifies whether authentication is
// required, the second return result specifies which scopes are
// required for access.
func policy(method, host, path string) (bool, []string) {
	for _, rule := range rules.Load().([]Rule) {
		log.Printf("checking %v against %v, %v", rule, host, path)
		if rule.match(host, path) {
			return rule.Public, strings.Fields(rule.Scopes)
		}
	}

	return false, []string{}
}

var kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")

func main() {
	flag.Parse()
	go controller(*kubeconfig, func(uns []map[string]interface{}) {
		newRules := make([]Rule, 0)
		for _, un := range uns {
			spec, ok := un["spec"].(map[string]interface{})
			if !ok {
				log.Printf("malformed object, bad spec: %v", uns)
				continue
			}
			unrules, ok := spec["rules"].([]interface{})
			if !ok {
				log.Printf("malformed object, bad rules: %v", uns)
				continue
			}

			for _, ur := range unrules {
				rule := Rule{}
				err := ms.Decode(ur, &rule)
				if err != nil {
					log.Print(err)
				} else {
					newRules = append(newRules, rule)
					log.Print(rule)
				}
			}
		}
		rules.Store(newRules)
	})

	err := godotenv.Load()
	if err != nil {
		log.Print("Error loading .env file")
	}

	common := negroni.Classic()
	common.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		client_id := r.Header.Get("Client-Id")
		client_secret := r.Header.Get("Client-Secret")
		if client_id != "" {
			auth, err := clientCredentials(client_id, client_secret)
			if err != nil {
				log.Println(err)
			} else {
				r.Header.Set("Authorization", fmt.Sprintf("%s %s", auth.Type, auth.Token))
			}
		}
		next(w, r)
	})

	jwtMware := jw.New(jw.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			c := token.Claims.(jwt.MapClaims)

			// Verify 'aud' claim
			if !c.VerifyAudience(os.Getenv("AUTH0_AUDIENCE"), false) {
				return token, errors.New("Invalid audience")
			}

			// Verify 'iss' claim
			// TODO(gsagula): this doesn't need to be allocated everytime.
			iss := fmt.Sprintf("https://%s/", os.Getenv("AUTH0_DOMAIN"))
			if !c.VerifyIssuer(iss, false) {
				return token, errors.New("Invalid issuer")
			}

			// Verify 'exp'
			now := time.Now().Unix()
			if c.VerifyExpiresAt(now, false) == false {
				return token, errors.New("Token is expired")
			}

			// Verify `iat`
			if c.VerifyIssuedAt(now, false) == false {
				return token, errors.New("Token used before issued")
			}

			// Verify `iat`
			if c.VerifyNotBefore(now, false) == false {
				return token, errors.New("Token is not valid yet")
			}

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod:       jwt.SigningMethodRS256,
		CredentialsOptional: true,
	})

	common.UseFunc(jwtMware.HandlerWithNext)

	common.UseHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		public, scopes := policy(r.Method, r.Host, r.URL.Path)
		if !public {
			token, _ := r.Context().Value(jwtMware.Options.UserProperty).(*jwt.Token)
			if token == nil {
				responseJSON("unauthorized", w, r, http.StatusUnauthorized)
				return
			} else {
				for _, scope := range scopes {
					if !checkScope(scope, token.Raw) {
						responseJSON("forbidden", w, r, http.StatusForbidden)
						return
					}
				}
			}
		}
		responseJSON("allowed", w, r, http.StatusOK)
	})

	fmt.Println("Listening on http://localhost:8080")
	http.ListenAndServe("0.0.0.0:8080", common)
}

type AuthRequest struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

type AuthResponse struct {
	Token string `json:"access_token"`
	Type  string `json:"token_type"`
}

func clientCredentials(client_id, client_secret string) (auth AuthResponse, err error) {
	req := AuthRequest{
		ClientId:     client_id,
		ClientSecret: client_secret,
		Audience:     os.Getenv("AUTH0_AUDIENCE"),
		GrantType:    "client_credentials",
	}
	body, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post("https://"+os.Getenv("AUTH0_DOMAIN")+"/oauth/token", "application/json",
		bytes.NewReader(body))
	if err != nil {
		return
	}
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
