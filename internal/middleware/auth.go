package middleware

import (
	"api-systemd/internal/pkg/config"
	"api-systemd/internal/pkg/logger"
	"net/http"
	"strings"
)

// BearerTokenAuth Bearer Token 认证中间件
func BearerTokenAuth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 跳过健康检查和ping接口
			if r.URL.Path == "/health" || r.URL.Path == "/ping" {
				next.ServeHTTP(w, r)
				return
			}

			// API_KEY为空时不应该发生，但作为安全检查
			if cfg.Security.APIKey == "" {
				logger.Error(r.Context(), "API_KEY is empty, this should not happen")
				respondWithError(w, http.StatusInternalServerError, "Authentication not properly configured")
				return
			}

			// 获取Authorization头
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn(r.Context(), "Missing Authorization header", "path", r.URL.Path, "method", r.Method)
				respondWithError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// 检查Bearer Token格式
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				logger.Warn(r.Context(), "Invalid Authorization header format", "path", r.URL.Path, "method", r.Method)
				respondWithError(w, http.StatusUnauthorized, "Authorization header must be Bearer token")
				return
			}

			// 提取token
			token := strings.TrimPrefix(authHeader, bearerPrefix)
			if token == "" {
				logger.Warn(r.Context(), "Empty Bearer token", "path", r.URL.Path, "method", r.Method)
				respondWithError(w, http.StatusUnauthorized, "Bearer token cannot be empty")
				return
			}

			// 验证token
			if token != cfg.Security.APIKey {
				logger.Warn(r.Context(), "Invalid Bearer token",
					"path", r.URL.Path,
					"method", r.Method,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
				)
				respondWithError(w, http.StatusUnauthorized, "Invalid Bearer token")
				return
			}

			// 认证成功，记录日志
			logger.Info(r.Context(), "Authentication successful",
				"path", r.URL.Path,
				"method", r.Method,
				"remote_addr", r.RemoteAddr,
			)

			// 继续处理请求
			next.ServeHTTP(w, r)
		})
	}
}

// respondWithError 返回错误响应
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// 简单的JSON响应
	w.Write([]byte(`{"code": -1, "status": "error", "message": "` + message + `"}`))
}
