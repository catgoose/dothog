// setup:feature:demo

package demo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationFilters_DefaultsEnabled(t *testing.T) {
	nf := NewNotificationFilters()
	for _, cat := range AllNotificationCategories {
		assert.True(t, nf.IsEnabled("u01", cat),
			"category %q should default to enabled", cat)
	}
}

func TestNotificationFilters_SetFilter(t *testing.T) {
	nf := NewNotificationFilters()
	nf.SetFilter("u01", CatOrder, false)

	assert.False(t, nf.IsEnabled("u01", CatOrder),
		"order should be disabled after explicit set")
	assert.True(t, nf.IsEnabled("u01", CatMention),
		"mention should still be enabled")
	assert.True(t, nf.IsEnabled("u02", CatOrder),
		"other users should be unaffected")
}

func TestNotificationFilters_SetFilterPersistsAcrossCategories(t *testing.T) {
	nf := NewNotificationFilters()
	// First SetFilter call seeds all-enabled defaults for the user;
	// subsequent calls only update the named category.
	nf.SetFilter("u01", CatOrder, false)
	nf.SetFilter("u01", CatAlert, false)

	enabled := nf.EnabledCategories("u01")
	assert.False(t, enabled[CatOrder])
	assert.False(t, enabled[CatAlert])
	assert.True(t, enabled[CatMention])
	assert.True(t, enabled[CatSystem])
}

func TestNotificationFilters_EnabledCategoriesIncludesAll(t *testing.T) {
	nf := NewNotificationFilters()
	enabled := nf.EnabledCategories("never-set")
	require.Len(t, enabled, len(AllNotificationCategories))
	for _, cat := range AllNotificationCategories {
		assert.True(t, enabled[cat],
			"category %q should be present and enabled by default", cat)
	}
}

func TestAssignIdentity_WrapsAroundPool(t *testing.T) {
	all := AllNotificationIdentities()
	require.NotEmpty(t, all)

	assert.Equal(t, all[0], AssignIdentity(0))
	assert.Equal(t, all[1], AssignIdentity(1))
	// Index past the end should wrap.
	assert.Equal(t, all[0], AssignIdentity(len(all)))
	assert.Equal(t, all[2], AssignIdentity(len(all)+2))
}

func TestIdentityIndexByID(t *testing.T) {
	all := AllNotificationIdentities()
	for i, ident := range all {
		assert.Equal(t, i, IdentityIndexByID(ident.ID),
			"index for %q should be %d", ident.ID, i)
	}
	assert.Equal(t, -1, IdentityIndexByID("u-not-real"))
	assert.Equal(t, -1, IdentityIndexByID(""))
}

func TestAllNotificationIdentities_ReturnsCopy(t *testing.T) {
	original := AllNotificationIdentities()
	mutated := AllNotificationIdentities()
	mutated[0].Name = "Mutated"
	// The next call should still return the unmodified pool.
	fresh := AllNotificationIdentities()
	assert.Equal(t, original[0].Name, fresh[0].Name)
}
