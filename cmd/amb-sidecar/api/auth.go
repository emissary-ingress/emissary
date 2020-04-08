package api

import (
	"context"
	"crypto/rsa"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/resourceserver/rfc6750"
	jwt "github.com/dgrijalva/jwt-go"
)

const AuthCookie = "edge_stack_auth"

type LoginClaimsV1 struct {
	LoginTokenVersion string `json:"login_token_version"`
	jwt.StandardClaims
}

func PermitCookieAuth(allowed func(string) bool, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowed(r.URL.Path) {
			allowedCtx := context.WithValue(r.Context(), allowCookieMarker, allowCookieMarker)
			handler.ServeHTTP(w, r.WithContext(allowedCtx))
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

var allowCookieMarker = struct{}{}

func allowCookieToken(r *http.Request) bool {
	ctx := r.Context()
	return ctx != nil && ctx.Value(allowCookieMarker) == allowCookieMarker
}

// Authorize a request with some tolerence. This is used from webui.go
// and kale.go.
func IsAuthorized(r *http.Request, pubkey *rsa.PublicKey) bool {
	log := dlog.GetLogger(r.Context())
	log.Warningln("PATH: %s", r.URL.Path)

	now := time.Now()
	duration := -5 * time.Minute
	toleratedNow := now.Add(duration)

	nowUnix := now.Unix()
	toleratedNowUnix := toleratedNow.Unix()

	tokenString, _ := rfc6750.GetFromHeader(r.Header)
	if tokenString == "" {
		if allowCookieToken(r) {
			c, err := r.Cookie(AuthCookie)
			if err != nil {
				log.Warningln("error getting cookie:", err)
				return false
			}
			tokenString = c.Value
		}
	}

	if tokenString == "" {
		return false
	}

	var claims LoginClaimsV1

	if pubkey == nil {
		log.Warningln("bypassing JWT validation for request")
		return true
	}
	jwtParser := jwt.Parser{ValidMethods: []string{"PS512"}}
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(tokenString, &claims, func(_ *jwt.Token) (interface{}, error) {
		return pubkey, nil
	}))
	if err != nil {
		return false
	}

	var expiresAtVerification = claims.VerifyExpiresAt(nowUnix, true)
	var issuedAtVerification = claims.VerifyIssuedAt(toleratedNowUnix, true)
	var notBeforeVerification = claims.VerifyNotBefore(toleratedNowUnix, true)
	var loginTokenVersionVerification = claims.LoginTokenVersion == "v1"
	if expiresAtVerification && /* issuedAtVerification && notBeforeVerification && */ loginTokenVersionVerification {
		return true
	} else {
		dlog.GetLogger(r.Context()).Warningln("token failed verification (exp,iat,nbf,vers): " +
			strconv.FormatBool(expiresAtVerification) + " " +
			strconv.FormatBool(issuedAtVerification) + " " +
			strconv.FormatBool(notBeforeVerification) + " " +
			strconv.FormatBool(loginTokenVersionVerification))
		return false
	}
}

func Forbidden(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	io.WriteString(w, "Ambassador Edge Stack admin webui API forbidden")
}

func AuthenticatedHTTPHandler(handler http.Handler, pubkey *rsa.PublicKey) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsAuthorized(r, pubkey) {
			handler.ServeHTTP(w, r)
		} else {
			Forbidden(w, r)
		}
	})
}
