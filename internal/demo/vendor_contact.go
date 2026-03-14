// setup:feature:demo

package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Vendor struct {
	ID       int
	Name     string
	Category string
}

type Contact struct {
	ID       int
	VendorID int
	Name     string
	Email    string
	Phone    string
	Role     string
}

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
	defer rows.Close()
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

func (d *DB) GetVendor(ctx context.Context, id int) (Vendor, error) {
	var v Vendor
	err := d.db.QueryRowContext(ctx, "SELECT id,name,category FROM vendors WHERE id = @ID", sql.Named("ID", id)).Scan(&v.ID, &v.Name, &v.Category)
	return v, err
}

func (d *DB) ListContacts(ctx context.Context, vendorID int) ([]Contact, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT id,vendor_id,name,email,phone,role FROM contacts WHERE vendor_id = @VendorID ORDER BY name", sql.Named("VendorID", vendorID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func (d *DB) GetContact(ctx context.Context, id int) (Contact, error) {
	var c Contact
	err := d.db.QueryRowContext(ctx, "SELECT id,vendor_id,name,email,phone,role FROM contacts WHERE id = @ID", sql.Named("ID", id)).Scan(&c.ID, &c.VendorID, &c.Name, &c.Email, &c.Phone, &c.Role)
	return c, err
}

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
		return err
	}
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_id INTEGER NOT NULL, name TEXT NOT NULL,
		email TEXT NOT NULL, phone TEXT, role TEXT,
		FOREIGN KEY (vendor_id) REFERENCES vendors(id)
	)`); err != nil {
		return err
	}

	var count int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM vendors").Scan(&count); err != nil {
		return err
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

	type cs struct{ vendorIdx int; name, email, role string }
	contacts := []cs{
		{0, "Alice Reed", "alice@acme.com", "Account Manager"},
		{0, "Bob Chen", "bob@acme.com", "Sales Rep"},
		{1, "Carol West", "carol@techflow.com", "CTO"},
		{1, "Dan Kim", "dan@techflow.com", "Lead Engineer"},
		{2, "Eve Santos", "eve@greenleaf.com", "Operations"},
		{3, "Frank Liu", "frank@byteworks.com", "CEO"},
		{3, "Grace Park", "grace@byteworks.com", "VP Engineering"},
		{4, "Hank Mueller", "hank@steelbridge.com", "Procurement"},
		{5, "Iris Tanaka", "iris@cloudnine.com", "Solutions Architect"},
		{5, "Jack Rivera", "jack@cloudnine.com", "Support Lead"},
		{6, "Karen Osei", "karen@pacifictrading.com", "Import Manager"},
		{7, "Leo Barnes", "leo@redwood.com", "Senior Consultant"},
		{8, "Mia Torres", "mia@summit.com", "Sales Director"},
		{9, "Noah Petrov", "noah@blueocean.com", "Logistics Coordinator"},
		{10, "Olivia Grant", "olivia@pinnacle.com", "Partner"},
		{11, "Pete Yamada", "pete@quantumlabs.com", "Research Lead"},
		{12, "Quinn Marsh", "quinn@eagleeye.com", "Security Analyst"},
		{13, "Rosa Vega", "rosa@freshstart.com", "Supply Chain"},
		{14, "Sam Taylor", "sam@silverline.com", "Creative Director"},
		{14, "Tina Novak", "tina@silverline.com", "Account Executive"},
	}

	if err := seedBulk(d.db,
		"INSERT INTO vendors (name,category) VALUES (?,?)",
		len(vendors), func(i int) []any {
			return []any{vendors[i].name, vendors[i].cat}
		}); err != nil {
		return err
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
