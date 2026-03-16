package routes

import (
	"fmt"
	"runtime"
	"runtime/pprof"
	"time"

	"catgoose/dothog/internal/admininfo"
	"catgoose/dothog/internal/config"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"
)

func (ar *appRoutes) initAdminCoreRoutes() {
	ar.e.GET("/admin/system", ar.handleSystemInfo)
	ar.e.GET("/admin/config", ar.handleConfigInfo)
}


func (ar *appRoutes) handleSystemInfo(c echo.Context) error {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	numThread := 0
	if p := pprof.Lookup("threadcreate"); p != nil {
		numThread = p.Count()
	}

	info := admininfo.SystemInfo{
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

func (ar *appRoutes) handleConfigInfo(c echo.Context) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load config", err)
	}

	entries := []admininfo.ConfigEntry{
		{"SERVER_LISTEN_PORT", cfg.ServerPort},
		{"APP_NAME", defaultStr(cfg.AppName, "(not set)")},
		{"ENABLE_DATABASE", fmt.Sprintf("%t", cfg.EnableDatabase)},
		{"INIT_REPO", fmt.Sprintf("%t", cfg.InitRepo)},
		{"CROONER_DISABLED", fmt.Sprintf("%t", cfg.CroonerDisabled)},
		{"SESSION_SECRET", maskSecret(cfg.SessionSecret)},
		{"CSRF_ROTATE_PER_REQUEST", fmt.Sprintf("%t", cfg.CSRFRotatePerRequest)},
	}
	if len(cfg.CSRFPerRequestPaths) > 0 {
		entries = append(entries, admininfo.ConfigEntry{"CSRF_PER_REQUEST_PATHS", fmt.Sprintf("%v", cfg.CSRFPerRequestPaths)})
	}
	if len(cfg.CSRFExemptPaths) > 0 {
		entries = append(entries, admininfo.ConfigEntry{"CSRF_EXEMPT_PATHS", fmt.Sprintf("%v", cfg.CSRFExemptPaths)})
	}

	return handler.RenderBaseLayout(c, views.AdminConfigPage(entries))
}

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
