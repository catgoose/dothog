// setup:feature:demo

package routes

import (
	"strconv"
	"sync"

	"catgoose/dothog/internal/admininfo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"

	"github.com/catgoose/porter"
	"github.com/labstack/echo/v4"
)

// prefsStore is a simple in-memory store keyed by session UUID.
// Applications should replace this with their own persistence.
var prefsStore = struct {
	sync.RWMutex
	m map[string]admininfo.UserPreferences
}{m: make(map[string]admininfo.UserPreferences)}

func (ar *appRoutes) initUserSettingsRoutes() {
	ar.e.GET("/user/settings", ar.handleUserSettings)
	ar.e.PUT("/user/settings", ar.handleUserSettingsSave)
}

func (ar *appRoutes) handleUserSettings(c echo.Context) error {
	prefs := getUserPrefs(c)
	return handler.RenderBaseLayout(c, views.UserSettingsPage(prefs))
}

func (ar *appRoutes) handleUserSettingsSave(c echo.Context) error {
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

	sessionID := porter.GetSessionSettings(c).SessionUUID
	prefsStore.Lock()
	prefsStore.m[sessionID] = prefs
	prefsStore.Unlock()

	return handler.RenderComponent(c, views.UserSettingsSaved())
}

func getUserPrefs(c echo.Context) admininfo.UserPreferences {
	sessionID := porter.GetSessionSettings(c).SessionUUID
	prefsStore.RLock()
	prefs, ok := prefsStore.m[sessionID]
	prefsStore.RUnlock()
	if !ok {
		return admininfo.DefaultUserPreferences()
	}
	return prefs
}
