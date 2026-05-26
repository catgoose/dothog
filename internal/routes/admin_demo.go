// setup:feature:demo

package routes

import (
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
	"time"

	"catgoose/dothog/internal/admininfo"
	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/version"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

// initAdminDemoRoutes registers the demo-only admin pages: runtime stats,
// update checker, config dump. These ship only with derived apps that keep
// the demo feature; scaffold-facing admin pages live in admin_core.go.
func (ar *AppRoutes) initAdminDemoRoutes() {
	admin := ar.e.Group("/admin")
	admin.GET("/system", ar.handleSystemInfo)
	admin.GET("/system/check-update", ar.handleCheckUpdate)
	admin.GET("/config", ar.handleConfigInfo)
}

func (ar *AppRoutes) handleSystemInfo(c echo.Context) error {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	numThread := 0
	if p := pprof.Lookup("threadcreate"); p != nil {
		numThread = p.Count()
	}

	info := admininfo.SystemInfo{
		Version:    version.Version,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		NumThread:  numThread,
		Uptime:     formatUptime(time.Since(ar.startTime)),

		HeapAllocMB:  fmt.Sprintf("%.1f MB", float64(ms.HeapAlloc)/1024/1024),
		HeapSysMB:    fmt.Sprintf("%.1f MB", float64(ms.HeapSys)/1024/1024),
		StackInUseMB: fmt.Sprintf("%.1f MB", float64(ms.StackInuse)/1024/1024),
		SysMB:        fmt.Sprintf("%.1f MB", float64(ms.Sys)/1024/1024),
		TotalAllocMB: fmt.Sprintf("%.1f MB", float64(ms.TotalAlloc)/1024/1024),

		GCCycles:        ms.NumGC,
		LastPauseMicros: ms.PauseNs[(ms.NumGC+255)%256] / 1000,
		NextGCMB:        fmt.Sprintf("%.1f MB", float64(ms.NextGC)/1024/1024),
		HeapObjects:     ms.HeapObjects,
		LiveObjects:     ms.Mallocs - ms.Frees,
	}

	return handler.RenderBaseLayout(c, views.AdminSystemPage(info))
}

func (ar *AppRoutes) handleCheckUpdate(c echo.Context) error {
	info, err := version.CheckLatest(c.Request().Context())
	if err != nil {
		return handler.RenderComponent(c, views.UpdateCheckResult(version.UpdateInfo{
			Current: version.Version,
		}, err))
	}
	return handler.RenderComponent(c, views.UpdateCheckResult(info, nil))
}

func (ar *AppRoutes) handleConfigInfo(c echo.Context) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load config", err)
	}

	entries := []admininfo.ConfigEntry{
		{Key: "SERVER_LISTEN_PORT", Value: cfg.ServerPort},
		{Key: "APP_NAME", Value: defaultStr(cfg.AppName, "(not set)")},
		// setup:feature:database:start
		{Key: "DATABASE_URL", Value: defaultStr(cfg.DatabaseURL, "(not set)")},
		// setup:feature:database:end
		// setup:feature:auth:start
		{Key: "AUTH_CONFIGURED", Value: fmt.Sprintf("%t", cfg.AuthConfigured())},
		{Key: "SESSION_SECRET", Value: maskSecret(cfg.SessionSecret)},
		// setup:feature:auth:end
		// setup:feature:csrf:start
		{Key: "CSRF_ROTATE_PER_REQUEST", Value: fmt.Sprintf("%t", cfg.CSRFRotatePerRequest)},
		// setup:feature:csrf:end
	}
	// setup:feature:csrf:start
	if len(cfg.CSRFPerRequestPaths) > 0 {
		entries = append(entries, admininfo.ConfigEntry{Key: "CSRF_PER_REQUEST_PATHS", Value: fmt.Sprintf("%v", cfg.CSRFPerRequestPaths)})
	}
	if len(cfg.CSRFExemptPaths) > 0 {
		entries = append(entries, admininfo.ConfigEntry{Key: "CSRF_EXEMPT_PATHS", Value: fmt.Sprintf("%v", cfg.CSRFExemptPaths)})
	}
	// setup:feature:csrf:end

	return handler.RenderBaseLayout(c, views.AdminConfigPage(entries))
}

// formatUptime renders a Duration as Nd Nh Nm Ns, dropping leading zero
// components. Used by the admin pages.
func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, mins, secs)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	return "***REDACTED***"
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
