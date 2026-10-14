package gateway

import (
	"backend/internal/store"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	oidcFlowCookieName    = "smallworld_oidc_flow"
	oidcSessionCookieName = "smallworld_session"
	oidcFlowLifetime      = 10 * time.Minute
	oidcSessionLifetime   = 12 * time.Hour
)

type oidcAuthenticator struct {
	oauth2Config  oauth2.Config
	provider      *oidc.Provider
	verifier      *oidc.IDTokenVerifier
	issuer        string
	providerName  string
	usernameClaim string
	cookieSecure  bool
	cookieCodec   signedCookieCodec
}

type oidcFlow struct {
	State        string `json:"state"`
	Nonce        string `json:"nonce"`
	PKCEVerifier string `json:"pkceVerifier"`
	ExpiresAt    int64  `json:"expiresAt"`
}

type oidcSession struct {
	Issuer    string `json:"issuer"`
	Subject   string `json:"subject"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

type signedCookieCodec struct {
	key []byte
}

func newOIDCAuthenticator(ctx context.Context, cfg Config) (*oidcAuthenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}

	return &oidcAuthenticator{
		oauth2Config: oauth2.Config{
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.OIDCRedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		provider:      provider,
		verifier:      provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID}),
		issuer:        cfg.OIDCIssuerURL,
		providerName:  cfg.OIDCProviderName,
		usernameClaim: cfg.OIDCUsernameClaim,
		cookieSecure:  cfg.OIDCCookieSecure,
		cookieCodec:   signedCookieCodec{key: []byte(cfg.OIDCSessionSecret)},
	}, nil
}

func (g *Gateway) handleAuthConfig(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{"enabled": g.oidc != nil}
	if g.oidc != nil {
		response["providerName"] = g.oidc.providerName
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (g *Gateway) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if g.oidc == nil {
		http.Error(w, "OpenID Connect is not enabled", http.StatusNotFound)
		return
	}

	state, err := randomURLToken(32)
	if err != nil {
		http.Error(w, "could not start authentication", http.StatusInternalServerError)
		return
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		http.Error(w, "could not start authentication", http.StatusInternalServerError)
		return
	}
	pkceVerifier, err := randomURLToken(32)
	if err != nil {
		http.Error(w, "could not start authentication", http.StatusInternalServerError)
		return
	}

	flow := oidcFlow{
		State:        state,
		Nonce:        nonce,
		PKCEVerifier: pkceVerifier,
		ExpiresAt:    time.Now().Add(oidcFlowLifetime).Unix(),
	}
	value, err := g.oidc.cookieCodec.encode(flow)
	if err != nil {
		http.Error(w, "could not start authentication", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcFlowCookieName,
		Value:    value,
		Path:     "/auth",
		MaxAge:   int(oidcFlowLifetime.Seconds()),
		HttpOnly: true,
		Secure:   g.oidc.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	url := g.oidc.oauth2Config.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(pkceVerifier),
	)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (g *Gateway) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if g.oidc == nil {
		http.Error(w, "OpenID Connect is not enabled", http.StatusNotFound)
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		http.Error(w, "identity provider rejected authentication", http.StatusUnauthorized)
		return
	}

	cookie, err := r.Cookie(oidcFlowCookieName)
	if err != nil {
		http.Error(w, "authentication flow cookie is missing", http.StatusBadRequest)
		return
	}
	g.clearOIDCFlowCookie(w)

	var flow oidcFlow
	if err := g.oidc.cookieCodec.decode(cookie.Value, &flow); err != nil || flow.ExpiresAt < time.Now().Unix() {
		http.Error(w, "authentication flow is invalid or expired", http.StatusBadRequest)
		return
	}
	if !hmac.Equal([]byte(flow.State), []byte(r.URL.Query().Get("state"))) {
		http.Error(w, "authentication state does not match", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "authorization code is missing", http.StatusBadRequest)
		return
	}

	token, err := g.oidc.oauth2Config.Exchange(r.Context(), code, oauth2.VerifierOption(flow.PKCEVerifier))
	if err != nil {
		http.Error(w, "authorization code exchange failed", http.StatusBadGateway)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "identity provider did not return an ID token", http.StatusBadGateway)
		return
	}
	idToken, err := g.oidc.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "identity token verification failed", http.StatusUnauthorized)
		return
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "identity claims could not be read", http.StatusUnauthorized)
		return
	}
	nonce, _ := claims["nonce"].(string)
	if nonce == "" || !hmac.Equal([]byte(flow.Nonce), []byte(nonce)) {
		http.Error(w, "identity token nonce does not match", http.StatusUnauthorized)
		return
	}
	username, _ := claims[g.oidc.usernameClaim].(string)
	username = strings.TrimSpace(username)
	if username == "" {
		userInfo, userInfoErr := g.oidc.provider.UserInfo(r.Context(), oauth2.StaticTokenSource(token))
		if userInfoErr == nil && userInfo.Subject == idToken.Subject {
			var userInfoClaims map[string]any
			if claimsErr := userInfo.Claims(&userInfoClaims); claimsErr == nil {
				username, _ = userInfoClaims[g.oidc.usernameClaim].(string)
				username = strings.TrimSpace(username)
			}
		}
	}
	if username == "" {
		http.Error(w, fmt.Sprintf("identity token is missing the %q claim", g.oidc.usernameClaim), http.StatusUnauthorized)
		return
	}

	username, err = g.store.EnsureOIDCUser(r.Context(), g.oidc.issuer, idToken.Subject, username)
	if err != nil {
		if errors.Is(err, store.ErrUserExists) {
			http.Error(w, "that username already belongs to another account", http.StatusConflict)
			return
		}
		http.Error(w, "could not create the local user", http.StatusInternalServerError)
		return
	}

	session := oidcSession{
		Issuer:    g.oidc.issuer,
		Subject:   idToken.Subject,
		Username:  username,
		ExpiresAt: time.Now().Add(oidcSessionLifetime).Unix(),
	}
	value, err := g.oidc.cookieCodec.encode(session)
	if err != nil {
		http.Error(w, "could not create the login session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcSessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(oidcSessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   g.oidc.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (g *Gateway) handleOIDCLogout(w http.ResponseWriter, r *http.Request) {
	secure := false
	if g.oidc != nil {
		secure = g.oidc.cookieSecure
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (g *Gateway) oidcSessionUsername(r *http.Request) (string, bool) {
	if g.oidc == nil {
		return "", false
	}
	cookie, err := r.Cookie(oidcSessionCookieName)
	if err != nil {
		return "", false
	}
	var session oidcSession
	if err := g.oidc.cookieCodec.decode(cookie.Value, &session); err != nil {
		return "", false
	}
	if session.ExpiresAt < time.Now().Unix() || session.Issuer != g.oidc.issuer || session.Subject == "" || session.Username == "" {
		return "", false
	}
	return session.Username, true
}

func (g *Gateway) clearOIDCFlowCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oidcFlowCookieName,
		Value:    "",
		Path:     "/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   g.oidc.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (c signedCookieCodec) encode(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte(payloadPart))
	signaturePart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadPart + "." + signaturePart, nil
}

func (c signedCookieCodec) decode(value string, destination any) error {
	payloadPart, signaturePart, ok := strings.Cut(value, ".")
	if !ok || payloadPart == "" || signaturePart == "" {
		return errors.New("invalid signed cookie")
	}
	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		return errors.New("invalid signed cookie signature")
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write([]byte(payloadPart))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return errors.New("signed cookie signature does not match")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return errors.New("invalid signed cookie payload")
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		return errors.New("invalid signed cookie JSON")
	}
	return nil
}

func randomURLToken(size int) (string, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
