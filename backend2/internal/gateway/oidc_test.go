package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadConfigRequiresStrongOIDCSessionSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("OIDC_ENABLED", "true")
	t.Setenv("OIDC_ISSUER_URL", "https://auth.example.com/application/o/smallworld/")
	t.Setenv("OIDC_CLIENT_ID", "smallworld")
	t.Setenv("OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("OIDC_REDIRECT_URL", "https://game.example.com/auth/callback")
	t.Setenv("OIDC_SESSION_SECRET", "too-short")
	t.Setenv("OIDC_COOKIE_SECURE", "true")
	t.Setenv("ROOM_WORKER_COUNT", "1")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Fatalf("LoadConfig error = %v, want session-secret length error", err)
	}
}

func TestOIDCLoginStartsAuthorizationCodeFlowWithPKCE(t *testing.T) {
	var providerURL string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                providerURL,
			"authorization_endpoint":                providerURL + "/authorize",
			"token_endpoint":                        providerURL + "/token",
			"jwks_uri":                              providerURL + "/jwks",
			"userinfo_endpoint":                     providerURL + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}))
	providerURL = provider.URL
	defer provider.Close()

	auth, err := newOIDCAuthenticator(t.Context(), Config{
		OIDCIssuerURL:     providerURL,
		OIDCClientID:      "smallworld",
		OIDCClientSecret:  "client-secret",
		OIDCRedirectURL:   "https://game.example.com/auth/callback",
		OIDCSessionSecret: "0123456789abcdef0123456789abcdef",
		OIDCProviderName:  "Authentik",
		OIDCUsernameClaim: "preferred_username",
		OIDCCookieSecure:  true,
	})
	if err != nil {
		t.Fatalf("new OIDC authenticator: %v", err)
	}

	gateway := &Gateway{oidc: auth}
	request := httptest.NewRequest(http.MethodGet, "https://game.example.com/auth/login", nil)
	response := httptest.NewRecorder()
	gateway.handleOIDCLogin(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusSeeOther)
	}
	location := response.Header().Get("Location")
	redirect, err := request.URL.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	query := redirect.Query()
	for _, key := range []string{"state", "nonce", "code_challenge"} {
		if query.Get(key) == "" {
			t.Errorf("authorization redirect has no %s", key)
		}
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", got)
	}
	if got := query.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q, want code", got)
	}
	if got := query.Get("client_id"); got != "smallworld" {
		t.Errorf("client_id = %q, want smallworld", got)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("flow cookies = %d, want 1", len(cookies))
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected flow cookie security attributes: %#v", cookies[0])
	}
}

func TestSignedCookieCodecRoundTrip(t *testing.T) {
	codec := signedCookieCodec{key: []byte("0123456789abcdef0123456789abcdef")}
	want := oidcSession{
		Issuer:    "https://auth.example.com/application/o/smallworld/",
		Subject:   "user-id",
		Username:  "arthur",
		ExpiresAt: 123456789,
	}

	encoded, err := codec.encode(want)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var got oidcSession
	if err := codec.decode(encoded, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != want {
		t.Fatalf("decoded session = %#v, want %#v", got, want)
	}
}

func TestSignedCookieCodecRejectsTampering(t *testing.T) {
	codec := signedCookieCodec{key: []byte("0123456789abcdef0123456789abcdef")}
	encoded, err := codec.encode(oidcSession{Username: "arthur"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	payload, signature, ok := strings.Cut(encoded, ".")
	if !ok {
		t.Fatal("encoded cookie has no signature separator")
	}
	tampered := payload + "x." + signature
	if err := codec.decode(tampered, &oidcSession{}); err == nil {
		t.Fatal("tampered cookie was accepted")
	}
}
