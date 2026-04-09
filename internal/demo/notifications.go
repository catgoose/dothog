// setup:feature:demo

package demo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
)

// NotificationIdentity represents a user identity for the notifications demo.
type NotificationIdentity struct {
	ID    string
	Name  string
	Color string // avatar color
}

// notificationIdentityPool is a fixed set of identities assigned to visitors.
var notificationIdentityPool = []NotificationIdentity{
	{ID: "u01", Name: "Captain Haddock", Color: "#3b82f6"},
	{ID: "u02", Name: "Ada Lovelace", Color: "#ec4899"},
	{ID: "u03", Name: "Pixel Pete", Color: "#22c55e"},
	{ID: "u04", Name: "Bug Hunter", Color: "#ef4444"},
	{ID: "u05", Name: "Null Pointer", Color: "#f97316"},
	{ID: "u06", Name: "Query Queen", Color: "#8b5cf6"},
	{ID: "u07", Name: "Stack Overflow", Color: "#06b6d4"},
	{ID: "u08", Name: "Merge Conflict", Color: "#eab308"},
	{ID: "u09", Name: "Deploy Dog", Color: "#14b8a6"},
	{ID: "u10", Name: "Cache Miss", Color: "#f43f5e"},
	{ID: "u11", Name: "Byte Bandit", Color: "#a855f7"},
	{ID: "u12", Name: "Syntax Error", Color: "#64748b"},
}

// NotificationCategory defines the notification types.
type NotificationCategory string

// Notification category constants.
const (
	CatOrder   NotificationCategory = "order"
	CatMention NotificationCategory = "mention"
	CatAlert   NotificationCategory = "alert"
	CatSystem  NotificationCategory = "system"
)

// AllNotificationCategories lists every category.
var AllNotificationCategories = []NotificationCategory{
	CatOrder, CatMention, CatAlert, CatSystem,
}

// NotificationMessages maps categories to pools of message templates.
var NotificationMessages = map[NotificationCategory][]string{
	CatOrder: {
		"Order #%d shipped via express",
		"Order #%d confirmed and processing",
		"Order #%d delivered successfully",
		"Order #%d refund processed",
		"Order #%d payment received",
	},
	CatMention: {
		"@you mentioned in Project Alpha",
		"Tagged in code review #%d",
		"Mentioned in standup notes",
		"@you replied to in thread #%d",
		"Tagged in design discussion",
	},
	CatAlert: {
		"CPU usage above 90%% on worker-%d",
		"Disk space low on volume /data",
		"Memory usage critical: %d%% used",
		"Response time spike detected",
		"Error rate exceeded threshold",
	},
	CatSystem: {
		"Maintenance scheduled for tonight",
		"New version v2.%d.0 available",
		"Database migration completed",
		"SSL certificate renewed",
		"Backup completed successfully",
	},
}

// NotificationFilters tracks per-user category filter settings.
type NotificationFilters struct {
	filters map[string]map[NotificationCategory]bool
	mu      sync.RWMutex
}

// NewNotificationFilters creates a new filter store.
func NewNotificationFilters() *NotificationFilters {
	return &NotificationFilters{
		filters: make(map[string]map[NotificationCategory]bool),
	}
}

// SetFilter enables or disables a category for a user.
func (nf *NotificationFilters) SetFilter(userID string, cat NotificationCategory, enabled bool) {
	nf.mu.Lock()
	defer nf.mu.Unlock()
	if _, ok := nf.filters[userID]; !ok {
		nf.filters[userID] = make(map[NotificationCategory]bool)
		for _, c := range AllNotificationCategories {
			nf.filters[userID][c] = true
		}
	}
	nf.filters[userID][cat] = enabled
}

// IsEnabled checks if a category is enabled for a user. Defaults to true.
func (nf *NotificationFilters) IsEnabled(userID string, cat NotificationCategory) bool {
	nf.mu.RLock()
	defer nf.mu.RUnlock()
	if m, ok := nf.filters[userID]; ok {
		if enabled, exists := m[cat]; exists {
			return enabled
		}
	}
	return true
}

// EnabledCategories returns all enabled categories for a user.
func (nf *NotificationFilters) EnabledCategories(userID string) map[NotificationCategory]bool {
	nf.mu.RLock()
	defer nf.mu.RUnlock()
	result := make(map[NotificationCategory]bool)
	for _, c := range AllNotificationCategories {
		result[c] = true
	}
	if m, ok := nf.filters[userID]; ok {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// AssignIdentity returns an identity for a given index (wraps around the pool).
func AssignIdentity(index int) NotificationIdentity {
	return notificationIdentityPool[index%len(notificationIdentityPool)]
}

// AllNotificationIdentities returns the full identity pool.
func AllNotificationIdentities() []NotificationIdentity {
	out := make([]NotificationIdentity, len(notificationIdentityPool))
	copy(out, notificationIdentityPool)
	return out
}

// IdentityIndexByID returns the pool index for a given identity ID, or -1.
func IdentityIndexByID(id string) int {
	for i, ident := range notificationIdentityPool {
		if ident.ID == id {
			return i
		}
	}
	return -1
}

// RandomIdentityIndex returns a random index into the identity pool.
func RandomIdentityIndex() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(notificationIdentityPool))))
	return int(n.Int64())
}

// GenerateNotifID returns a short random ID for a notification.
func GenerateNotifID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// FormatNotification renders a random message for the given category.
func FormatNotification(cat NotificationCategory) string {
	msgs := NotificationMessages[cat]
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(msgs))))
	tmpl := msgs[n.Int64()]
	num, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf(tmpl, num.Int64()+1000)
}
