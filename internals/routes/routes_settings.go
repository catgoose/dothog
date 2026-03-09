// setup:feature:demo

package routes

import (
	"catgoose/go-htmx-demo/internals/demo"
	"catgoose/go-htmx-demo/internals/routes/handler"
	"catgoose/go-htmx-demo/web/views"

	"github.com/labstack/echo/v4"
)

type settingsRoutes struct {
	store *demo.SettingsStore
}

func (ar *appRoutes) initSettingsRoutes(store *demo.SettingsStore) {
	s := &settingsRoutes{store: store}
	ar.e.GET("/demo/settings", s.handleSettingsPage)
	ar.e.GET("/demo/settings/:id", s.handleSettingsSection)
	ar.e.PUT("/demo/settings/:id", s.handleSettingsSave)
}

func (s *settingsRoutes) handleSettingsPage(c echo.Context) error {
	sections := s.store.AllSections()
	return handler.RenderBaseLayout(c, views.SettingsPage(sections))
}

func (s *settingsRoutes) handleSettingsSection(c echo.Context) error {
	id := c.Param("id")
	sec, ok := s.store.GetSection(id)
	if !ok {
		return handler.HandleHypermediaError(c, 404, "Section not found", nil)
	}
	return handler.RenderComponent(c, views.SettingsSectionForm(sec))
}

func (s *settingsRoutes) handleSettingsSave(c echo.Context) error {
	id := c.Param("id")
	sec, ok := s.store.GetSection(id)
	if !ok {
		return handler.HandleHypermediaError(c, 404, "Section not found", nil)
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
