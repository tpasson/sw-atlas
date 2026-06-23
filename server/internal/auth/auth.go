// Package auth provides a minimal single-editor authentication: a bcrypt-verified
// login that issues a signed JWT in an http-only cookie. The Auth type is small
// and self-contained so an SSO/OIDC provider can replace it later without
// touching the feature handlers.
package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const cookieName = "atlas_session"
const sessionTTL = 7 * 24 * time.Hour

type Auth struct {
	secret   []byte
	username string
	hash     string
}

func New(secret, username, hash string) *Auth {
	return &Auth{secret: []byte(secret), username: username, hash: hash}
}

// Verify checks the supplied credentials against the configured editor account.
func (a *Auth) Verify(username, password string) bool {
	if a.hash == "" || username != a.username {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(a.hash), []byte(password)) == nil
}

func (a *Auth) issueToken(username string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(sessionTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// SetCookie issues a session token for username and stores it in an http-only cookie.
func (a *Auth) SetCookie(w http.ResponseWriter, username string) error {
	tok, err := a.issueToken(username)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
	return nil
}

// ClearCookie removes the session cookie.
func (a *Auth) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// IsAuthed reports whether the request carries a valid editor session.
func (a *Auth) IsAuthed(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	tok, err := jwt.Parse(c.Value, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	return err == nil && tok.Valid
}
