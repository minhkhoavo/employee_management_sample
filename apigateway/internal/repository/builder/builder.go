package builder

import (
	"fmt"
	"strings"
)

// SQLBuilder helps construct SQL queries dynamically.
type SQLBuilder struct {
	table      string
	columns    []string
	values     []interface{}
	where      []string
	args       []interface{}
	joins      []string
	orderBy    []string
	limit      int
	offset     int
	updateCols []string
	isInsert   bool
	isUpdate   bool
	isDelete   bool
	isSelect   bool
	// New fields for enhancements
	orConditions  []orCondition
	whereGroups   []whereGroup
	rawConditions []rawCondition
}

// orCondition represents an OR condition
type orCondition struct {
	condition string
	args      []interface{}
}

// whereGroup represents a grouped (parenthesized) condition
type whereGroup struct {
	builder *SQLBuilder
}

// rawCondition represents a raw SQL condition
type rawCondition struct {
	sql  string
	args []interface{}
}

// NewSQLBuilder creates a new instance of SQLBuilder.
func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{}
}

// Select specifies the columns to retrieve.
func (b *SQLBuilder) Select(cols ...string) *SQLBuilder {
	b.isSelect = true
	b.columns = cols
	return b
}

// Insert specifies the table and columns for insertion.
func (b *SQLBuilder) Insert(table string, cols ...string) *SQLBuilder {
	b.isInsert = true
	b.table = table
	b.columns = cols
	return b
}

// Update specifies the table to update.
func (b *SQLBuilder) Update(table string) *SQLBuilder {
	b.isUpdate = true
	b.table = table
	return b
}

// Delete specifies the table to delete from.
func (b *SQLBuilder) Delete(table string) *SQLBuilder {
	b.isDelete = true
	b.table = table
	return b
}

// From specifies the table to select from.
func (b *SQLBuilder) From(table string) *SQLBuilder {
	b.table = table
	return b
}

// Set specifies the columns and values for update.
func (b *SQLBuilder) Set(col string, val interface{}) *SQLBuilder {
	b.updateCols = append(b.updateCols, col)
	b.args = append(b.args, val)
	return b
}

// Values specifies the values for insertion.
func (b *SQLBuilder) Values(vals ...interface{}) *SQLBuilder {
	b.values = vals
	b.args = append(b.args, vals...)
	return b
}

// Where adds a condition to the query.
func (b *SQLBuilder) Where(condition string, args ...interface{}) *SQLBuilder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)
	return b
}

// Join adds a JOIN clause.
func (b *SQLBuilder) Join(joinType, table, on string) *SQLBuilder {
	b.joins = append(b.joins, fmt.Sprintf("%s JOIN %s ON %s", joinType, table, on))
	return b
}

// OrderBy adds an ORDER BY clause.
func (b *SQLBuilder) OrderBy(order string) *SQLBuilder {
	b.orderBy = append(b.orderBy, order)
	return b
}

// Limit adds a LIMIT clause.
func (b *SQLBuilder) Limit(limit int) *SQLBuilder {
	b.limit = limit
	return b
}

// Offset adds an OFFSET clause.
func (b *SQLBuilder) Offset(offset int) *SQLBuilder {
	b.offset = offset
	return b
}

// Or adds an OR condition to the query.
func (b *SQLBuilder) Or(condition string, args ...interface{}) *SQLBuilder {
	b.orConditions = append(b.orConditions, orCondition{
		condition: condition,
		args:      args,
	})
	return b
}

// WhereGroup adds a grouped (parenthesized) WHERE condition.
// The provided function receives a new SQLBuilder for building the grouped conditions.
func (b *SQLBuilder) WhereGroup(fn func(*SQLBuilder) *SQLBuilder) *SQLBuilder {
	groupBuilder := NewSQLBuilder()
	groupBuilder = fn(groupBuilder)
	b.whereGroups = append(b.whereGroups, whereGroup{
		builder: groupBuilder,
	})
	return b
}

// WhereRaw adds a raw SQL condition with arguments.
func (b *SQLBuilder) WhereRaw(sql string, args ...interface{}) *SQLBuilder {
	b.rawConditions = append(b.rawConditions, rawCondition{
		sql:  sql,
		args: args,
	})
	return b
}

// BuildSafe constructs the final SQL string and arguments with safety validation.
// Returns an error if the number of placeholders doesn't match the number of arguments.
func (b *SQLBuilder) BuildSafe() (string, []interface{}, error) {
	sql, args := b.Build()

	// Count the number of placeholder markers in the generated SQL
	// Since Build() replaces "?" with "$1", "$2", etc., we count those
	placeholderCount := 0
	for i := 1; i <= len(args)+10; i++ { // Check up to a reasonable limit
		if strings.Contains(sql, fmt.Sprintf("$%d", i)) {
			placeholderCount++
		} else if i > len(args) {
			break
		}
	}

	if placeholderCount != len(args) {
		return "", nil, fmt.Errorf("placeholder count (%d) does not match argument count (%d)", placeholderCount, len(args))
	}

	return sql, args, nil
}

// Build constructs the final SQL string and arguments.
func (b *SQLBuilder) Build() (string, []interface{}) {
	var sb strings.Builder

	if b.isSelect {
		sb.WriteString("SELECT ")
		sb.WriteString(strings.Join(b.columns, ", "))
		sb.WriteString(" FROM ")
		sb.WriteString(b.table)
		for _, join := range b.joins {
			sb.WriteString(" ")
			sb.WriteString(join)
		}
	} else if b.isInsert {
		sb.WriteString("INSERT INTO ")
		sb.WriteString(b.table)
		sb.WriteString(" (")
		sb.WriteString(strings.Join(b.columns, ", "))
		sb.WriteString(") VALUES (")
		placeholders := make([]string, len(b.values))
		for i := range b.values {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		sb.WriteString(strings.Join(placeholders, ", "))
		sb.WriteString(")")
		return sb.String(), b.args
	} else if b.isUpdate {
		sb.WriteString("UPDATE ")
		sb.WriteString(b.table)
		sb.WriteString(" SET ")
		setClauses := make([]string, len(b.updateCols))
		for i, col := range b.updateCols {
			setClauses[i] = fmt.Sprintf("%s = $%d", col, i+1)
		}
		sb.WriteString(strings.Join(setClauses, ", "))
	} else if b.isDelete {
		sb.WriteString("DELETE FROM ")
		sb.WriteString(b.table)
	}

	// Build WHERE clause with all condition types
	hasWhere := len(b.where) > 0 || len(b.orConditions) > 0 || len(b.whereGroups) > 0 || len(b.rawConditions) > 0

	if hasWhere {
		sb.WriteString(" WHERE ")

		// Adjust placeholders for WHERE clause if needed (for Update, offset by set args)
		offset := 0
		if b.isUpdate {
			offset = len(b.updateCols)
		}

		argIndex := offset + 1
		var conditions []string

		// Process regular WHERE conditions (combined with AND)
		if len(b.where) > 0 {
			whereClause := strings.Join(b.where, " AND ")
			finalWhere := ""
			parts := strings.Split(whereClause, "?")
			for i, part := range parts {
				finalWhere += part
				if i < len(parts)-1 {
					finalWhere += fmt.Sprintf("$%d", argIndex)
					argIndex++
				}
			}
			conditions = append(conditions, finalWhere)
		}

		// Process grouped WHERE conditions (parenthesized)
		for _, group := range b.whereGroups {
			if len(group.builder.where) > 0 || len(group.builder.orConditions) > 0 || len(group.builder.rawConditions) > 0 {
				var groupConditions []string

				// Process regular WHERE in group
				if len(group.builder.where) > 0 {
					whereClause := strings.Join(group.builder.where, " AND ")
					finalWhere := ""
					parts := strings.Split(whereClause, "?")
					for i, part := range parts {
						finalWhere += part
						if i < len(parts)-1 {
							finalWhere += fmt.Sprintf("$%d", argIndex)
							argIndex++
						}
					}
					groupConditions = append(groupConditions, finalWhere)
				}

				// Append group where args to main args
				b.args = append(b.args, group.builder.args...)

				// Process OR conditions in group
				for _, orCond := range group.builder.orConditions {
					finalOr := ""
					parts := strings.Split(orCond.condition, "?")
					for i, part := range parts {
						finalOr += part
						if i < len(parts)-1 {
							finalOr += fmt.Sprintf("$%d", argIndex)
							argIndex++
						}
					}
					groupConditions = append(groupConditions, finalOr)
					// Append OR condition args to main args
					b.args = append(b.args, orCond.args...)
				}

				// Process raw conditions in group
				for _, rawCond := range group.builder.rawConditions {
					finalRaw := ""
					parts := strings.Split(rawCond.sql, "?")
					for i, part := range parts {
						finalRaw += part
						if i < len(parts)-1 {
							finalRaw += fmt.Sprintf("$%d", argIndex)
							argIndex++
						}
					}
					groupConditions = append(groupConditions, finalRaw)
					// Append raw condition args to main args
					b.args = append(b.args, rawCond.args...)
				}

				if len(groupConditions) > 0 {
					// Join with OR since groups can contain mixed WHERE and OR conditions
					groupClause := "(" + strings.Join(groupConditions, " OR ") + ")"
					conditions = append(conditions, groupClause)
				}
			}
		}

		// Process OR conditions
		for _, orCond := range b.orConditions {
			finalOr := ""
			parts := strings.Split(orCond.condition, "?")
			for i, part := range parts {
				finalOr += part
				if i < len(parts)-1 {
					finalOr += fmt.Sprintf("$%d", argIndex)
					argIndex++
				}
			}
			conditions = append(conditions, finalOr)
		}

		// Process raw conditions
		for _, rawCond := range b.rawConditions {
			finalRaw := ""
			parts := strings.Split(rawCond.sql, "?")
			for i, part := range parts {
				finalRaw += part
				if i < len(parts)-1 {
					finalRaw += fmt.Sprintf("$%d", argIndex)
					argIndex++
				}
			}
			conditions = append(conditions, finalRaw)
		}

		// Combine all conditions with OR
		sb.WriteString(strings.Join(conditions, " OR "))
	}

	if len(b.orderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(b.orderBy, ", "))
	}

	if b.limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
	}

	if b.offset > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", b.offset))
	}

	// Append args from OR conditions
	for _, orCond := range b.orConditions {
		b.args = append(b.args, orCond.args...)
	}

	// Append args from raw conditions
	for _, rawCond := range b.rawConditions {
		b.args = append(b.args, rawCond.args...)
	}

	return sb.String(), b.args
}
