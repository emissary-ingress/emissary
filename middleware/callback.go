package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/secret"
	"github.com/sirupsen/logrus"
)

const (
	// TokenURLFmt is used for exchanging the authorization code.
	TokenURLFmt = "https://%s/oauth/token"
	// AuthorizeFmt is a template string for the authorize post request payload.
	AuthorizeFmt = "{\"grant_type\":\"authorization_code\",\"client_id\": \"%s\",\"code\": \"%s\",\"redirect_uri\": \"%s\"}"
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
)

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

// TokenResponse used for de-serializing response from /oauth/token
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Callback ...
type Callback struct {
	Logger *logrus.Logger
	Config *config.Config
	Secret *secret.Secret
}

func (c *Callback) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.URL.Path == "/callback" {
		c.Logger.Debug("received callback request")
		if err := r.URL.Query().Get("error"); err != "" {
			unauthorized(rw, r)
			return
		}

		redirectPath, err := c.checkState(r)
		if err != nil {
			c.Logger.Errorf("check state failed: %v", err)
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		var tokenRES *TokenResponse
		tokenRES, err = c.checkCode(r)
		if err != nil {
			c.Logger.Errorf("check code failed: %v", err)
			unauthorized(rw, r)
			return
		}

		c.Logger.Debug("authorized: %s, access_token %v bytes, token_id: %v bytes",
			tokenRES.TokenType,
			len(tokenRES.AccessToken),
			len(tokenRES.IDToken),
		)
		c.Logger.Debug("setting %s cookie", AccessTokenCookie)

		http.SetCookie(rw, &http.Cookie{
			Name:    AccessTokenCookie,
			Value:   tokenRES.AccessToken,
			Expires: time.Now().Add(time.Duration(tokenRES.ExpiresIn) * time.Second),
		})

		c.Logger.Debugf("redirecting to path: %s", redirectPath)
		http.Redirect(rw, r, redirectPath, http.StatusFound)
		return
	}

	// TODO(gagula): clean this up.
	cid := r.Header.Get("Client-Id")
	secret := r.Header.Get("Client-Secret")
	if cid != "" && secret != "" {
		auth, err := c.clientCredentials(cid, secret)
		if err != nil {
			c.Logger.Error(err)
		} else {
			r.Header.Set("Authorization", fmt.Sprintf("%s %s", auth.Type, auth.Token))
		}
	} else {
		c.Logger.Debugf("checking %s cookie", AccessTokenCookie)
		cookie, err := r.Cookie(AccessTokenCookie)
		if err != nil {
			c.Logger.Warnf("%s cookie %v", AccessTokenCookie, err)
		} else {
			c.Logger.Debug("setting authorization header")
			r.Header.Set("Authorization", fmt.Sprintf("%s %s", "Bearer", cookie.Value))
		}
	}

	next(rw, r)
}

func (c *Callback) checkState(r *http.Request) (string, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return "", errors.New("empty state param")
	}

	// TODO(gsagula): use parse with claims
	token, err := jwt.Parse(state, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return "", fmt.Errorf("unexpected signing method %v", token.Header["path"])
		}
		return c.Secret.GetPublicKey(), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return "", errors.New("state token validation failed")
	}

	return claims["path"].(string), nil
}

func (c *Callback) checkCode(r *http.Request) (*TokenResponse, error) {
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, errors.New("request does not contain code")
	}

	url := fmt.Sprintf(TokenURLFmt, c.Config.Domain)
	payload := strings.NewReader(fmt.Sprintf(AuthorizeFmt, c.Config.ClientID, code, c.Config.CallbackURL))

	var req *http.Request
	var res *http.Response
	var body []byte
	var err error

	if req, err = http.NewRequest("POST", url, payload); err != nil {
		return nil, err
	}

	c.Logger.Debug("authorizing with the idp")
	req.Header.Add("Content-Type", "Application/Json")

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	token := &TokenResponse{}
	if err = json.Unmarshal(body, token); err != nil {
		return nil, err
	}
	return token, nil
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
		c.Logger.Errorf("checking credentials: %v", err)
		return
	}
	resp, err := http.Post(fmt.Sprintf(TokenURLFmt, c.Config.Domain), "Application/Json", bytes.NewReader(body))
	if err != nil {
		c.Logger.Errorf("checking credentials: %v", err)
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

func unauthorized(w http.ResponseWriter, r *http.Request) {
	jsonResponse, err := json.Marshal(&Response{"unauthorized"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "Application/Json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(jsonResponse)
}
