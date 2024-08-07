package schema

// FieldMapping contains mapping information between a struct field and database column
type FieldMapping struct {
	Name         string         // Name of the field in the struct
	Column       string         // Name of the database column
	Index        int            // Index of the field within a struct
	IsReadOnly   bool           // Is only for select queries
	IsPrimaryKey bool           // Is a pk field
	ForeignKey   *ForeignKey    // Foreign key meta data
	Schema       *StructMapping // Embeded schema
}

// ForeignKey represents foreign key field metadata
type ForeignKey struct {
	Table  string // Foreign table name
	Column string // Foreign table column
}

// HasSchema returns true when the field contains an embeded schema
func (f *FieldMapping) HasSchema() bool {
	return f.Schema != nil
}

// IsWriteable is true when the fields value can be included in an insert or update statement
func (f *FieldMapping) IsWriteable() bool {
	return !f.IsReadOnly && !f.IsPrimaryKey
}

type FieldMappings []FieldMapping

// Find recursively searches for the field that matches the predicate and returns the field along with the index path
func (fields FieldMappings) Find(predicate func(*FieldMapping) bool) (*FieldMapping, []int, error) {
	var index []int

	for _, field := range fields {
		if predicate(&field) {
			index = append(index, field.Index)
			return &field, index, nil
		}

		// recursively look through embeded schemas
		if field.HasSchema() {
			index = append(index, field.Index)
			f, indexes, err := field.Schema.Fields.Find(predicate)
			if err != nil {
				break
			}

			index = append(index, indexes...)
			return f, index, nil
		}
	}

	return nil, nil, ErrFieldNotFound
}

// FindByColumn returns the field and index which has the given column name
func (fields FieldMappings) FindByColumn(col string) (*FieldMapping, []int, error) {
	return fields.Find(func(f *FieldMapping) bool {
		return f.Column == col
	})
}

// FindPK returns the first identity field found
func (fields FieldMappings) FindPK() (*FieldMapping, []int, error) {
	return fields.Find(func(f *FieldMapping) bool {
		return f.IsPrimaryKey
	})
}

// FindFKS are fields representing foreign keys
func (fields FieldMappings) FindFKS() FieldMappings {
	info := []FieldMapping{}
	for _, f := range fields {
		if f.ForeignKey != nil {
			info = append(info, f)
		}
	}
	return info
}

// Writeable returns all fields excluding identity and readonly
func (fields FieldMappings) Writeable() FieldMappings {
	var ret FieldMappings

	for _, field := range fields {
		if !field.IsWriteable() {
			continue
		}

		if field.HasSchema() {
			fs := field.Schema.Fields.Writeable()
			ret = append(ret, fs...)
			continue
		}

		ret = append(ret, field)
	}

	return ret
}

// Columns recursively maps through fields and returns their column names
func (fields FieldMappings) Columns() (columns Columns) {
	for _, f := range fields {
		if f.HasSchema() {
			columns = append(columns, f.Schema.Fields.Columns()...)
			continue
		}

		columns = append(columns, f.Column)
	}
	return
}
