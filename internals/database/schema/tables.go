package schema

var UsersTable = NewTable("Users").
	Columns(
		AutoIncrCol("ID"),
		Col("AzureId", TypeVarchar(255)).NotNull().Unique(),
		Col("GivenName", TypeString(255)),
		Col("Surname", TypeString(255)),
		Col("DisplayName", TypeString(255)),
		Col("UserPrincipalName", TypeString(255)).NotNull(),
		Col("Mail", TypeString(255)),
		Col("JobTitle", TypeString(255)),
		Col("OfficeLocation", TypeString(255)),
		Col("Department", TypeString(255)),
		Col("CompanyName", TypeString(255)),
		Col("AccountName", TypeString(255)),
		Col("LastLoginAt", TypeTimestamp()),
	).
	WithTimestamps().
	Indexes(
		Index("idx_users_azureid", "AzureId"),
		Index("idx_users_userprincipalname", "UserPrincipalName"),
		Index("idx_users_displayname", "DisplayName"),
		Index("idx_users_mail", "Mail"),
		Index("idx_users_lastloginat", "LastLoginAt"),
	)
