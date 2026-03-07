package schema

// IndexDef defines a table index.
type IndexDef struct {
	name    string
	columns string
}

// Index creates a new index definition.
func Index(name, columns string) IndexDef {
	return IndexDef{name: name, columns: columns}
}
