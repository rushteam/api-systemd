package router

import (
	"api-systemd/internal/app"
	"api-systemd/internal/pkg/logger"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// New 创建新的路由器
func New() *chi.Mux {
	r := chi.NewRouter()

	// 全局中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customLogger)
	r.Use(middleware.Recoverer)
	r.Use(customCORS)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.Compress(5))

	// 创建应用实例
	app := app.New()

	// 设置路由
	setupRoutes(r, app)

	return r
}

// setupRoutes 设置所有路由
func setupRoutes(r *chi.Mux, app *app.App) {
	// API v1 路由组
	r.Route("/api/v1", func(r chi.Router) {
		// 服务管理路由组
		r.Route("/services", func(r chi.Router) {
			r.Post("/deploy", app.Deploy)
			r.Get("/start", app.StartService)
			r.Get("/stop", app.Stop)
			r.Get("/restart", app.Restart)
			r.Delete("/remove", app.Remove)
			r.Get("/status", app.GetStatus)
			r.Get("/logs", app.GetLogs)

			// 支持带参数的路由
			r.Route("/{serviceName}", func(r chi.Router) {
				r.Get("/status", app.GetStatus)
				r.Get("/logs", app.GetLogs)
				r.Post("/start", app.StartService)
				r.Post("/stop", app.Stop)
				r.Post("/restart", app.Restart)
				r.Delete("/", app.Remove)
			})
		})

		// 配置管理路由组
		r.Route("/configs", func(r chi.Router) {
			r.Post("/create", app.CreateConfig)
			r.Delete("/delete", app.DeleteConfig)
		})
	})

	// 兼容旧版本路由（无版本前缀）
	r.Route("/services", func(r chi.Router) {
		r.Post("/deploy", app.Deploy)
		r.Get("/start", app.StartService)
		r.Get("/stop", app.Stop)
		r.Get("/restart", app.Restart)
		r.Delete("/remove", app.Remove)
		r.Get("/status", app.GetStatus)
		r.Get("/logs", app.GetLogs)
	})

	r.Route("/configs", func(r chi.Router) {
		r.Post("/create", app.CreateConfig)
		r.Delete("/delete", app.DeleteConfig)
	})

	// 健康检查和系统信息
	r.Get("/health", app.HealthCheck)
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("pong"))
	})

	// 添加路由调试信息（开发环境）
	r.Mount("/debug", middleware.Profiler())
}

// customLogger 自定义日志中间件
func customLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 记录请求开始
		logger.Info(r.Context(), "HTTP Request",
			"method", r.Method,
			"url", r.URL.String(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"request_id", middleware.GetReqID(r.Context()),
		)

		// 创建响应记录器
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// 记录请求完成
		logger.Info(r.Context(), "HTTP Response",
			"method", r.Method,
			"url", r.URL.String(),
			"status_code", ww.Status(),
			"bytes_written", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

// customCORS 自定义CORS中间件
func customCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
