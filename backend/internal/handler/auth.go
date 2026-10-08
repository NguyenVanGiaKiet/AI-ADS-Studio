package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ai-ads-studio/backend/internal/service"
)

const authCookieName = "ai_ads_studio_session"

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Thông tin đăng nhập không hợp lệ", http.StatusBadRequest)
		return
	}

	token, err := h.AuthService.LoginFrom(r.RemoteAddr, request.Username, request.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		http.Error(w, "Tên đăng nhập hoặc mật khẩu không chính xác", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, service.ErrLoginRateLimited) {
		w.Header().Set("Retry-After", "900")
		http.Error(w, "Quá nhiều lần đăng nhập sai. Vui lòng thử lại sau 15 phút.", http.StatusTooManyRequests)
		return
	}
	if err != nil {
		http.Error(w, "Không thể tạo phiên đăng nhập", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(service.AuthSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"username": request.Username},
	})
}

func (h *Handler) AuthSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil || !h.AuthService.IsAuthenticated(cookie.Value) {
		respondJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"authenticated": false}})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"authenticated": true},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(authCookieName); err == nil {
		h.AuthService.Logout(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
	respondJSON(w, http.StatusOK, map[string]string{"message": "Đã đăng xuất"})
}

func (h *Handler) RequireAuthentication(next http.Handler, allowedOrigins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions ||
			r.URL.Path == "/api/health" ||
			r.URL.Path == "/api/auth/login" ||
			r.URL.Path == "/api/auth/session" {
			next.ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origin := r.Header.Get("Origin"); origin != "" && !allowedOrigins[origin] {
				http.Error(w, "Origin không được phép", http.StatusForbidden)
				return
			}
		}

		cookie, err := r.Cookie(authCookieName)
		if err == nil && h.AuthService.IsAuthenticated(cookie.Value) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Yêu cầu đăng nhập", http.StatusUnauthorized)
	})
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
