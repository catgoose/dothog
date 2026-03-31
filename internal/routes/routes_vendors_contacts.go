// setup:feature:demo

package routes

import (
	"catgoose/dothog/internal/demo"
	"catgoose/dothog/internal/routes/handler"
	"catgoose/dothog/internal/routes/hypermedia"
	"catgoose/dothog/internal/routes/params"
	ssebroker "github.com/catgoose/tavern"
	"catgoose/dothog/web/views"

	"github.com/labstack/echo/v4"

)

type vendorContactRoutes struct {
	db     *demo.DB
	actLog *demo.ActivityLog
	broker *ssebroker.SSEBroker
}

func (ar *appRoutes) initVendorContactRoutes(db *demo.DB, actLog *demo.ActivityLog, broker *ssebroker.SSEBroker) {
	v := &vendorContactRoutes{db: db, actLog: actLog, broker: broker}
	ar.e.GET("/demo/vendors", v.handleVendorsPage)
	ar.e.GET("/demo/vendors/list", v.handleVendorsList)
	ar.e.GET("/demo/vendors/:id/contacts", v.handleVendorContacts)
	ar.e.GET("/demo/vendors/contacts/:id/edit", v.handleContactEdit)
	ar.e.GET("/demo/vendors/contacts/:id/card", v.handleContactCard)
	ar.e.PUT("/demo/vendors/contacts/:id", v.handleContactUpdate)
}

func (v *vendorContactRoutes) handleVendorsPage(c echo.Context) error {
	search := c.QueryParam("q")
	category := c.QueryParam("category")
	vendors, err := v.db.ListVendors(c.Request().Context(), search, category)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load vendors", err)
	}
	bar := v.buildFilterBar(search, category)
	return handler.RenderBaseLayout(c, views.VendorContactsPage(vendors, bar))
}

func (v *vendorContactRoutes) handleVendorsList(c echo.Context) error {
	search := c.QueryParam("q")
	category := c.QueryParam("category")
	vendors, err := v.db.ListVendors(c.Request().Context(), search, category)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load vendors", err)
	}
	return handler.RenderComponent(c, views.VendorListFiltered(vendors))
}

func (v *vendorContactRoutes) handleVendorContacts(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid vendor ID", err)
	}
	vendor, err := v.db.GetVendor(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Vendor not found", err)
	}
	contacts, err := v.db.ListContacts(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to load contacts", err)
	}
	return handler.RenderComponent(c, views.VendorContactsDetail(vendor, contacts))
}

func (v *vendorContactRoutes) handleContactEdit(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Contact not found", err)
	}
	return handler.RenderComponent(c, views.ContactEditForm(contact))
}

func (v *vendorContactRoutes) handleContactCard(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Contact not found", err)
	}
	return handler.RenderComponent(c, views.ContactCard(contact))
}

func (v *vendorContactRoutes) handleContactUpdate(c echo.Context) error {
	id, err := params.ParseParamID(c, "id")
	if err != nil {
		return handler.HandleHypermediaError(c, 400, "Invalid contact ID", err)
	}
	contact, err := v.db.GetContact(c.Request().Context(), id)
	if err != nil {
		return handler.HandleHypermediaError(c, 404, "Contact not found", err)
	}
	contact.Name = c.FormValue("name")
	contact.Email = c.FormValue("email")
	contact.Phone = c.FormValue("phone")
	contact.Role = c.FormValue("role")
	if err := v.db.UpdateContact(c.Request().Context(), contact); err != nil {
		return handler.HandleHypermediaError(c, 500, "Failed to update contact", err)
	}
	evt := v.actLog.Record("updated", "contact", id, contact.Name, "contact updated")
	BroadcastActivity(v.broker, evt)
	return handler.RenderComponent(c, views.ContactCard(contact))
}

func (v *vendorContactRoutes) buildFilterBar(search, category string) hypermedia.FilterBar {
	return hypermedia.NewFilterBar("/demo/vendors/list", "#vendor-list",
		hypermedia.SearchField("q", "Search vendors\u2026", search),
		hypermedia.SelectField("category", "Category", category,
			hypermedia.SelectOptions(category, vendorCategoryPairs()...)),
	)
}

func vendorCategoryPairs() []string {
	pairs := []string{"", "All Categories"}
	for _, c := range demo.VendorCategories {
		pairs = append(pairs, c, c)
	}
	return pairs
}
