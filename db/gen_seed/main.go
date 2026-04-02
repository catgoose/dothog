// Command gen_seed creates db/seed.db with reference lookup data.
// Run: go run db/gen_seed/main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	_ = os.Remove("db/seed.db")
	db, err := sql.Open("sqlite3", "db/seed.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mustExec(db, `CREATE TABLE first_names (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		gender TEXT NOT NULL DEFAULT 'U'
	)`)

	mustExec(db, `CREATE TABLE last_names (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE email_domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL,
		name TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE cities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		state_code TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE vendors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE departments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)

	mustExec(db, `CREATE TABLE job_titles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		department TEXT NOT NULL
	)`)

	// --- Seed data ---

	firstNames := []struct{ name, gender string }{
		{"James", "M"}, {"Mary", "F"}, {"Robert", "M"}, {"Patricia", "F"},
		{"John", "M"}, {"Jennifer", "F"}, {"Michael", "M"}, {"Linda", "F"},
		{"David", "M"}, {"Elizabeth", "F"}, {"William", "M"}, {"Barbara", "F"},
		{"Richard", "M"}, {"Susan", "F"}, {"Joseph", "M"}, {"Jessica", "F"},
		{"Thomas", "M"}, {"Sarah", "F"}, {"Christopher", "M"}, {"Karen", "F"},
		{"Charles", "M"}, {"Lisa", "F"}, {"Daniel", "M"}, {"Nancy", "F"},
		{"Matthew", "M"}, {"Betty", "F"}, {"Anthony", "M"}, {"Margaret", "F"},
		{"Mark", "M"}, {"Sandra", "F"}, {"Donald", "M"}, {"Ashley", "F"},
		{"Steven", "M"}, {"Kimberly", "F"}, {"Andrew", "M"}, {"Emily", "F"},
		{"Paul", "M"}, {"Donna", "F"}, {"Joshua", "M"}, {"Michelle", "F"},
		{"Kenneth", "M"}, {"Carol", "F"}, {"Kevin", "M"}, {"Amanda", "F"},
		{"Brian", "M"}, {"Dorothy", "F"}, {"George", "M"}, {"Melissa", "F"},
		{"Timothy", "M"}, {"Deborah", "F"},
	}
	for _, fn := range firstNames {
		mustExec(db, "INSERT INTO first_names (name, gender) VALUES (?, ?)", fn.name, fn.gender)
	}

	lastNames := []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones",
		"Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
		"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
		"Thomas", "Taylor", "Moore", "Jackson", "Martin",
		"Lee", "Perez", "Thompson", "White", "Harris",
		"Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
		"Walker", "Young", "Allen", "King", "Wright",
		"Scott", "Torres", "Nguyen", "Hill", "Flores",
		"Green", "Adams", "Nelson", "Baker", "Hall",
		"Rivera", "Campbell", "Mitchell", "Carter", "Roberts",
	}
	for _, ln := range lastNames {
		mustExec(db, "INSERT INTO last_names (name) VALUES (?)", ln)
	}

	emailDomains := []string{
		"example.com", "test.org", "acme.co", "initech.com",
		"globex.net", "hooli.com", "piedpiper.io", "wayneent.com",
		"starkindustries.com", "umbrella.corp",
	}
	for _, d := range emailDomains {
		mustExec(db, "INSERT INTO email_domains (domain) VALUES (?)", d)
	}

	states := []struct{ code, name string }{
		{"AL", "Alabama"}, {"AK", "Alaska"}, {"AZ", "Arizona"}, {"AR", "Arkansas"},
		{"CA", "California"}, {"CO", "Colorado"}, {"CT", "Connecticut"}, {"DE", "Delaware"},
		{"FL", "Florida"}, {"GA", "Georgia"}, {"HI", "Hawaii"}, {"ID", "Idaho"},
		{"IL", "Illinois"}, {"IN", "Indiana"}, {"IA", "Iowa"}, {"KS", "Kansas"},
		{"KY", "Kentucky"}, {"LA", "Louisiana"}, {"ME", "Maine"}, {"MD", "Maryland"},
		{"MA", "Massachusetts"}, {"MI", "Michigan"}, {"MN", "Minnesota"}, {"MS", "Mississippi"},
		{"MO", "Missouri"}, {"MT", "Montana"}, {"NE", "Nebraska"}, {"NV", "Nevada"},
		{"NH", "New Hampshire"}, {"NJ", "New Jersey"}, {"NM", "New Mexico"}, {"NY", "New York"},
		{"NC", "North Carolina"}, {"ND", "North Dakota"}, {"OH", "Ohio"}, {"OK", "Oklahoma"},
		{"OR", "Oregon"}, {"PA", "Pennsylvania"}, {"RI", "Rhode Island"}, {"SC", "South Carolina"},
		{"SD", "South Dakota"}, {"TN", "Tennessee"}, {"TX", "Texas"}, {"UT", "Utah"},
		{"VT", "Vermont"}, {"VA", "Virginia"}, {"WA", "Washington"}, {"WV", "West Virginia"},
		{"WI", "Wisconsin"}, {"WY", "Wyoming"},
	}
	for _, s := range states {
		mustExec(db, "INSERT INTO states (code, name) VALUES (?, ?)", s.code, s.name)
	}

	cities := []struct{ name, state string }{
		{"New York", "NY"}, {"Los Angeles", "CA"}, {"Chicago", "IL"}, {"Houston", "TX"},
		{"Phoenix", "AZ"}, {"Philadelphia", "PA"}, {"San Antonio", "TX"}, {"San Diego", "CA"},
		{"Dallas", "TX"}, {"San Jose", "CA"}, {"Austin", "TX"}, {"Jacksonville", "FL"},
		{"Fort Worth", "TX"}, {"Columbus", "OH"}, {"Charlotte", "NC"}, {"Indianapolis", "IN"},
		{"San Francisco", "CA"}, {"Seattle", "WA"}, {"Denver", "CO"}, {"Nashville", "TN"},
		{"Portland", "OR"}, {"Las Vegas", "NV"}, {"Memphis", "TN"}, {"Louisville", "KY"},
		{"Baltimore", "MD"}, {"Milwaukee", "WI"}, {"Albuquerque", "NM"}, {"Tucson", "AZ"},
		{"Fresno", "CA"}, {"Sacramento", "CA"}, {"Atlanta", "GA"}, {"Miami", "FL"},
		{"Raleigh", "NC"}, {"Omaha", "NE"}, {"Minneapolis", "MN"}, {"Tampa", "FL"},
		{"New Orleans", "LA"}, {"Cleveland", "OH"}, {"Pittsburgh", "PA"}, {"St. Louis", "MO"},
	}
	for _, ct := range cities {
		mustExec(db, "INSERT INTO cities (name, state_code) VALUES (?, ?)", ct.name, ct.state)
	}

	vendors := []struct{ name, category string }{
		{"Acme Corp", "General"}, {"TechFlow Inc", "Technology"}, {"GreenLeaf Supply", "Agriculture"},
		{"ByteWorks", "Technology"}, {"SteelBridge Ltd", "Manufacturing"}, {"CloudNine Systems", "Technology"},
		{"Pacific Trading Co", "Import/Export"}, {"Redwood Analytics", "Consulting"},
		{"Summit Hardware", "Hardware"}, {"BlueOcean Logistics", "Logistics"},
		{"Pinnacle Services", "Consulting"}, {"Quantum Labs", "Research"},
		{"HarborView Inc", "Real Estate"}, {"EagleEye Security", "Security"},
		{"FreshStart Foods", "Food & Beverage"}, {"Silverline Media", "Marketing"},
		{"NorthStar Energy", "Energy"}, {"VeloTech Motors", "Automotive"},
		{"BrightPath Education", "Education"}, {"CoreStack Software", "Technology"},
	}
	for _, v := range vendors {
		mustExec(db, "INSERT INTO vendors (name, category) VALUES (?, ?)", v.name, v.category)
	}

	departments := []string{
		"Engineering", "Sales", "Marketing", "Finance", "Human Resources",
		"Operations", "Legal", "Customer Support", "Product", "Design",
	}
	for _, d := range departments {
		mustExec(db, "INSERT INTO departments (name) VALUES (?)", d)
	}

	jobTitles := []struct{ title, dept string }{
		{"Software Engineer", "Engineering"}, {"Senior Engineer", "Engineering"},
		{"Staff Engineer", "Engineering"}, {"Engineering Manager", "Engineering"},
		{"QA Engineer", "Engineering"}, {"DevOps Engineer", "Engineering"},
		{"Sales Representative", "Sales"}, {"Account Executive", "Sales"},
		{"Sales Manager", "Sales"}, {"Marketing Coordinator", "Marketing"},
		{"Content Strategist", "Marketing"}, {"Marketing Director", "Marketing"},
		{"Financial Analyst", "Finance"}, {"Controller", "Finance"},
		{"HR Specialist", "Human Resources"}, {"Recruiter", "Human Resources"},
		{"Operations Manager", "Operations"}, {"Logistics Coordinator", "Operations"},
		{"Legal Counsel", "Legal"}, {"Paralegal", "Legal"},
		{"Support Specialist", "Customer Support"}, {"Support Lead", "Customer Support"},
		{"Product Manager", "Product"}, {"Product Designer", "Design"},
		{"UX Researcher", "Design"}, {"Visual Designer", "Design"},
	}
	for _, jt := range jobTitles {
		mustExec(db, "INSERT INTO job_titles (title, department) VALUES (?, ?)", jt.title, jt.dept)
	}

	fmt.Println("db/seed.db created successfully")
}

func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		log.Fatalf("exec %q: %v", query, err)
	}
}
