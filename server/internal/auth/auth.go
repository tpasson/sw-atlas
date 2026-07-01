// Package auth handles the session cookie: it mints and verifies a signed JWT
// that carries the logged-in user's identity (id, name, role, workspace). It
// also exposes stateless bcrypt helpers. Credential storage lives in the store;
// this package only knows about tokens and hashing, so an SSO/OIDC provider can
// replace the login flow later without touching the feature handlers.
package auth

import (
	"net/http"
	"strings"
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
	UserID        string
	Username      string
	Role          string
	WorkspaceID   string
	WorkspaceSlug string
	TokenVersion  int // must match the user's current token_version (revocation)
}

// sessionClaims embeds the registered JWT claims plus ATLAS identity fields.
type sessionClaims struct {
	jwt.RegisteredClaims
	Username      string `json:"usr"`
	Role          string `json:"rol"`
	WorkspaceID   string `json:"wsp"`
	WorkspaceSlug string `json:"wss"`
	TokenVersion  int    `json:"tv"`
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

// dummyHash is a valid bcrypt hash used only to spend the same CPU time on a
// login attempt for a non-existent user, so response timing doesn't reveal
// whether a username exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// DummyPasswordCheck runs a throwaway bcrypt comparison (constant-ish time).
func DummyPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}

func (a *Auth) issueToken(s Session) (string, error) {
	claims := sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   s.UserID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sessionTTL)),
		},
		Username:      s.Username,
		Role:          s.Role,
		WorkspaceID:   s.WorkspaceID,
		WorkspaceSlug: s.WorkspaceSlug,
		TokenVersion:  s.TokenVersion,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
}

// SetCookie issues a session token for the identity and stores it in an
// http-only cookie. Secure is set when the request arrived over HTTPS (directly
// or via a trusted proxy's X-Forwarded-Proto), so the cookie never travels over
// plain HTTP in a TLS deployment — while local http dev still works.
func (a *Auth) SetCookie(w http.ResponseWriter, r *http.Request, s Session) error {
	tok, err := a.issueToken(s)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
	return nil
}

// isHTTPS reports whether the request effectively arrived over TLS.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
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
		UserID:        claims.Subject,
		Username:      claims.Username,
		Role:          claims.Role,
		WorkspaceID:   claims.WorkspaceID,
		WorkspaceSlug: claims.WorkspaceSlug,
		TokenVersion:  claims.TokenVersion,
	}, true
}

// IsAuthed reports whether the request carries a valid session.
func (a *Auth) IsAuthed(r *http.Request) bool {
	_, ok := a.SessionFromRequest(r)
	return ok
}
