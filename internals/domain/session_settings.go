// setup:feature:session_settings

package domain

import "time"

// SessionSettings holds user preferences keyed by a browser UUID cookie.
type SessionSettings struct {
	SessionUUID string    `db:"SessionUUID"`
	Theme       string    `db:"Theme"`
	UpdatedAt   time.Time `db:"UpdatedAt"`
	ID          int       `db:"Id"`
}

// Default settings values.
const DefaultTheme = "light"

// NewDefaultSettings returns a SessionSettings with defaults for the given UUID.
func NewDefaultSettings(uuid string) *SessionSettings {
	return &SessionSettings{
		SessionUUID: uuid,
		Theme:       DefaultTheme,
	}
}
