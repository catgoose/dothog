// setup:feature:graph

package domain

import (
	"database/sql"
	"time"
)

// User represents a user in the system, storing Azure user information
type User struct {
	UpdatedAt         time.Time      `db:"UpdatedAt" json:"updatedAt"`
	CreatedAt         time.Time      `db:"CreatedAt" json:"createdAt"`
	LastLoginAt       sql.NullTime   `db:"LastLoginAt" json:"lastLoginAt,omitempty"`
	UserPrincipalName string         `db:"UserPrincipalName" json:"userPrincipalName"`
	AzureID           string         `db:"AzureId" json:"azureId"`
	Mail              sql.NullString `db:"Mail" json:"mail,omitempty"`
	JobTitle          sql.NullString `db:"JobTitle" json:"jobTitle,omitempty"`
	OfficeLocation    sql.NullString `db:"OfficeLocation" json:"officeLocation,omitempty"`
	Department        sql.NullString `db:"Department" json:"department,omitempty"`
	CompanyName       sql.NullString `db:"CompanyName" json:"companyName,omitempty"`
	AccountName       sql.NullString `db:"AccountName" json:"accountName,omitempty"`
	DisplayName       sql.NullString `db:"DisplayName" json:"displayName,omitempty"`
	Surname           sql.NullString `db:"Surname" json:"surname,omitempty"`
	GivenName         sql.NullString `db:"GivenName" json:"givenName,omitempty"`
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
