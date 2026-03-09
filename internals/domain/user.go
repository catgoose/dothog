// setup:feature:graph

package domain

import (
	"database/sql"
	"time"
)


// User represents a user in the system, storing Azure user information
type User struct {
	UpdatedAt         time.Time      `db:"UpdatedAt" json:"updated_at"`
	CreatedAt         time.Time      `db:"CreatedAt" json:"created_at"`
	LastLoginAt       sql.NullTime   `db:"LastLoginAt" json:"last_login_at,omitempty"`
	UserPrincipalName string         `db:"UserPrincipalName" json:"user_principal_name"`
	AzureID           string         `db:"AzureId" json:"azure_id"`
	Mail              sql.NullString `db:"Mail" json:"mail,omitempty"`
	JobTitle          sql.NullString `db:"JobTitle" json:"job_title,omitempty"`
	OfficeLocation    sql.NullString `db:"OfficeLocation" json:"office_location,omitempty"`
	Department        sql.NullString `db:"Department" json:"department,omitempty"`
	CompanyName       sql.NullString `db:"CompanyName" json:"company_name,omitempty"`
	AccountName       sql.NullString `db:"AccountName" json:"account_name,omitempty"`
	DisplayName       sql.NullString `db:"DisplayName" json:"display_name,omitempty"`
	Surname           sql.NullString `db:"Surname" json:"surname,omitempty"`
	GivenName         sql.NullString `db:"GivenName" json:"given_name,omitempty"`
	ID                int            `db:"ID" json:"id"`
}

// FromGraphUser creates a User from a GraphUser
func (u *User) FromGraphUser(graphUser *GraphUser) {
	u.AzureID = graphUser.AzureID
	u.UserPrincipalName = graphUser.UserPrincipalName
	u.GivenName = ToNullString(graphUser.GivenName)
	u.Surname = ToNullString(graphUser.Surname)
	u.DisplayName = ToNullString(graphUser.DisplayName)
	u.Mail = ToNullString(graphUser.Mail)
	u.JobTitle = ToNullString(graphUser.JobTitle)
	u.OfficeLocation = ToNullString(graphUser.OfficeLocation)
	u.Department = ToNullString(graphUser.Department)
	u.CompanyName = ToNullString(graphUser.CompanyName)
	u.AccountName = ToNullString(graphUser.AccountName)
}
