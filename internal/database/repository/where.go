// setup:feature:database

package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

// WhereBuilder constructs composable WHERE clauses with named parameters.
type WhereBuilder struct {
	clauses []string
	args    []any
}

// NewWhere creates a new WhereBuilder.
func NewWhere() *WhereBuilder {
	return &WhereBuilder{}
}

// And adds an AND condition with optional named args.
func (w *WhereBuilder) And(condition string, args ...any) *WhereBuilder {
	if len(w.clauses) == 0 {
		w.clauses = append(w.clauses, condition)
	} else {
		w.clauses = append(w.clauses, "AND "+condition)
	}
	w.args = append(w.args, args...)
	return w
}

// AndIf adds an AND condition only when ok is true.
func (w *WhereBuilder) AndIf(ok bool, condition string, args ...any) *WhereBuilder {
	if !ok {
		return w
	}
	return w.And(condition, args...)
}

// Or adds an OR branch to the previous condition.
func (w *WhereBuilder) Or(condition string, args ...any) *WhereBuilder {
	if len(w.clauses) == 0 {
		w.clauses = append(w.clauses, condition)
	} else {
		w.clauses = append(w.clauses, "OR "+condition)
	}
	w.args = append(w.args, args...)
	return w
}

// OrIf adds an OR condition only when ok is true.
func (w *WhereBuilder) OrIf(ok bool, condition string, args ...any) *WhereBuilder {
	if !ok {
		return w
	}
	return w.Or(condition, args...)
}

// Search bridges to the existing BuildSearchCondition / BuildSearchPattern helpers.
func (w *WhereBuilder) Search(search string, fields ...string) *WhereBuilder {
	if search == "" || len(fields) == 0 {
		return w
	}
	pattern := BuildSearchPattern(search)
	cond := BuildSearchCondition(search, pattern, fields...)
	w.And(cond,
		sql.Named("Search", search),
		sql.Named("SearchPattern", pattern),
	)
	return w
}

// String returns the full WHERE clause or an empty string when no conditions exist.
func (w *WhereBuilder) String() string {
	if len(w.clauses) == 0 {
		return ""
	}
	return fmt.Sprintf("WHERE %s", strings.Join(w.clauses, " "))
}

// Args returns the collected named arguments.
func (w *WhereBuilder) Args() []any {
	return w.args
}

// HasConditions reports whether any conditions have been added.
func (w *WhereBuilder) HasConditions() bool {
	return len(w.clauses) > 0
}

// NotDeleted adds a "DeletedAt IS NULL" condition for soft-delete filtering.
func (w *WhereBuilder) NotDeleted() *WhereBuilder {
	return w.And("DeletedAt IS NULL")
}

// NotExpired adds a condition that filters out expired records.
func (w *WhereBuilder) NotExpired() *WhereBuilder {
	return w.And("(ExpiresAt IS NULL OR ExpiresAt > CURRENT_TIMESTAMP)")
}

// HasStatus adds a "Status = @Status" condition.
func (w *WhereBuilder) HasStatus(status string) *WhereBuilder {
	return w.And("Status = @Status", sql.Named("Status", status))
}

// IsRoot adds a "ParentID IS NULL" condition for tree root nodes.
func (w *WhereBuilder) IsRoot() *WhereBuilder {
	return w.And("ParentID IS NULL")
}

// HasParent adds a "ParentID = @ParentID" condition.
func (w *WhereBuilder) HasParent(parentID int64) *WhereBuilder {
	return w.And("ParentID = @ParentID", sql.Named("ParentID", parentID))
}

// NotReplaced adds a "ReplacedByID IS NULL" condition for current (non-replaced) records.
func (w *WhereBuilder) NotReplaced() *WhereBuilder {
	return w.And("ReplacedByID IS NULL")
}

// ReplacedBy adds a "ReplacedByID = @ReplacedByID" condition.
func (w *WhereBuilder) ReplacedBy(id int64) *WhereBuilder {
	return w.And("ReplacedByID = @ReplacedByID", sql.Named("ReplacedByID", id))
}

// NotArchived adds an "ArchivedAt IS NULL" condition for archive filtering.
func (w *WhereBuilder) NotArchived() *WhereBuilder {
	return w.And("ArchivedAt IS NULL")
}

// HasVersion adds a "Version = @Version" condition for optimistic locking.
func (w *WhereBuilder) HasVersion(version int) *WhereBuilder {
	return w.And("Version = @Version", sql.Named("Version", version))
}
