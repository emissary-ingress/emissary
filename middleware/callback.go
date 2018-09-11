package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

const (
	// AuthzKEY header key.
	AuthzKEY = "Authorization"
	// ClientIDKey header key
	ClientIDKey = "Client-Id"
	// ClientSECKey header key
	ClientSECKey = "Client-Secret"
	// StateSignature secret is used to sign the authorization state value.
	StateSignature = "vg=pgHoAAWgCsGuKBX,U3qrUGmqrPGE3"
	// ContentTYPE HTTP header key
	ContentTYPE = "Content-Type"
	// ApplicationJSON HTTP header value
	ApplicationJSON = "Application/Json"
	// EmptyString ...
	EmptyString = ""
	// TokenURLFmt is used for exchanging the authorization code.
	TokenURLFmt = "https://%s/oauth/token"
	// AuthorizeFmt is a template string for the authorize post request payload.
	AuthorizeFmt = "{\"grant_type\":\"authorization_code\",\"client_id\": \"%s\",\"code\": \"%s\",\"redirect_uri\": \"%s\"}"
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
)

// Callback ...
type Callback struct {
	Logger     *logrus.Logger
	Config     *config.Config
	StateKV    *map[string]string
	PrivateKey string
}

func (c *Callback) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	c.Logger.Infof("callback midleware path requested: %s", r.URL.Path)
	if r.URL.Path == "/callback" {
		if err := r.URL.Query().Get("error"); err != EmptyString {
			unauthorized(rw, r)
			return
		}

		state := r.URL.Query().Get("state")
		if state == EmptyString {
			c.Logger.Warnf("host: %s did not provide state param", r.Host)
			http.Redirect(rw, r, "/", http.StatusFound)
			return
		}

		token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["path"])
			}
			return []byte(c.PrivateKey), nil
		})

		if err != nil {
			c.Logger.Warnf("error parsing signed state: %v", err)
			http.Redirect(rw, r, "/", http.StatusFound)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !(ok && token.Valid) {
			c.Logger.Errorf("state validation failed: %v", err)
			http.Redirect(rw, r, "/", http.StatusFound)
			return
		}

		code := r.URL.Query().Get("code")
		if code != EmptyString {
			url := fmt.Sprintf(TokenURLFmt, c.Config.Domain)
			payload := strings.NewReader(fmt.Sprintf(AuthorizeFmt, c.Config.ClientID, code, c.Config.CallbackURL))

			c.Logger.Info("authorizing with idp")
			req, reqerr := http.NewRequest("POST", url, payload)
			if reqerr != nil {
				c.Logger.Errorf("calling idp: %v", reqerr)
				unauthorized(rw, r)
				return
			}

			req.Header.Add(ContentTYPE, ApplicationJSON)
			res, reserr := http.DefaultClient.Do(req)
			if reserr != nil {
				unauthorized(rw, r)
				return
			}

			defer res.Body.Close()
			body, readerr := ioutil.ReadAll(res.Body)
			if readerr != nil {
				unauthorized(rw, r)
				return
			}

			tokenRES := TokenResponse{}
			if err := json.Unmarshal(body, &tokenRES); err != nil {
				unauthorized(rw, r)
				return
			}

			c.Logger.Infof("setting %s cookie", AccessTokenCookie)
			http.SetCookie(rw, &http.Cookie{
				Name:    AccessTokenCookie,
				Value:   tokenRES.AccessToken,
				Expires: time.Now().Add(time.Duration(tokenRES.ExpiresIn) * time.Second),
				Domain:  r.Host},
			)

			redirectPath := claims["path"].(string)
			c.Logger.Infof("redirecting to path: %s", redirectPath)
			http.Redirect(rw, r, redirectPath, http.StatusFound)
			return
		}
	}

	c.Logger.Info("checking for auth_session cookie")
	cookie, err := r.Cookie(AccessTokenCookie)
	if err == nil {
		c.Logger.Info("setting authorization header")
		r.Header.Set(AuthzKEY, fmt.Sprintf("%s %s", "Bearer", cookie.Value))
	} else {
		// Check for Client-Id and Client-Secret headers
		cid := r.Header.Get(ClientIDKey)
		secret := r.Header.Get(ClientSECKey)
		if cid != EmptyString && secret != EmptyString {
			auth, err := c.clientCredentials(cid, secret)
			if err != nil {
				c.Logger.Error(err)
			} else {
				r.Header.Set(AuthzKEY, fmt.Sprintf("%s %s", auth.Type, auth.Token))
			}
		}
	}

	next(rw, r)
}

func (c *Callback) clientCredentials(cid, secret string) (auth AuthResponse, err error) {
	req := AuthRequest{
		ClientID:     cid,
		ClientSecret: secret,
		Audience:     c.Config.Audience,
		GrantType:    "client_credentials",
	}
	body, err := json.Marshal(req)
	if err != nil {
		return
	}
	resp, err := http.Post(fmt.Sprintf(TokenURLFmt, c.Config.Domain), ApplicationJSON,
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

// AuthResponse TODO(gsagula): comment
type AuthResponse struct {
	Token string `json:"access_token"`
	Type  string `json:"token_type"`
}

// AuthRequest TODO(gsagula): comment
type AuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

// Response TODO(gsagula): comment
type Response struct {
	Message string `json:"message"`
}

func (c *Callback) validateState(key string) bool {
	token, err := jwt.Parse(key, func(token *jwt.Token) (interface{}, error) {
		return []byte(StateSignature), nil
	})
	if err != nil || !token.Valid {
		return false
	}
	return true
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	jsonResponse, err := json.Marshal(&Response{"unauthorized"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentTYPE, ApplicationJSON)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(jsonResponse)
}

// TokenResponse used for de-serializing response from /oauth/token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
