// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/params"
	"catgoose/dothog/web/views"
	"github.com/catgoose/linkwell"
	"github.com/catgoose/tavern"
	"net/http"

	"github.com/labstack/echo/v4"
)

type vendorContactRoutes struct {
	db     *demo.DB
	actLog *demo.ActivityLog
	broker *tavern.SSEBroker
}

func (ar *AppRoutes) initVendorContactRoutes(db *demo.DB, actLog *demo.ActivityLog, broker *tavern.SSEBroker) {
	v := &vendorContactRoutes{db: db, actLog: actLog, broker: broker}
	vendors := ar.e.Group("/apps/vendors")
	vendors.GET("", v.handleVendorsPage)
	vendors.GET("/list", v.handleVendorsList)
	vendors.GET("/:id/contacts", v.handleVendorContacts)
	vendors.GET("/contacts/:id/edit", v.handleContactEdit)
	vendors.GET("/contacts/:id/card", v.handleContactCard)
	vendors.PUT("/contacts/:id", v.handleContactUpdate)
}

func (v *vendorContactRoutes) handleVendorsPage(c echo.Context) error {
	search := c.QueryParam("q")
	category := c.QueryParam("category")
	vendors, err := v.db.ListVendors(c.Request().Context(), search, category)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load vendors", err)
	}
	bar := v.buildFilterBar(search, category)
	return handler.RenderBaseLayout(c, views.VendorContactsPage(vendors, bar))
}

func (v *vendorContactRoutes) handleVendorsList(c echo.Context) error {
	search := c.QueryParam("q")
	category := c.QueryParam("category")
	vendors, err := v.db.ListVendors(c.Request().Context(), search, category)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load vendors", err)
	}
	return handler.RenderComponent(c, views.VendorListFiltered(vendors))
}

func (v *vendorContactRoutes) handleVendorContacts(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid vendor ID", err)
	}
	vendor, err := v.db.GetVendor(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Vendor not found", err)
	}
	contacts, err := v.db.ListContacts(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to load contacts", err)
	}
	return handler.RenderComponent(c, views.VendorContactsDetail(vendor, contacts))
}

func (v *vendorContactRoutes) handleContactEdit(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Contact not found", err)
	}
	return handler.RenderComponent(c, views.ContactEditForm(contact))
}

func (v *vendorContactRoutes) handleContactCard(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Contact not found", err)
	}
	return handler.RenderComponent(c, views.ContactCard(contact))
}

func (v *vendorContactRoutes) handleContactUpdate(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusBadRequest, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, http.StatusNotFound, "Contact not found", err)
	}
	contact.Name = c.FormValue("name")
	contact.Email = c.FormValue("email")
	contact.Phone = c.FormValue("phone")
	contact.Role = c.FormValue("role")
	if err := v.db.UpdateContact(c.Request().Context(), contact); err != nil {
		return handler.HandleHypermediaError(c, http.StatusInternalServerError, "Failed to update contact", err)
	}
	evt := v.actLog.Record("updated", "contact", id, contact.Name, "contact updated")
	BroadcastActivity(v.broker, evt)
	return handler.RenderComponent(c, views.ContactCard(contact))
}

func (v *vendorContactRoutes) buildFilterBar(search, category string) linkwell.FilterBar {
	return linkwell.NewFilterBar("/apps/vendors/list", "#vendor-list",
		linkwell.SearchField("q", "Search vendors\u2026", search),
		linkwell.SelectField("category", "Category", category,
			linkwell.SelectOptions(category, vendorCategoryPairs()...)),
	)
}

func vendorCategoryPairs() []string {
	pairs := []string{"", "All Categories"}
	for _, c := range demo.VendorCategories {
		pairs = append(pairs, c, c)
	}
	return pairs
}
