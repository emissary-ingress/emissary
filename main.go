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
	// TokenURLFmt is used for exchanging the authorization code.
	TokenURLFmt = "https://%s/oauth/token"
	// WellKnownURLFmt ..
	WellKnownURLFmt = "https://%s/.well-known/jwks.json"
	// AuthorizeFmt is a template string for the authorize post request payload.
	AuthorizeFmt = "{\"grant_type\":\"authorization_code\",\"client_id\": \"%s\",\"code\": \"%s\",\"redirect_uri\": \"%s\"}"
	// StateSignature secret is used to sign the authorization state value.
	StateSignature = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3"
	// ClientIDKey header key
	ClientIDKey = "Client-Id"
	// ClientSECKey header key
	ClientSECKey = "Client-Secret"
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
	// AuthzKEY header key.
	AuthzKEY = "Authorization"
	// StringNIL is used for empty string comparisson
	StringNIL = ""
	// ContentTYPE HTTP header key
	ContentTYPE = "Content-Type"
	// ApplicationJSON HTTP header value
	ApplicationJSON = "Application/Json"
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

// TokenResponse used for de-serializing response from /oauth/token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

var kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")

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

// Map is used to to track state and the initial request url, so
// it the call can be redirected after acquiring the access token.
var stateURLKv map[string]string

func init() {
	rules.Store(make([]Rule, 0))
	stateURLKv = make(map[string]string)
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

func main() {
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Println("failed to load from .env file")
	}

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

	common := negroni.Classic()
	common.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		// We get token if has code...
		if r.URL.Path == "/callback" {
			if err := r.URL.Query().Get("error"); err != StringNIL {
				responseJSON(err, w, r, http.StatusUnauthorized)
				return
			}

			stateKEY := r.URL.Query().Get("state")
			if !validateState(stateKEY) && stateURLKv[stateKEY] == StringNIL {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			code := r.URL.Query().Get("code")
			if code != StringNIL {
				// Remove hardcoded stuff..
				url := fmt.Sprintf(TokenURLFmt, os.Getenv("AUTH0_DOMAIN"))

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

				req.Header.Add(ContentTYPE, ApplicationJSON)
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

				log.Printf("setting %s cookie", AccessTokenCookie)
				http.SetCookie(w, &http.Cookie{
					Name:    AccessTokenCookie,
					Value:   tokenRES.AccessToken,
					Expires: time.Now().Add(time.Duration(tokenRES.ExpiresIn) * time.Second),
					Domain:  r.Host},
				)
				http.Redirect(w, r, stateURLKv[stateKEY], http.StatusFound)
				delete(stateURLKv, stateKEY)
			}
		}

		// Check for auth_session cookie
		cookie, err := r.Cookie(AccessTokenCookie)
		if err == nil {
			r.Header.Set(AuthzKEY, fmt.Sprintf("%s %s", "Bearer", cookie.Value))
		} else {
			// Check for Client-Id and Client-Secret headers
			cid := r.Header.Get(ClientIDKey)
			secret := r.Header.Get(ClientSECKey)
			if cid != StringNIL && secret != StringNIL {
				auth, err := clientCredentials(cid, secret)
				if err != nil {
					log.Println(err)
				} else {
					r.Header.Set(AuthzKEY, fmt.Sprintf("%s %s", auth.Type, auth.Token))
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
			if !c.VerifyIssuer(fmt.Sprintf("https://%s/", os.Getenv("AUTH0_DOMAIN")), false) {
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
				if os.Getenv("AUTH0_ON_FAILURE") == "login" {
					authorize(w, r)
				} else {
					responseJSON("unauthorized", w, r, http.StatusUnauthorized)
				}
				return
			}

			if err := token.Claims.Valid(); err != nil {
				// TODO(gsagula): consider logging the error.
				if os.Getenv("AUTH0_ON_FAILURE") == "login" {
					authorize(w, r)
				} else {
					responseJSON("unauthorized", w, r, http.StatusUnauthorized)
				}
				return
			}

			// TODO(gsagula): considere redirecting to consent uri and logging the error.
			for _, scope := range scopes {
				if !checkScope(scope, token.Raw) {
					if os.Getenv("AUTH0_ON_FAILURE") == "login" {
						authorize(w, r)
					} else {
						responseJSON("unauthorized", w, r, http.StatusUnauthorized)
					}
					return
				}
			}
		}

		responseJSON("allowed", w, r, http.StatusOK)
	})
	log.Println("serving on port 8080")
	if err := http.ListenAndServe(":8080", common); err != nil {
		log.Println(err)
	}
}

func signState(url string) string {
	token := jwt.New(&jwt.SigningMethodHMAC{Name: "HS256", Hash: crypto.SHA256})
	key, err := token.SignedString([]byte(StateSignature))
	if err != nil {
		log.Fatal(err)
	}
	stateURLKv[key] = url
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

	stateKEY := signState(buf.String())
	stateURLKv[stateKEY] = buf.String()

	redirectURL := fmt.Sprintf(
		RedirectURLFmt,
		os.Getenv("AUTH0_DOMAIN"),
		os.Getenv("AUTH0_AUDIENCE"),
		os.Getenv("AUTH0_CALLBACK_URL"),
		os.Getenv("AUTH0_CLIENT_ID"),
		stateKEY,
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
	resp, err := http.Post(fmt.Sprintf(TokenURLFmt, os.Getenv("AUTH0_DOMAIN")), ApplicationJSON,
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

var pemCert = StringNIL

func getPemCert(token *jwt.Token) (string, error) {
	var err error
	if pemCert != StringNIL {
		return pemCert, nil
	}
	pemCert, err := getPemCertUncached(token)
	return pemCert, err
}

func getPemCertUncached(token *jwt.Token) (string, error) {
	cert := StringNIL
	resp, err := http.Get(fmt.Sprintf(WellKnownURLFmt, os.Getenv("AUTH0_DOMAIN")))
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

	if cert == StringNIL {
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
		w.Header().Set(AuthzKEY, r.Header.Get(AuthzKEY))
		w.Header().Del(ClientSECKey)
	} else {
		w.Write(jsonResponse)
	}

	w.Header().Set(ContentTYPE, ApplicationJSON)
	w.WriteHeader(statusCode)
}
