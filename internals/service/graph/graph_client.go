// setup:feature:graph

package graph

import (
	"context"
	"fmt"
	"strings"

	"catgoose/go-htmx-demo/internals/domain"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/microsoft/kiota-abstractions-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

// Client wraps the Microsoft Graph SDK client for app-only access.
type Client struct {
	client *msgraphsdk.GraphServiceClient
}

// NewGraphClient creates a Graph client using client credentials (tenant ID, client ID, client secret).
func NewGraphClient(tenantID, clientID, clientSecret string) (*Client, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("client secret credential: %w", err)
	}
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return nil, fmt.Errorf("graph service client: %w", err)
	}
	return &Client{client: client}, nil
}

func ptr[T any](v T) *T {
	return &v
}

func consistencyLevelHeaders() *abstractions.RequestHeaders {
	h := abstractions.NewRequestHeaders()
	h.TryAdd("ConsistencyLevel", "eventual")
	return h
}

// FetchAllEnabledUsers fetches all enabled users from Microsoft Graph with filter and select equivalent to the previous implementation.
func (c *Client) FetchAllEnabledUsers() ([]domain.GraphUser, error) {
	ctx := context.Background()
	filter := "accountEnabled eq true"
	selectCols := []string{"id", "displayName", "userPrincipalName", "mail", "officeLocation", "department", "givenName", "surname", "companyName", "jobTitle"}
	requestConfig := &users.UsersRequestBuilderGetRequestConfiguration{
		QueryParameters: &users.UsersRequestBuilderGetQueryParameters{
			Filter: &filter,
			Select: selectCols,
			Count:  ptr(true),
			Top:    ptr(int32(999)),
		},
		Headers: consistencyLevelHeaders(),
	}
	result, err := c.client.Users().Get(ctx, requestConfig)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	pageIterator, err := msgraphcore.NewPageIterator[models.Userable](result, c.client.GetAdapter(), models.CreateUserCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, fmt.Errorf("page iterator: %w", err)
	}
	var all []domain.GraphUser
	err = pageIterator.Iterate(ctx, func(user models.Userable) bool {
		all = append(all, userableToGraphUser(user))
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return all, nil
}

func userableToGraphUser(u models.Userable) domain.GraphUser {
	g := domain.GraphUser{}
	if u == nil {
		return g
	}
	if v := u.GetId(); v != nil {
		g.AzureID = *v
	}
	if v := u.GetGivenName(); v != nil {
		g.GivenName = *v
	}
	if v := u.GetSurname(); v != nil {
		g.Surname = *v
	}
	if v := u.GetDisplayName(); v != nil {
		g.DisplayName = *v
	}
	if v := u.GetUserPrincipalName(); v != nil {
		g.UserPrincipalName = *v
		if parts := strings.Split(*v, "@"); len(parts) > 0 {
			g.AccountName = parts[0]
		}
	}
	if v := u.GetMail(); v != nil {
		g.Mail = *v
	}
	if v := u.GetJobTitle(); v != nil {
		g.JobTitle = *v
	}
	if v := u.GetOfficeLocation(); v != nil {
		g.OfficeLocation = *v
	}
	if v := u.GetDepartment(); v != nil {
		g.Department = *v
	}
	if v := u.GetCompanyName(); v != nil {
		g.CompanyName = *v
	}
	return g
}
