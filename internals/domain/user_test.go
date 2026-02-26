// setup:feature:graph
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_FromGraphUser_AllFields(t *testing.T) {
	u := &User{}
	g := &GraphUser{
		AzureID:           "azure-123",
		UserPrincipalName: "user@example.com",
		GivenName:         "John",
		Surname:           "Doe",
		DisplayName:       "John Doe",
		Mail:              "john@example.com",
		JobTitle:          "Engineer",
		OfficeLocation:    "Seattle",
		Department:        "Engineering",
		CompanyName:       "Acme",
		AccountName:       "jdoe",
	}
	u.FromGraphUser(g)

	assert.Equal(t, "azure-123", u.AzureID)
	assert.Equal(t, "user@example.com", u.UserPrincipalName)
	require.True(t, u.GivenName.Valid)
	assert.Equal(t, "John", u.GivenName.String)
	require.True(t, u.Surname.Valid)
	assert.Equal(t, "Doe", u.Surname.String)
	require.True(t, u.DisplayName.Valid)
	assert.Equal(t, "John Doe", u.DisplayName.String)
	require.True(t, u.Mail.Valid)
	assert.Equal(t, "john@example.com", u.Mail.String)
	require.True(t, u.JobTitle.Valid)
	assert.Equal(t, "Engineer", u.JobTitle.String)
	require.True(t, u.OfficeLocation.Valid)
	assert.Equal(t, "Seattle", u.OfficeLocation.String)
	require.True(t, u.Department.Valid)
	assert.Equal(t, "Engineering", u.Department.String)
	require.True(t, u.CompanyName.Valid)
	assert.Equal(t, "Acme", u.CompanyName.String)
	require.True(t, u.AccountName.Valid)
	assert.Equal(t, "jdoe", u.AccountName.String)
}

func TestUser_FromGraphUser_EmptyFields(t *testing.T) {
	u := &User{}
	g := &GraphUser{
		AzureID:           "azure-456",
		UserPrincipalName: "upn@example.com",
	}
	u.FromGraphUser(g)

	assert.Equal(t, "azure-456", u.AzureID)
	assert.Equal(t, "upn@example.com", u.UserPrincipalName)
	assert.False(t, u.GivenName.Valid)
	assert.False(t, u.Surname.Valid)
	assert.False(t, u.DisplayName.Valid)
	assert.False(t, u.Mail.Valid)
	assert.False(t, u.JobTitle.Valid)
	assert.False(t, u.OfficeLocation.Valid)
	assert.False(t, u.Department.Valid)
	assert.False(t, u.CompanyName.Valid)
	assert.False(t, u.AccountName.Valid)
}

func TestUser_FromGraphUser_Partial(t *testing.T) {
	u := &User{}
	g := &GraphUser{
		AzureID:           "azure-789",
		UserPrincipalName: "a@b.com",
		GivenName:         "Alice",
		DisplayName:       "Alice Smith",
	}
	u.FromGraphUser(g)

	assert.Equal(t, "azure-789", u.AzureID)
	assert.Equal(t, "a@b.com", u.UserPrincipalName)
	require.True(t, u.GivenName.Valid)
	assert.Equal(t, "Alice", u.GivenName.String)
	assert.False(t, u.Surname.Valid)
	require.True(t, u.DisplayName.Valid)
	assert.Equal(t, "Alice Smith", u.DisplayName.String)
	assert.False(t, u.Mail.Valid)
	assert.False(t, u.JobTitle.Valid)
}
