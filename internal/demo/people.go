// setup:feature:demo

package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Person represents a person in the directory.
type Person struct {
	FirstName  string
	LastName   string
	Email      string
	Phone      string
	City       string
	State      string
	Department string
	JobTitle   string
	Bio        string
	CreatedAt  string
	ID         int
}

// FullName returns "First Last".
func (p Person) FullName() string { return p.FirstName + " " + p.LastName }

// Departments is the list of available departments for filters.
var Departments = []string{
	"Engineering", "Sales", "Marketing", "Finance", "Human Resources",
	"Operations", "Legal", "Customer Support", "Product", "Design",
}

var allowedPeopleSort = map[string]string{
	"name":       "last_name",
	"department": "department",
	"city":       "city",
	"title":      "job_title",
}

// ListPeople returns a paginated list of people matching optional search and department filters.
func (d *DB) ListPeople(ctx context.Context, search, department, sortBy, sortDir string, page, perPage int) ([]Person, int, error) {
	col, ok := allowedPeopleSort[sortBy]
	if !ok {
		col = "last_name"
		sortDir = "asc"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	var conds []string
	var args []any
	if search != "" {
		conds = append(conds, "(first_name LIKE @Search OR last_name LIKE @Search OR email LIKE @Search)")
		args = append(args, sql.Named("Search", "%"+search+"%"))
	}
	if department != "" {
		conds = append(conds, "department = @Department")
		args = append(args, sql.Named("Department", department))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := d.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM people %s", where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage
	query := fmt.Sprintf(
		"SELECT id,first_name,last_name,email,phone,city,state,department,job_title,bio,created_at FROM people %s ORDER BY %s %s LIMIT @Limit OFFSET @Offset",
		where, col, sortDir)
	la := make([]any, len(args), len(args)+2)
	copy(la, args)
	la = append(la, sql.Named("Limit", perPage), sql.Named("Offset", offset))

	rows, err := d.db.QueryContext(ctx, query, la...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.City, &p.State, &p.Department, &p.JobTitle, &p.Bio, &p.CreatedAt); err != nil {
			return nil, 0, err
		}
		people = append(people, p)
	}
	return people, total, rows.Err()
}

// GetPerson returns a single person by ID.
func (d *DB) GetPerson(ctx context.Context, id int) (Person, error) {
	var p Person
	err := d.db.QueryRowContext(ctx,
		"SELECT id,first_name,last_name,email,phone,city,state,department,job_title,bio,created_at FROM people WHERE id = @ID", sql.Named("ID", id),
	).Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.City, &p.State, &p.Department, &p.JobTitle, &p.Bio, &p.CreatedAt)
	if err != nil {
		return Person{}, fmt.Errorf("get person %d: %w", id, err)
	}
	return p, nil
}

// UpdatePerson updates a person row.
func (d *DB) UpdatePerson(ctx context.Context, p Person) error {
	res, err := d.db.ExecContext(ctx,
		"UPDATE people SET first_name=@FirstName, last_name=@LastName, email=@Email, phone=@Phone, city=@City, state=@State, department=@Department, job_title=@JobTitle, bio=@Bio WHERE id=@ID",
		sql.Named("FirstName", p.FirstName), sql.Named("LastName", p.LastName), sql.Named("Email", p.Email), sql.Named("Phone", p.Phone), sql.Named("City", p.City), sql.Named("State", p.State), sql.Named("Department", p.Department), sql.Named("JobTitle", p.JobTitle), sql.Named("Bio", p.Bio), sql.Named("ID", p.ID))
	if err != nil {
		return fmt.Errorf("update person %d: %w", p.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update person %d: no rows affected", p.ID)
	}
	return nil
}

func (d *DB) initPeople() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS people (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL, last_name TEXT NOT NULL,
		email TEXT NOT NULL, phone TEXT,
		city TEXT, state TEXT,
		department TEXT, job_title TEXT, bio TEXT,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create people table: %w", err)
	}
	var count int
	if err := d.db.QueryRow("SELECT COUNT(*) FROM people").Scan(&count); err != nil {
		return fmt.Errorf("count people rows: %w", err)
	}
	if count == 0 {
		return d.seedPeople()
	}
	return nil
}

func (d *DB) seedPeople() error {
	type ps struct{ fn, ln, city, st, dept, title string }
	data := []ps{
		{"James", "Smith", "New York", "NY", "Engineering", "Software Engineer"},
		{"Mary", "Johnson", "Los Angeles", "CA", "Marketing", "Content Strategist"},
		{"Robert", "Williams", "Chicago", "IL", "Sales", "Account Executive"},
		{"Patricia", "Brown", "Houston", "TX", "Finance", "Financial Analyst"},
		{"John", "Jones", "Phoenix", "AZ", "Engineering", "Senior Engineer"},
		{"Jennifer", "Garcia", "Seattle", "WA", "Design", "UX Researcher"},
		{"Michael", "Miller", "Denver", "CO", "Operations", "Operations Manager"},
		{"Linda", "Davis", "Austin", "TX", "Human Resources", "HR Specialist"},
		{"David", "Rodriguez", "Portland", "OR", "Engineering", "DevOps Engineer"},
		{"Elizabeth", "Martinez", "Nashville", "TN", "Product", "Product Manager"},
		{"William", "Wilson", "Atlanta", "GA", "Sales", "Sales Manager"},
		{"Barbara", "Anderson", "Miami", "FL", "Marketing", "Marketing Director"},
		{"Richard", "Thomas", "Raleigh", "NC", "Engineering", "Staff Engineer"},
		{"Susan", "Taylor", "Minneapolis", "MN", "Legal", "Legal Counsel"},
		{"Joseph", "Moore", "San Francisco", "CA", "Engineering", "Engineering Manager"},
		{"Jessica", "Jackson", "Columbus", "OH", "Customer Support", "Support Lead"},
		{"Thomas", "Martin", "San Diego", "CA", "Finance", "Controller"},
		{"Sarah", "Lee", "Dallas", "TX", "Design", "Product Designer"},
		{"Christopher", "Thompson", "San Jose", "CA", "Engineering", "QA Engineer"},
		{"Karen", "White", "Jacksonville", "FL", "Marketing", "Marketing Coordinator"},
		{"Daniel", "Harris", "Indianapolis", "IN", "Operations", "Logistics Coordinator"},
		{"Lisa", "Clark", "Fort Worth", "TX", "Human Resources", "Recruiter"},
		{"Matthew", "Lewis", "Charlotte", "NC", "Sales", "Sales Representative"},
		{"Nancy", "Robinson", "San Antonio", "TX", "Legal", "Paralegal"},
		{"Anthony", "Walker", "Philadelphia", "PA", "Engineering", "Software Engineer"},
	}

	return seedBulk(d.db,
		"INSERT INTO people (first_name,last_name,email,phone,city,state,department,job_title,bio,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		len(data), func(i int) []any {
			p := data[i]
			email := fmt.Sprintf("%s.%s@example.com", strings.ToLower(p.fn), strings.ToLower(p.ln))
			phone := fmt.Sprintf("555-%04d", 1000+i)
			bio := fmt.Sprintf("%s in the %s department, based in %s, %s.", p.title, p.dept, p.city, p.st)
			date := fmt.Sprintf("2024-%02d-%02d", (i%12)+1, (i%28)+1)
			return []any{p.fn, p.ln, email, phone, p.city, p.st, p.dept, p.title, bio, date}
		})
}
