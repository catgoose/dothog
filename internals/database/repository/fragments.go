// setup:feature:database

package repository

import (
	"database/sql"
	"fmt"
	"strings"
)

// Columns joins column names into a comma-separated list.
//
//	Columns("ID", "Name", "Email") => "ID, Name, Email"
func Columns(cols ...string) string {
	return strings.Join(cols, ", ")
}

// Placeholders returns named placeholders for the given columns.
//
//	Placeholders("ID", "Name", "Email") => "@ID, @Name, @Email"
func Placeholders(cols ...string) string {
	ps := make([]string, len(cols))
	for i, c := range cols {
		ps[i] = "@" + c
	}
	return strings.Join(ps, ", ")
}

// SetClause builds a SET fragment for UPDATE statements.
//
//	SetClause("Name", "Email") => "Name = @Name, Email = @Email"
func SetClause(cols ...string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s = @%s", c, c)
	}
	return strings.Join(parts, ", ")
}

// InsertInto builds a full INSERT INTO … VALUES … statement.
//
//	InsertInto("Users", "Name", "Email") =>
//	  "INSERT INTO Users (Name, Email) VALUES (@Name, @Email)"
func InsertInto(table string, cols ...string) string {
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, Columns(cols...), Placeholders(cols...))
}

// NamedArgs converts a map to a slice of sql.NamedArg values suitable for
// passing to database/sql query methods.
func NamedArgs(m map[string]any) []any {
	args := make([]any, 0, len(m))
	for k, v := range m {
		args = append(args, sql.Named(k, v))
	}
	return args
}
