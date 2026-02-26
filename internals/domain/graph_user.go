// setup:feature:graph

package domain

// GraphUser represents a user from Microsoft Graph API
type GraphUser struct {
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
