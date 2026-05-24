// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/web/views"
	"net/http"

	"github.com/labstack/echo/v4"
)

type settingsRoutes struct {
	store *demo.SettingsStore
}

func (ar *AppRoutes) initSettingsRoutes(store *demo.SettingsStore) {
	s := &settingsRoutes{store: store}
	settings := ar.e.Group("/platform/settings")
	settings.GET("", s.handleSettingsPage)
	settings.GET("/:id", s.handleSettingsSection)
	settings.PUT("/:id", s.handleSettingsSave)
}

func (s *settingsRoutes) handleSettingsPage(c echo.Context) error {
	sections := s.store.AllSections()
	return handler.RenderBaseLayout(c, views.SettingsPage(sections))
}

func (s *settingsRoutes) handleSettingsSection(c echo.Context) error {
	id := c.Param("id")
	sec, ok := s.store.GetSection(id)
	if !ok {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Section not found", nil)
	}
	return handler.RenderComponent(c, views.SettingsSectionForm(sec))
}

func (s *settingsRoutes) handleSettingsSave(c echo.Context) error {
	id := c.Param("id")
	sec, ok := s.store.GetSection(id)
	if !ok {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Section not found", nil)
	}
	values := make(map[string]string)
	for _, f := range sec.Fields {
		if f.Kind == "toggle" {
			if c.FormValue(f.Key) == "true" {
				values[f.Key] = "true"
			} else {
				values[f.Key] = "false"
			}
		} else {
			values[f.Key] = c.FormValue(f.Key)
		}
	}
	if _, ok := s.store.UpdateSection(id, values); !ok {
		return handler.RenderComponent(c, views.SettingsSaveResult(false, "Section not found"))
	}
	return handler.RenderComponent(c, views.SettingsSaveResult(true, "Settings saved"))
}
