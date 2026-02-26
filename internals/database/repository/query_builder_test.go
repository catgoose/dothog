// setup:feature:database
package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPaginationClause(t *testing.T) {
	clause := BuildPaginationClause(10, 25)
	assert.Equal(t, "OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY", clause)
}

func TestBuildSearchPattern(t *testing.T) {
	assert.Empty(t, BuildSearchPattern(""))
	assert.Equal(t, "%foo%", BuildSearchPattern("foo"))
	assert.Equal(t, "%search term%", BuildSearchPattern("search term"))
}

func TestBuildSearchCondition_NoFields(t *testing.T) {
	assert.Equal(t, "1=1", BuildSearchCondition("foo", "%foo%"))
}

func TestBuildSearchCondition_EmptySearch(t *testing.T) {
	assert.Equal(t, "1=1", BuildSearchCondition("", "%foo%", "Name", "Email"))
}

func TestBuildSearchCondition_WithFields(t *testing.T) {
	cond := BuildSearchCondition("foo", "%foo%", "Name", "Email")
	assert.Contains(t, cond, "Name LIKE @SearchPattern")
	assert.Contains(t, cond, "Email LIKE @SearchPattern")
	assert.Contains(t, cond, "@Search = ''")
	assert.Contains(t, cond, " OR ")
}

func TestBuildOrderByClause_EmptySortStr_WithDefault(t *testing.T) {
	columnMap := map[string]string{"name": "Name", "date": "CreatedAt"}
	assert.Equal(t, "ORDER BY CreatedAt ASC", BuildOrderByClause("", columnMap, "CreatedAt ASC"))
}

func TestBuildOrderByClause_EmptySortStr_NoDefault(t *testing.T) {
	columnMap := map[string]string{"name": "Name"}
	assert.Empty(t, BuildOrderByClause("", columnMap, ""))
}

func TestBuildOrderByClause_ValidSort(t *testing.T) {
	columnMap := map[string]string{"name": "Name", "date": "CreatedAt"}
	clause := BuildOrderByClause("name:asc,date:desc", columnMap, "CreatedAt ASC")
	assert.Contains(t, clause, "ORDER BY ")
	assert.Contains(t, clause, "Name ASC")
	assert.Contains(t, clause, "CreatedAt DESC")
}

func TestBuildOrderByClause_UnknownColumnSkipped(t *testing.T) {
	columnMap := map[string]string{"name": "Name"}
	clause := BuildOrderByClause("name:asc,unknown:desc", columnMap, "Name ASC")
	assert.Contains(t, clause, "Name ASC")
	assert.NotContains(t, clause, "unknown")
}

func TestBuildOrderByClause_InvalidDirection_Skipped(t *testing.T) {
	columnMap := map[string]string{"name": "Name"}
	clause := BuildOrderByClause("name:invalid", columnMap, "")
	assert.Empty(t, clause)
}
