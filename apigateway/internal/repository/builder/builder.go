package builder

import (
	"fmt"
	"strings"
)

// SQLBuilder helps construct SQL queries dynamically.
type SQLBuilder struct {
	table       string
	columns     []string
	batchValues [][]interface{}
	where       []string
	whereArgs   []interface{}
	updateArgs  []interface{}
	joins       []string
	orderBy     []string
	groupBy     []string
	limit       int
	offset      int
	updateCols  []string
	isInsert    bool
	isUpdate    bool
	isDelete    bool
	isSelect    bool
	// New fields for enhancements
	orConditions  []orCondition
	whereGroups   []whereGroup
	rawConditions []rawCondition
	// New field for Upsert
	onConflict     string
	onConflictArgs []interface{}
	useUnionSelect bool
	unions         []unionClause
	insertSelect   *SQLBuilder
	// Batch Upsert fields
	isBatchUpsert  bool
}

type unionClause struct {
	builder *SQLBuilder
	all     bool
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

// InsertSelectUnion specifies that the insert should use UNION ALL SELECT instead of VALUES.
func (b *SQLBuilder) InsertSelectUnion() *SQLBuilder {
	b.useUnionSelect = true
	return b
}

// SelectQuery specifies a subquery for INSERT INTO ... SELECT.
func (b *SQLBuilder) SelectQuery(builder *SQLBuilder) *SQLBuilder {
	b.insertSelect = builder
	return b
}

// Union adds a UNION clause.
func (b *SQLBuilder) Union(other *SQLBuilder) *SQLBuilder {
	b.unions = append(b.unions, unionClause{builder: other, all: false})
	return b
}

// UnionAll adds a UNION ALL clause.
func (b *SQLBuilder) UnionAll(other *SQLBuilder) *SQLBuilder {
	b.unions = append(b.unions, unionClause{builder: other, all: true})
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
	b.updateArgs = append(b.updateArgs, val)
	return b
}

// Values specifies the values for insertion. Can be called multiple times for batch insertion.
func (b *SQLBuilder) Values(vals ...interface{}) *SQLBuilder {
	b.batchValues = append(b.batchValues, vals)
	return b
}

// Where adds a condition to the query.
func (b *SQLBuilder) Where(condition string, args ...interface{}) *SQLBuilder {
	b.where = append(b.where, condition)
	b.whereArgs = append(b.whereArgs, args...)
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

// GroupBy adds a GROUP BY clause.
func (b *SQLBuilder) GroupBy(cols ...string) *SQLBuilder {
	b.groupBy = append(b.groupBy, cols...)
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

// OnConflict adds an ON CONFLICT clause to the query.
func (b *SQLBuilder) OnConflict(clause string, args ...interface{}) *SQLBuilder {
	b.onConflict = clause
	b.onConflictArgs = args
	return b
}

// ============================================================================
// Batch Upsert Methods
// ============================================================================

// BatchUpsert initiates batch upsert mode for bulk INSERT ... ON CONFLICT ... DO UPDATE.
func (b *SQLBuilder) BatchUpsert(table string, cols ...string) *SQLBuilder {
	b.isBatchUpsert = true
	b.isInsert = true
	b.table = table
	b.columns = cols
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

// Build constructs the final SQL string and arguments, ready for DB prepared statements
// Build constructs the final SQL string and arguments, ready for DB prepared statements
func (b *SQLBuilder) Build() (string, []interface{}) {
	rawSQL, args := b.toRawSQL()

	var sb strings.Builder
	argIdx := 1
	parts := strings.Split(rawSQL, "?")
	for i, part := range parts {
		sb.WriteString(part)
		if i < len(parts)-1 {
			sb.WriteString(fmt.Sprintf("$%d", argIdx))
			argIdx++
		}
	}

	return sb.String(), args
}

// BuildInline constructs the final SQL string with all arguments inlined.
// This is useful for debugging or when a non-parameterized query is required.
func (b *SQLBuilder) BuildInline() string {
	rawSQL, args := b.toRawSQL()

	var sb strings.Builder
	argIdx := 0
	parts := strings.Split(rawSQL, "?")
	for i, part := range parts {
		sb.WriteString(part)
		if i < len(parts)-1 && argIdx < len(args) {
			sb.WriteString(formatSQLValue(args[argIdx]))
			argIdx++
		}
	}

	return sb.String()
}

func formatSQLValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		// Fallback for other types like time.Time or fmt.Stringer
		return fmt.Sprintf("'%v'", v)
	}
}

// toRawSQL constructs the SQL string with "?" placeholders and collects all arguments.
func (b *SQLBuilder) toRawSQL() (string, []interface{}) {
	var sb strings.Builder
	var allArgs []interface{}

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
		sb.WriteString(") ")

		if b.insertSelect != nil {
			subSQL, subArgs := b.insertSelect.toRawSQL()
			sb.WriteString(subSQL)
			allArgs = append(allArgs, subArgs...)
		} else if b.useUnionSelect {
			var selectClauses []string
			for i, row := range b.batchValues {
				rowPlaceholders := make([]string, len(row))
				for j := range row {
					rowPlaceholders[j] = "?"
				}
				prefix := "SELECT "
				if i > 0 {
					prefix = "UNION ALL SELECT "
				}
				selectClauses = append(selectClauses, prefix+strings.Join(rowPlaceholders, ", "))
				allArgs = append(allArgs, row...)
			}
			sb.WriteString(strings.Join(selectClauses, " "))
		} else {
			sb.WriteString("VALUES ")
			var allPlaceholders []string
			for _, row := range b.batchValues {
				rowPlaceholders := make([]string, len(row))
				for i := range row {
					rowPlaceholders[i] = "?"
				}
				allPlaceholders = append(allPlaceholders, "("+strings.Join(rowPlaceholders, ", ")+")")
				allArgs = append(allArgs, row...)
			}
			sb.WriteString(strings.Join(allPlaceholders, ", "))
		}

		if b.onConflict != "" {
			sb.WriteString(" ON CONFLICT ")
			sb.WriteString(b.onConflict)
			allArgs = append(allArgs, b.onConflictArgs...)
		}

		// Insert is a special case that often doesn't have WHERE/ORDER/LIMIT in standard SQL,
		// but if we need them, we would proceed. Most DBs don't support WHERE for INSERT ... VALUES.
		return sb.String(), allArgs
	} else if b.isUpdate {
		sb.WriteString("UPDATE ")
		sb.WriteString(b.table)
		sb.WriteString(" SET ")
		setClauses := make([]string, len(b.updateCols))
		for i, col := range b.updateCols {
			setClauses[i] = fmt.Sprintf("%s = ?", col)
		}
		sb.WriteString(strings.Join(setClauses, ", "))
		allArgs = append(allArgs, b.updateArgs...)
	} else if b.isDelete {
		sb.WriteString("DELETE FROM ")
		sb.WriteString(b.table)
	}

	// Build WHERE clause
	hasWhere := len(b.where) > 0 || len(b.orConditions) > 0 || len(b.whereGroups) > 0 || len(b.rawConditions) > 0
	if hasWhere {
		sb.WriteString(" WHERE ")
		var conditions []string

		if len(b.where) > 0 {
			conditions = append(conditions, strings.Join(b.where, " AND "))
			allArgs = append(allArgs, b.whereArgs...)
		}

		for _, group := range b.whereGroups {
			// Extract just the WHERE part from subquery or handle it specifically
			// For simplicity, let's assume whereGroups are built carefully
			condSQL, condArgs := group.builder.buildConditions()
			if condSQL != "" {
				conditions = append(conditions, "("+condSQL+")")
				allArgs = append(allArgs, condArgs...)
			}
		}

		for _, orCond := range b.orConditions {
			conditions = append(conditions, orCond.condition)
			allArgs = append(allArgs, orCond.args...)
		}

		for _, rawCond := range b.rawConditions {
			conditions = append(conditions, rawCond.sql)
			allArgs = append(allArgs, rawCond.args...)
		}

		// Join with OR logic (as per existing implementation's intent, though confusingly mixed)
		// Existing code joined everything with OR at the end.
		sb.WriteString(strings.Join(conditions, " OR "))
	}

	if len(b.groupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(b.groupBy, ", "))
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

	// Handle Unions
	for _, u := range b.unions {
		if u.all {
			sb.WriteString(" UNION ALL ")
		} else {
			sb.WriteString(" UNION ")
		}
		subSQL, subArgs := u.builder.toRawSQL()
		sb.WriteString(subSQL)
		allArgs = append(allArgs, subArgs...)
	}

	return sb.String(), allArgs
}

// buildConditions is a helper to build only the WHERE part
func (b *SQLBuilder) buildConditions() (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if len(b.where) > 0 {
		conditions = append(conditions, strings.Join(b.where, " AND "))
		args = append(args, b.whereArgs...)
	}

	for _, group := range b.whereGroups {
		subSQL, subArgs := group.builder.buildConditions()
		if subSQL != "" {
			conditions = append(conditions, "("+subSQL+")")
			args = append(args, subArgs...)
		}
	}

	for _, orCond := range b.orConditions {
		conditions = append(conditions, orCond.condition)
		args = append(args, orCond.args...)
	}

	for _, rawCond := range b.rawConditions {
		conditions = append(conditions, rawCond.sql)
		args = append(args, rawCond.args...)
	}

	if len(conditions) == 0 {
		return "", nil
	}

	return strings.Join(conditions, " OR "), args
}

// MultiStatementSQLBuilder allows executing multiple SQL queries in a single query string.
type MultiStatementSQLBuilder struct {
	builders []*SQLBuilder
}

// NewMultiStatementSQLBuilder creates a new instance of MultiStatementBuilder.
func NewMultiStatementSQLBuilder() *MultiStatementSQLBuilder {
	return &MultiStatementSQLBuilder{}
}

// Add appends one or more SQLBuilders to the multi-statement batch.
func (m *MultiStatementSQLBuilder) Add(builders ...*SQLBuilder) *MultiStatementSQLBuilder {
	m.builders = append(m.builders, builders...)
	return m
}

// Build constructs the final SQL string containing multiple statements and their arguments.
func (m *MultiStatementSQLBuilder) Build() (string, []interface{}) {
	var rawQueries []string
	var allArgs []interface{}

	for _, b := range m.builders {
		rawSQL, args := b.toRawSQL()
		rawQueries = append(rawQueries, rawSQL)
		allArgs = append(allArgs, args...)
	}

	combinedRawSQL := strings.Join(rawQueries, "; ")
	var sb strings.Builder
	argIdx := 1
	parts := strings.Split(combinedRawSQL, "?")
	for i, part := range parts {
		sb.WriteString(part)
		if i < len(parts)-1 {
			sb.WriteString(fmt.Sprintf("$%d", argIdx))
			argIdx++
		}
	}

	return sb.String(), allArgs
}

// BuildInline constructs the final SQL string containing multiple statements with all arguments inlined.
func (m *MultiStatementSQLBuilder) BuildInline() string {
	var inlineQueries []string

	for _, b := range m.builders {
		inlineQueries = append(inlineQueries, b.BuildInline())
	}

	return strings.Join(inlineQueries, "; ")
}
