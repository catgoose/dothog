// setup:feature:demo

package demo

import "sync"

// SettingField is one editable control in a settings form; Options is used only when Kind is "select".
type SettingField struct {
	Key     string
	Label   string
	Value   string
	Kind    string   // text, toggle, select, textarea
	Options []string // used when Kind is "select"
}

// SettingsSection groups related SettingField entries into a tab/panel rendered by the settings page.
type SettingsSection struct {
	ID          string
	Title       string
	Description string
	Fields      []SettingField
}

// SettingsStore is a thread-safe in-memory store for application settings.
type SettingsStore struct {
	sections []SettingsSection
	mu       sync.RWMutex
}

// NewSettingsStore pre-seeds the four standard sections: general, notifications, security, appearance.
func NewSettingsStore() *SettingsStore {
	return &SettingsStore{
		sections: []SettingsSection{
			{
				ID:          "general",
				Title:       "General",
				Description: "Basic application settings",
				Fields: []SettingField{
					{Key: "app_name", Label: "App Name", Value: "", Kind: "text"},
					{Key: "timezone", Label: "Timezone", Value: "UTC", Kind: "select", Options: []string{"UTC", "US/Eastern", "US/Pacific", "Europe/London"}},
					{Key: "language", Label: "Language", Value: "en", Kind: "select", Options: []string{"en", "es", "fr", "de"}},
				},
			},
			{
				ID:          "notifications",
				Title:       "Notifications",
				Description: "Configure how and when you receive notifications",
				Fields: []SettingField{
					{Key: "email_notifications", Label: "Email Notifications", Value: "false", Kind: "toggle"},
					{Key: "slack_integration", Label: "Slack Integration", Value: "false", Kind: "toggle"},
					{Key: "digest_frequency", Label: "Digest Frequency", Value: "daily", Kind: "select", Options: []string{"daily", "weekly", "monthly"}},
				},
			},
			{
				ID:          "security",
				Title:       "Security",
				Description: "Security and access control settings",
				Fields: []SettingField{
					{Key: "two_factor_auth", Label: "Two-Factor Auth", Value: "false", Kind: "toggle"},
					{Key: "session_timeout", Label: "Session Timeout", Value: "30m", Kind: "select", Options: []string{"15m", "30m", "1h", "4h"}},
					{Key: "ip_allowlist", Label: "IP Allowlist", Value: "", Kind: "textarea"},
				},
			},
			{
				ID:          "appearance",
				Title:       "Appearance",
				Description: "Customize the look and feel",
				Fields: []SettingField{
					{Key: "theme", Label: "Theme", Value: "system", Kind: "select", Options: []string{"light", "dark", "system"}},
					{Key: "compact_mode", Label: "Compact Mode", Value: "false", Kind: "toggle"},
					{Key: "show_avatars", Label: "Show Avatars", Value: "true", Kind: "toggle"},
				},
			},
		},
	}
}

// GetSection looks up by ID; ok is false (and the section is zero) when not found.
func (s *SettingsStore) GetSection(id string) (SettingsSection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sec := range s.sections {
		if sec.ID == id {
			return sec, true
		}
	}
	return SettingsSection{}, false
}

// AllSections is a defensive copy; safe to mutate independently.
func (s *SettingsStore) AllSections() []SettingsSection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SettingsSection, len(s.sections))
	copy(out, s.sections)
	return out
}

// UpdateSection applies the provided key/value pairs to the matching section's
// fields and returns the updated section. It returns false if the section ID is
// not found.
func (s *SettingsStore) UpdateSection(id string, values map[string]string) (SettingsSection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sec := range s.sections {
		if sec.ID == id {
			for j, f := range sec.Fields {
				if v, ok := values[f.Key]; ok {
					s.sections[i].Fields[j].Value = v
				}
			}
			return s.sections[i], true
		}
	}
	return SettingsSection{}, false
}
