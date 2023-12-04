package schema

import (
	"fmt"
	"strings"
)

// Columns is an alias for []string that contains methods that faciliate formatting sql strings
type Columns []string

// List returns a comma seperated string of columns
func (c Columns) List() string {
	return strings.Join(c, ", ")
}

// ValueList returns a postgres parameter list in the format of $1, $2, ...
// The start value determines when counting starts.
func (c Columns) ValueList(start int) string {
	var parts []string
	for i := range c {
		parts = append(parts, fmt.Sprintf("$%d", start+i))
	}
	return strings.Join(parts, ", ")
}

// PrefixedList returns a List where each column is prefixed by the given argument.
// A '.' is automatically added to the prefix
func (c Columns) PrefixedList(prefix string) string {
	var cols []string
	for _, col := range c {
		cols = append(cols, prefix+"."+col)
	}
	return strings.Join(cols, ", ")
}

// AssignmentList returns an assignment list in the format of column1 = $1, column2 = $2, ...
// The start argument determines the initial number for the parameters.
func (c Columns) AssignmentList(start int) string {
	var assignments []string
	for i, col := range c {
		assignments = append(assignments, fmt.Sprintf("%s = $%d", col, start+i))
	}
	return strings.Join(assignments, ", ")
}
