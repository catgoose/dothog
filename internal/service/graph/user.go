// setup:feature:graph

package graph

// User mirrors the Microsoft Graph /users payload with extra db tags for
// local persistence in the Graph directory cache.
type User struct {
	AzureID           string `json:"id" db:"AzureId"`
	GivenName         string `json:"givenName" db:"GivenName"`
	Surname           string `json:"surname" db:"Surname"`
	DisplayName       string `json:"displayName" db:"DisplayName"`
	UserPrincipalName string `json:"userPrincipalName" db:"UserPrincipalName"`
	Mail              string `json:"mail" db:"Mail"`
	JobTitle          string `json:"jobTitle" db:"JobTitle"`
	OfficeLocation    string `json:"officeLocation" db:"OfficeLocation"`
	Department        string `json:"department" db:"Department"`
	CompanyName       string `json:"companyName" db:"CompanyName"`
	AccountName       string `db:"AccountName"`
}
