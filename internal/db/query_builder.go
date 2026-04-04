package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// QueryBuilder helps construct safe SQL queries
type QueryBuilder struct {
	db     *sql.DB
	table  string
	fields []string
	where  []string
	args   []interface{}
	limit  int
	offset int
	order  []string
}

// NewQueryBuilder creates a new query builder for a specific table
func NewQueryBuilder(db *sql.DB, tableName string) *QueryBuilder {
	return &QueryBuilder{
		db:     db,
		table:  tableName,
		fields: []string{"*"},
	}
}

// Select sets the fields to select (defaults to *)
func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	if len(fields) > 0 {
		qb.fields = fields
	}
	return qb
}

// Where adds a WHERE condition
// Example: Where("age > ?", 18) or Where("status = ? AND role = ?", "active", "admin")
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.where = append(qb.where, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// Limit sets the maximum number of rows to return
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the offset for pagination
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// OrderBy adds an ORDER BY clause
// Example: OrderBy("created_at DESC") or OrderBy("id ASC", "name ASC")
func (qb *QueryBuilder) OrderBy(orders ...string) *QueryBuilder {
	qb.order = append(qb.order, orders...)
	return qb
}

// Build constructs the SQL query and returns the args
func (qb *QueryBuilder) Build() (string, []interface{}) {
	var sb strings.Builder

	// SELECT clause
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(qb.fields, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(qb.table)

	// WHERE clause
	if len(qb.where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(qb.where, " AND "))
	}

	// ORDER BY clause
	if len(qb.order) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(qb.order, ", "))
	}

	// LIMIT clause
	if qb.limit > 0 {
		sb.WriteString(" LIMIT ")
		sb.WriteString(fmt.Sprintf("%d", qb.limit))
	}

	// OFFSET clause
	if qb.offset > 0 {
		sb.WriteString(" OFFSET ")
		sb.WriteString(fmt.Sprintf("%d", qb.offset))
	}

	return sb.String(), qb.args
}

// Query executes the SELECT query and returns rows
func (qb *QueryBuilder) Query() (*sql.Rows, error) {
	query, args := qb.Build()
	return qb.db.Query(query, args...)
}

// QueryRow executes the SELECT query and returns a single row
func (qb *QueryBuilder) QueryRow() *sql.Row {
	query, args := qb.Build()
	return qb.db.QueryRow(query, args...)
}

// Insert builds and executes an INSERT statement
func Insert(db *sql.DB, table string, columns []string, values []interface{}) (sql.Result, error) {
	if len(columns) != len(values) {
		return nil, fmt.Errorf("columns and values count mismatch")
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	sb.WriteString(table)
	sb.WriteString(" (")
	sb.WriteString(strings.Join(columns, ", "))
	sb.WriteString(") VALUES (")

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	sb.WriteString(strings.Join(placeholders, ", "))
	sb.WriteString(")")

	// Note: The placeholder style (?) is generic.
	// For PostgreSQL, you might need to adjust to $1, $2, etc.
	// This is a simplified builder. A production version would handle dialects.

	// For PostgreSQL compatibility in this example, we will assume the caller
	// handles dialect specifics or we use a helper to convert placeholders.
	// However, to keep it simple and standard library compliant for the prompt:
	// We will assume the driver handles '?' or we use a dialect-aware version below.

	// Let's implement a PostgreSQL specific version for the "default" DB context
	// since the prompt implies a primary Postgres setup.

	// Re-building for PG compatibility ($1, $2...)
	var pgSB strings.Builder
	pgSB.WriteString("INSERT INTO ")
	pgSB.WriteString(table)
	pgSB.WriteString(" (")
	pgSB.WriteString(strings.Join(columns, ", "))
	pgSB.WriteString(") VALUES (")

	pgPlaceholders := make([]string, len(values))
	for i := range values {
		pgPlaceholders[i] = fmt.Sprintf("$%d", i+1)
	}
	pgSB.WriteString(strings.Join(pgPlaceholders, ", "))
	pgSB.WriteString(")")

	return db.Exec(pgSB.String(), values...)
}

// Update builds and executes an UPDATE statement
func Update(db *sql.DB, table string, setColumns []string, setValues []interface{}, whereClause string, whereArgs ...interface{}) (sql.Result, error) {
	if len(setColumns) != len(setValues) {
		return nil, fmt.Errorf("set columns and values count mismatch")
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(table)
	sb.WriteString(" SET ")

	setParts := make([]string, len(setColumns))
	for i, col := range setColumns {
		setParts[i] = fmt.Sprintf("%s = $%d", col, i+1)
	}
	sb.WriteString(strings.Join(setParts, ", "))

	if whereClause != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereClause)
	}

	// Combine args: set values first, then where args
	allArgs := append(setValues, whereArgs...)

	return db.Exec(sb.String(), allArgs...)
}

// Delete builds and executes a DELETE statement
func Delete(db *sql.DB, table string, whereClause string, args ...interface{}) (sql.Result, error) {
	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(table)

	if whereClause != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereClause)
	}

	return db.Exec(sb.String(), args...)
}

// Count returns the number of rows matching the query
func (qb *QueryBuilder) Count() (int64, error) {
	var count int64
	query, args := qb.Build()
	// Replace SELECT fields with COUNT(*)
	parts := strings.SplitN(query, "FROM", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid query structure")
	}
	countQuery := "SELECT COUNT(*) FROM" + parts[1]

	err := qb.db.QueryRow(countQuery, args...).Scan(&count)
	return count, err
}
