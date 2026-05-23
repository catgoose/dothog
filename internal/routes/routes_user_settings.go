// setup:feature:demo

package routes

import (
	"strconv"
	"sync"
	"time"

	"catgoose/dothog/internal/admininfo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"catgoose/dothog/internal/session"
	"github.com/labstack/echo/v4"
)

type prefsEntry struct {
	touchedAt time.Time
	prefs     admininfo.UserPreferences
}

// prefsStore is a simple in-memory store keyed by session UUID with TTL eviction.
var prefsStore = struct {
	m map[string]prefsEntry
	sync.RWMutex
}{m: make(map[string]prefsEntry)}

const prefsTTL = 24 * time.Hour

func evictStalePrefs() {
	now := time.Now()
	for k, v := range prefsStore.m {
		if now.Sub(v.touchedAt) > prefsTTL {
			delete(prefsStore.m, k)
		}
	}
}

func (ar *AppRoutes) initUserSettingsRoutes() {
	ar.e.GET("/user/settings", ar.handleUserSettings)
	ar.e.PUT("/user/settings", ar.handleUserSettingsSave)
}

func (ar *AppRoutes) handleUserSettings(c echo.Context) error {
	prefs := getUserPrefs(c)
	return handler.RenderBaseLayout(c, views.UserSettingsPage(prefs))
}

func (ar *AppRoutes) handleUserSettingsSave(c echo.Context) error {
	pageSize, _ := strconv.Atoi(c.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = 20
	}

	prefs := admininfo.UserPreferences{
		PageSize:             pageSize,
		DateFormat:           c.FormValue("date_format"),
		CompactTables:        c.FormValue("compact_tables") == "true",
		EmailOnError:         c.FormValue("email_on_error") == "true",
		DesktopNotifications: c.FormValue("desktop_notifications") == "true",
		ReduceMotion:         c.FormValue("reduce_motion") == "true",
		HighContrast:         c.FormValue("high_contrast") == "true",
	}
	if prefs.DateFormat == "" {
		prefs.DateFormat = "relative"
	}

	sessionID := session.GetSettings(c.Request()).SessionUUID
	prefsStore.Lock()
	evictStalePrefs()
	prefsStore.m[sessionID] = prefsEntry{prefs: prefs, touchedAt: time.Now()}
	prefsStore.Unlock()

	return handler.RenderComponent(c, views.UserSettingsSaved())
}

func getUserPrefs(c echo.Context) admininfo.UserPreferences {
	sessionID := session.GetSettings(c.Request()).SessionUUID
	prefsStore.RLock()
	entry, ok := prefsStore.m[sessionID]
	prefsStore.RUnlock()
	if !ok || time.Since(entry.touchedAt) > prefsTTL {
		return admininfo.DefaultUserPreferences()
	}
	return entry.prefs
}
