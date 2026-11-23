package web

// Sesión del Rostro: cookie opaca firmada HMAC, sin JWT ni JWKS (ADR-111).

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"trotamundos/pater/views"
)

const sessionCookie = "trotamundos_session"

type sessionClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

func sessionSecret() string {
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		return v
	}
	return "trotamundos-dev-session"
}

func sign(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// SetSession emite una cookie HttpOnly con valor firmado (no legible por JS).
func SetSession(w http.ResponseWriter, identity views.UserIdentity) error {
	nonce := make([]byte, 6)
	_, _ = rand.Read(nonce)
	claims := sessionClaims{
		UserID: identity.ID, Email: identity.Email,
		Exp: time.Now().Add(12 * time.Hour).Unix(),
	}
	body, err := json.Marshal(claims)
	if err != nil {
		return err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	nonceHex := hex.EncodeToString(nonce)
	token := payload + "." + nonceHex + "." + sign(payload+nonceHex, sessionSecret())
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: int(12 * time.Hour / time.Second),
	})
	return nil
}

func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

func (h *Handlers) currentUser(r *http.Request) views.UserIdentity {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return views.UserIdentity{}
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return views.UserIdentity{}
	}
	if !hmac.Equal([]byte(sign(parts[0]+parts[1], sessionSecret())), []byte(parts[2])) {
		return views.UserIdentity{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return views.UserIdentity{}
	}
	var claims sessionClaims
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp < time.Now().Unix() || claims.UserID == "" {
		return views.UserIdentity{}
	}
	return views.UserIdentity{ID: claims.UserID, Email: claims.Email}
}

func (h *Handlers) signedIn(r *http.Request) bool { return h.currentUser(r).ID != "" }
