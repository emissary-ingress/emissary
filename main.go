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
	"crypto"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	jwtmid "github.com/auth0/go-jwt-middleware"
	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/gobwas/glob"
	"github.com/joho/godotenv"
	ms "github.com/mitchellh/mapstructure"
)

const (
	// Letters is used for random string generator.
	Letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// RedirectURLFmt is a template string for redirect url.
	RedirectURLFmt = "https://%s/authorize?audience=%s&response_type=code&redirect_uri=%s&client_id=%s&state=%s&scope=%s"
	// AuthorizeFmt is a template string for the authorize post request payload.
	AuthorizeFmt = "{\"grant_type\":\"authorization_code\",\"client_id\": \"%s\",\"code\": \"%s\",\"redirect_uri\": \"%s\"}"
	// StateSignature secret is used to sign the authorization state value.
	StateSignature = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3"
)

// Response TODO(gsagula): comment
type Response struct {
	Message string `json:"message"`
}

// Jwks TODO(gsagula): comment
type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

// JSONWebKeys TODO(gsagula): comment
type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// Rule TODO(gsagula): comment
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
var states map[string]string

func init() {
	rules.Store(make([]Rule, 0))
	states = make(map[string]string)

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
					log.Printf("loading rule: %v", rule)
					newRules = append(newRules, rule)
				}
			}
		}
		rules.Store(newRules)
	})
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

// TokenResponse used for de-serializing response from /oauth/token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RandomString .. this can be optimized..
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = Letters[rand.Intn(len(Letters))]
	}
	return string(b)
}

func main() {
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Println("not loaded from .env file")
	}

	common := negroni.Classic()
	common.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// We get token if has code...
		if r.URL.Path == "/callback" {
			if err := r.URL.Query().Get("error"); err != "" {
				responseJSON(err, w, r, http.StatusUnauthorized)
				return
			}

			key := r.URL.Query().Get("state")
			if !validateState(key) {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			code := r.URL.Query().Get("code")
			if code != "" {
				// Remove hardcoded stuff..
				url := "https://gsagula.auth0.com/oauth/token"

				// This needs to be https...
				redirectURL := fmt.Sprintf("http://%s%s", r.Host, r.URL.Path)
				payload := strings.NewReader(fmt.Sprintf(AuthorizeFmt, os.Getenv("AUTH0_CLIENT_ID"), code, redirectURL))

				log.Println("authorizing..")
				req, reqerr := http.NewRequest("POST", url, payload)
				if reqerr != nil {
					log.Println(reqerr)
					responseJSON("access_denied", w, r, http.StatusUnauthorized)
					return
				}

				req.Header.Add("content-type", "application/json")
				res, reserr := http.DefaultClient.Do(req)
				if reserr != nil {
					log.Println(reserr)
					responseJSON("access_denied", w, r, http.StatusUnauthorized)
					return
				}

				defer res.Body.Close()
				body, readerr := ioutil.ReadAll(res.Body)
				if readerr != nil {
					log.Println(readerr)
					responseJSON("access_denied", w, r, http.StatusUnauthorized)
					return
				}

				tokenRES := TokenResponse{}
				if err := json.Unmarshal(body, &tokenRES); err != nil {
					log.Println(err)
					responseJSON("access_denied", w, r, http.StatusUnauthorized)
					return
				}

				log.Printf("setting auth_session cookie")
				expiration := time.Now().Add(time.Duration(tokenRES.ExpiresIn) * time.Second)
				cookie := http.Cookie{Name: "auth_session", Value: tokenRES.AccessToken, Expires: expiration, Domain: r.Host}
				http.SetCookie(w, &cookie)
				http.Redirect(w, r, states[key], http.StatusFound)
				delete(states, key)
			}
		}

		// Check for auth_session cookie
		cookie, err := r.Cookie("auth_session")
		if err == nil {
			r.Header.Set("Authorization", fmt.Sprintf("%s %s", "Bearer", cookie.Value))
		} else {
			// Check for Client-Id and Client-Secret headers
			cid := r.Header.Get("Client-Id")
			secret := r.Header.Get("Client-Secret")
			if cid != "" && secret != "" {
				auth, err := clientCredentials(cid, secret)
				if err != nil {
					log.Println(err)
				} else {
					r.Header.Set("Authorization", fmt.Sprintf("%s %s", auth.Type, auth.Token))
				}
			}
		}

		next(w, r)
	})

	jwtMware := jwtmid.New(jwtmid.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			c := token.Claims.(jwt.MapClaims)

			// Verify 'aud' claim
			if !c.VerifyAudience(os.Getenv("AUTH0_AUDIENCE"), false) {
				return token, errors.New("Invalid audience")
			}

			// Verify 'iss' claim
			// TODO(gsagula): this string doesn't need to be instantianted everytime. Inspect other occurrences in the code.
			iss := fmt.Sprintf("https://%s/", os.Getenv("AUTH0_DOMAIN"))
			if !c.VerifyIssuer(iss, false) {
				return token, errors.New("Invalid issuer")
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
				authorize(w, r)
				return
			}

			if err := token.Claims.Valid(); err != nil {
				// TODO(gsagula): consider logging the error.
				authorize(w, r)
				return
			}

			// TODO(gsagula): considere redirecting to consent uri and logging the error.
			for _, scope := range scopes {
				if !checkScope(scope, token.Raw) {
					responseJSON("forbidden", w, r, http.StatusForbidden)
					return
				}
			}
		}

		responseJSON("allowed", w, r, http.StatusOK)
	})

	log.Println("listening on :8080")
	http.ListenAndServe("0.0.0.0:8080", common)
}

func signState(url string) string {
	token := jwt.New(&jwt.SigningMethodHMAC{Name: "HS256", Hash: crypto.SHA256})
	key, err := token.SignedString([]byte(StateSignature))
	if err != nil {
		log.Fatal(err)
	}
	states[key] = url
	return key
}

func validateState(key string) bool {
	token, err := jwt.Parse(key, func(token *jwt.Token) (interface{}, error) {
		return []byte(StateSignature), nil
	})
	if err != nil || !token.Valid {
		return false
	}
	return true
}

func authorize(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString(r.URL.Path)
	if len(r.URL.RawQuery) > 0 {
		buf.WriteString("?")
		buf.WriteString(r.URL.RawQuery)
	}

	state := signState(buf.String())
	states[state] = buf.String()

	redirectURL := fmt.Sprintf(
		RedirectURLFmt,
		os.Getenv("AUTH0_DOMAIN"),
		os.Getenv("AUTH0_AUDIENCE"),
		fmt.Sprintf("http://%s/callback", r.Host),
		os.Getenv("AUTH0_CLIENT_ID"),
		state,
		os.Getenv("AUTH0_SCOPES"),
	)

	log.Printf("authorizing with the IDP.")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// AuthRequest TODO(gsagula): comment
type AuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

// AuthResponse TODO(gsagula): comment
type AuthResponse struct {
	Token string `json:"access_token"`
	Type  string `json:"token_type"`
}

func clientCredentials(cid, secret string) (auth AuthResponse, err error) {
	req := AuthRequest{
		ClientID:     cid,
		ClientSecret: secret,
		Audience:     os.Getenv("AUTH0_AUDIENCE"),
		GrantType:    "client_credentials",
	}
	body, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(fmt.Sprintf("https://%s/oauth/token", os.Getenv("AUTH0_DOMAIN")), "application/json",
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

// CustomClaims TODO(gsagula): comment
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
	}
	pemCert, err := getPemCertUncached(token)
	return pemCert, err
}

func getPemCertUncached(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get(fmt.Sprintf("https://%s/.well-known/jwks.json", os.Getenv("AUTH0_DOMAIN")))
	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("Unable to find appropriate key")
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
			w.Header().Set("Client-Secret", "")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jsonResponse)
}
