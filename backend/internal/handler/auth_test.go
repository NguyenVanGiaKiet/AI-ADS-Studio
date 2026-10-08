package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-ads-studio/backend/internal/service"
)

func TestRequireAuthenticationProtectsRoutesAndAllowsHealth(t *testing.T) {
	auth, err := service.NewAuthService("admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	h := NewHandler(nil, nil)
	h.AuthService = auth

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/videos", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	protected := h.RequireAuthentication(mux, map[string]bool{"http://localhost:5173": true})

	healthResponse := httptest.NewRecorder()
	protected.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthResponse.Code, http.StatusOK)
	}

	unauthorizedResponse := httptest.NewRecorder()
	protected.ServeHTTP(unauthorizedResponse, httptest.NewRequest(http.MethodGet, "/api/videos", nil))
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthorizedResponse.Code, http.StatusUnauthorized)
	}

	token, err := auth.LoginFrom("192.0.2.2:12345", "admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	authorizedRequest := httptest.NewRequest(http.MethodGet, "/api/videos", nil)
	authorizedRequest.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	authorizedResponse := httptest.NewRecorder()
	protected.ServeHTTP(authorizedResponse, authorizedRequest)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, want %d", authorizedResponse.Code, http.StatusOK)
	}

	forbiddenRequest := httptest.NewRequest(http.MethodPost, "/api/videos", nil)
	forbiddenRequest.Header.Set("Origin", "https://untrusted.example")
	forbiddenRequest.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	forbiddenResponse := httptest.NewRecorder()
	protected.ServeHTTP(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("untrusted origin status = %d, want %d", forbiddenResponse.Code, http.StatusForbidden)
	}
}

func TestLoginSetsHttpOnlySessionCookie(t *testing.T) {
	auth, err := service.NewAuthService("admin", "correct-horse-battery")
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	h := NewHandler(nil, nil)
	h.AuthService = auth

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(
		`{"username":"admin","password":"correct-horse-battery"}`,
	))
	response := httptest.NewRecorder()
	h.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Login() status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != authCookieName {
		t.Fatalf("Login() cookies = %#v, want session cookie", cookies)
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].MaxAge <= 0 {
		t.Fatalf("session cookie attributes = %#v, want HttpOnly, SameSite=Strict, and a positive MaxAge", cookies[0])
	}
}
