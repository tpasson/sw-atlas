// Package auth handles the session cookie: it mints and verifies a signed JWT
// that carries the logged-in user's identity (id, name, role, workspace). It
// also exposes stateless bcrypt helpers. Credential storage lives in the store;
// this package only knows about tokens and hashing, so an SSO/OIDC provider can
// replace the login flow later without touching the feature handlers.
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
	secret []byte
}

func New(secret string) *Auth {
	return &Auth{secret: []byte(secret)}
}

// Session is the authenticated identity carried in the cookie.
type Session struct {
	UserID      string
	Username    string
	Role        string
	WorkspaceID string
}

// sessionClaims embeds the registered JWT claims plus ATLAS identity fields.
type sessionClaims struct {
	jwt.RegisteredClaims
	Username    string `json:"usr"`
	Role        string `json:"rol"`
	WorkspaceID string `json:"wsp"`
}

// HashPassword returns a bcrypt hash suitable for storage.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (a *Auth) issueToken(s Session) (string, error) {
	claims := sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   s.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sessionTTL)),
		},
		Username:    s.Username,
		Role:        s.Role,
		WorkspaceID: s.WorkspaceID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// SetCookie issues a session token for the identity and stores it in an
// http-only cookie.
func (a *Auth) SetCookie(w http.ResponseWriter, s Session) error {
	tok, err := a.issueToken(s)
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

// SessionFromRequest parses and validates the session cookie, returning the
// identity it carries.
func (a *Auth) SessionFromRequest(r *http.Request) (Session, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, false
	}
	var claims sessionClaims
	tok, err := jwt.ParseWithClaims(c.Value, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return a.secret, nil
	})
	if err != nil || !tok.Valid {
		return Session{}, false
	}
	return Session{
		UserID:      claims.Subject,
		Username:    claims.Username,
		Role:        claims.Role,
		WorkspaceID: claims.WorkspaceID,
	}, true
}

// IsAuthed reports whether the request carries a valid session.
func (a *Auth) IsAuthed(r *http.Request) bool {
	_, ok := a.SessionFromRequest(r)
	return ok
}
