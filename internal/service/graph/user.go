// setup:feature:graph

package graph

import (
	"database/sql"
	"time"
)

// User is the chuck-backed persisted row for an Azure-cached Graph user.
// Nullable Graph attributes use sql.Null* so JSON omitzero suppresses unset
// values. Graph-owned: the shape mirrors UsersTable and is populated from
// GraphUser via FromGraphUser.
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

// FromGraphUser copies fields from graphUser into u, wrapping strings as NullString.
func (u *User) FromGraphUser(graphUser *GraphUser) {
	u.AzureID = graphUser.AzureID
	u.UserPrincipalName = graphUser.UserPrincipalName
	u.GivenName = toNullString(graphUser.GivenName)
	u.Surname = toNullString(graphUser.Surname)
	u.DisplayName = toNullString(graphUser.DisplayName)
	u.Mail = toNullString(graphUser.Mail)
	u.JobTitle = toNullString(graphUser.JobTitle)
	u.OfficeLocation = toNullString(graphUser.OfficeLocation)
	u.Department = toNullString(graphUser.Department)
	u.CompanyName = toNullString(graphUser.CompanyName)
	u.AccountName = toNullString(graphUser.AccountName)
}

// toNullString treats an empty string as NULL.
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
