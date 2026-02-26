-- Create Users table to store Azure user information
-- Updated on login to keep user data current

CREATE TABLE Users (
    ID INT PRIMARY KEY IDENTITY(1,1),
    AzureId VARCHAR(255) NOT NULL UNIQUE,
    GivenName NVARCHAR(255),
    Surname NVARCHAR(255),
    DisplayName NVARCHAR(255),
    UserPrincipalName NVARCHAR(255) NOT NULL,
    Mail NVARCHAR(255),
    JobTitle NVARCHAR(255),
    OfficeLocation NVARCHAR(255),
    Department NVARCHAR(255),
    CompanyName NVARCHAR(255),
    AccountName NVARCHAR(255),
    LastLoginAt DATETIME,
    CreatedAt DATETIME NOT NULL DEFAULT GETDATE(),
    UpdatedAt DATETIME NOT NULL DEFAULT GETDATE()
);

-- Create indexes for better performance
CREATE INDEX idx_users_azureid ON Users(AzureId);
CREATE INDEX idx_users_userprincipalname ON Users(UserPrincipalName);
CREATE INDEX idx_users_displayname ON Users(DisplayName);
CREATE INDEX idx_users_mail ON Users(Mail);
CREATE INDEX idx_users_lastloginat ON Users(LastLoginAt);
