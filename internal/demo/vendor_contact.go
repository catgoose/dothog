// setup:feature:demo

package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Vendor represents a vendor in the demo dataset.
type Vendor struct {
	Name     string
	Category string
	ID       int
}

// Contact represents a vendor contact in the demo dataset.
type Contact struct {
	Name     string
	Email    string
	Phone    string
	Role     string
	ID       int
	VendorID int
}

// ListVendors returns vendors matching the optional search and category filters.
func (d *DB) ListVendors(ctx context.Context, search, category string) ([]Vendor, error) {
	var conds []string
	var args []any
	if search != "" {
		conds = append(conds, "name LIKE @Search")
		args = append(args, sql.Named("Search", "%"+search+"%"))
	}
	if category != "" {
		conds = append(conds, "category = @Category")
		args = append(args, sql.Named("Category", category))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	rows, err := d.db.QueryContext(ctx, fmt.Sprintf("SELECT id,name,category FROM vendors %s ORDER BY name", where), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var vendors []Vendor
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Category); err != nil {
			return nil, err
		}
		vendors = append(vendors, v)
	}
	return vendors, rows.Err()
}

// GetVendor returns a single vendor by ID.
func (d *DB) GetVendor(ctx context.Context, id int) (Vendor, error) {
	var v Vendor
	err := d.db.QueryRowContext(ctx, "SELECT id,name,category FROM vendors WHERE id = @ID", sql.Named("ID", id)).Scan(&v.ID, &v.Name, &v.Category)
	return v, err
}

// ListContacts returns all contacts for a vendor.
func (d *DB) ListContacts(ctx context.Context, vendorID int) ([]Contact, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT id,vendor_id,name,email,phone,role FROM contacts WHERE vendor_id = @VendorID ORDER BY name", sql.Named("VendorID", vendorID))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var contacts []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.VendorID, &c.Name, &c.Email, &c.Phone, &c.Role); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// GetContact returns a single contact by ID.
func (d *DB) GetContact(ctx context.Context, id int) (Contact, error) {
	var c Contact
	err := d.db.QueryRowContext(ctx, "SELECT id,vendor_id,name,email,phone,role FROM contacts WHERE id = @ID", sql.Named("ID", id)).Scan(&c.ID, &c.VendorID, &c.Name, &c.Email, &c.Phone, &c.Role)
	return c, err
}

// UpdateContact updates a contact row.
func (d *DB) UpdateContact(ctx context.Context, c Contact) error {
	res, err := d.db.ExecContext(ctx, "UPDATE contacts SET name=@Name, email=@Email, phone=@Phone, role=@Role WHERE id=@ID",
		sql.Named("Name", c.Name), sql.Named("Email", c.Email), sql.Named("Phone", c.Phone), sql.Named("Role", c.Role), sql.Named("ID", c.ID))
	if err != nil {
		return fmt.Errorf("update contact %d: %w", c.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update contact %d: no rows affected", c.ID)
	}
	return nil
}

// VendorCategories for filter dropdowns.
var VendorCategories = []string{
	"General", "Technology", "Agriculture", "Manufacturing",
	"Consulting", "Research", "Security", "Food & Beverage",
	"Marketing", "Energy", "Automotive", "Education", "Logistics",
}

func (d *DB) initVendors() error {
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS vendors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, category TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create vendors table: %w", err)
	}
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_id INTEGER NOT NULL, name TEXT NOT NULL,
		email TEXT NOT NULL, phone TEXT, role TEXT,
		FOREIGN KEY (vendor_id) REFERENCES vendors(id)
	)`); err != nil {
		return fmt.Errorf("create contacts table: %w", err)
	}

	var count int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count); err != nil {
		return fmt.Errorf("count vendors rows: %w", err)
	}
	if count > 0 {
		return nil
	}

	type vs struct{ name, cat string }
	vendors := []vs{
		{"Acme Corp", "General"}, {"TechFlow Inc", "Technology"},
		{"GreenLeaf Supply", "Agriculture"}, {"ByteWorks", "Technology"},
		{"SteelBridge Ltd", "Manufacturing"}, {"CloudNine Systems", "Technology"},
		{"Pacific Trading Co", "General"}, {"Redwood Analytics", "Consulting"},
		{"Summit Hardware", "General"}, {"BlueOcean Logistics", "Logistics"},
		{"Pinnacle Services", "Consulting"}, {"Quantum Labs", "Research"},
		{"EagleEye Security", "Security"}, {"FreshStart Foods", "Food & Beverage"},
		{"Silverline Media", "Marketing"},
	}

	type cs struct {
		name      string
		email     string
		role      string
		vendorIdx int
	}
	contacts := []cs{
		{vendorIdx: 0, name: "Alice Reed", email: "alice@acme.com", role: "Account Manager"},
		{vendorIdx: 0, name: "Bob Chen", email: "bob@acme.com", role: "Sales Rep"},
		{vendorIdx: 1, name: "Carol West", email: "carol@techflow.com", role: "CTO"},
		{vendorIdx: 1, name: "Dan Kim", email: "dan@techflow.com", role: "Lead Engineer"},
		{vendorIdx: 2, name: "Eve Santos", email: "eve@greenleaf.com", role: "Operations"},
		{vendorIdx: 3, name: "Frank Liu", email: "frank@byteworks.com", role: "CEO"},
		{vendorIdx: 3, name: "Grace Park", email: "grace@byteworks.com", role: "VP Engineering"},
		{vendorIdx: 4, name: "Hank Mueller", email: "hank@steelbridge.com", role: "Procurement"},
		{vendorIdx: 5, name: "Iris Tanaka", email: "iris@cloudnine.com", role: "Solutions Architect"},
		{vendorIdx: 5, name: "Jack Rivera", email: "jack@cloudnine.com", role: "Support Lead"},
		{vendorIdx: 6, name: "Karen Osei", email: "karen@pacifictrading.com", role: "Import Manager"},
		{vendorIdx: 7, name: "Leo Barnes", email: "leo@redwood.com", role: "Senior Consultant"},
		{vendorIdx: 8, name: "Mia Torres", email: "mia@summit.com", role: "Sales Director"},
		{vendorIdx: 9, name: "Noah Petrov", email: "noah@blueocean.com", role: "Logistics Coordinator"},
		{vendorIdx: 10, name: "Olivia Grant", email: "olivia@pinnacle.com", role: "Partner"},
		{vendorIdx: 11, name: "Pete Yamada", email: "pete@quantumlabs.com", role: "Research Lead"},
		{vendorIdx: 12, name: "Quinn Marsh", email: "quinn@eagleeye.com", role: "Security Analyst"},
		{vendorIdx: 13, name: "Rosa Vega", email: "rosa@freshstart.com", role: "Supply Chain"},
		{vendorIdx: 14, name: "Sam Taylor", email: "sam@silverline.com", role: "Creative Director"},
		{vendorIdx: 14, name: "Tina Novak", email: "tina@silverline.com", role: "Account Executive"},
	}

	if err := seedBulk(d.db,
		"INSERT INTO vendors (name,category) VALUES (?,?)",
		len(vendors), func(i int) []any {
			return []any{vendors[i].name, vendors[i].cat}
		}); err != nil {
		return fmt.Errorf("seed vendors: %w", err)
	}
	return seedBulk(d.db,
		"INSERT INTO contacts (vendor_id,name,email,phone,role) VALUES (?,?,?,?,?)",
		len(contacts), func(i int) []any {
			c := contacts[i]
			vid := c.vendorIdx + 1
			phone := fmt.Sprintf("555-%04d", 2000+c.vendorIdx*10)
			return []any{vid, c.name, c.email, phone, c.role}
		})
}
