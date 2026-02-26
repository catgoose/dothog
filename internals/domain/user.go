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

	if graphUser.GivenName != "" {
		u.GivenName = sql.NullString{String: graphUser.GivenName, Valid: true}
	}
	if graphUser.Surname != "" {
		u.Surname = sql.NullString{String: graphUser.Surname, Valid: true}
	}
	if graphUser.DisplayName != "" {
		u.DisplayName = sql.NullString{String: graphUser.DisplayName, Valid: true}
	}
	if graphUser.Mail != "" {
		u.Mail = sql.NullString{String: graphUser.Mail, Valid: true}
	}
	if graphUser.JobTitle != "" {
		u.JobTitle = sql.NullString{String: graphUser.JobTitle, Valid: true}
	}
	if graphUser.OfficeLocation != "" {
		u.OfficeLocation = sql.NullString{String: graphUser.OfficeLocation, Valid: true}
	}
	if graphUser.Department != "" {
		u.Department = sql.NullString{String: graphUser.Department, Valid: true}
	}
	if graphUser.CompanyName != "" {
		u.CompanyName = sql.NullString{String: graphUser.CompanyName, Valid: true}
	}
	if graphUser.AccountName != "" {
		u.AccountName = sql.NullString{String: graphUser.AccountName, Valid: true}
	}
}
