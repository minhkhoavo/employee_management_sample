package builder

import (
	"strings"
	"testing"
	"time"
)

func TestSQLBuilder(t *testing.T) {
	t.Run("Select", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("id", "name").From("users").Where("id = ?", 1).Build()
		expected := "SELECT id, name FROM users WHERE id = $1"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("expected args [1], got %v", args)
		}
	})

	t.Run("Insert", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Insert("users", "name", "age").Values("Alice", 30).Build()
		expected := "INSERT INTO users (name, age) VALUES ($1, $2)"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 2 || args[0] != "Alice" || args[1] != 30 {
			t.Errorf("expected args [Alice 30], got %v", args)
		}
	})

	t.Run("Update", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Update("users").Set("name", "Bob").Where("id = ?", 1).Build()
		expected := "UPDATE users SET name = $1 WHERE id = $2"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 2 || args[0] != "Bob" || args[1] != 1 {
			t.Errorf("expected args [Bob 1], got %v", args)
		}
	})
}

// Test new enhancement features
func TestSQLBuilderEnhancements(t *testing.T) {
	t.Run("Or Operator", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("emp_no", "first_name", "last_name").
			From("employees").
			Or("dept_no = ?", "d001").
			Or("dept_no = ?", "d002").
			Build()

		expected := "SELECT emp_no, first_name, last_name FROM employees WHERE dept_no = $1 OR dept_no = $2"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 2 || args[0] != "d001" || args[1] != "d002" {
			t.Errorf("expected args [d001 d002], got %v", args)
		}
	})

	t.Run("WhereGroup with And conditions", func(t *testing.T) {
		testTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		b := NewSQLBuilder()
		query, args := b.Select("e.emp_no", "e.first_name", "d.dept_name").
			From("employees e").
			Join("INNER", "dept_emp de", "e.emp_no = de.emp_no").
			Join("INNER", "departments d", "de.dept_no = d.dept_no").
			WhereGroup(func(g *SQLBuilder) *SQLBuilder {
				return g.
					Where("gender = ?", "M").
					Where("hire_date > ?", testTime)
			}).
			Or("salary > ?", 80000).
			Build()

		// Check that query contains the grouped condition
		if !strings.Contains(query, "(gender = $1 AND hire_date > $2)") {
			t.Errorf("expected grouped condition, got %s", query)
		}
		if !strings.Contains(query, "OR salary > $3") {
			t.Errorf("expected OR condition, got %s", query)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})

	t.Run("WhereRaw with complex expression", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("*").
			From("employees").
			WhereRaw("(salary BETWEEN ? AND ?) OR (title = ?)", 50000, 100000, "Senior Engineer").
			Build()

		expected := "SELECT * FROM employees WHERE (salary BETWEEN $1 AND $2) OR (title = $3)"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 3 || args[0] != 50000 || args[1] != 100000 || args[2] != "Senior Engineer" {
			t.Errorf("expected args [50000 100000 Senior Engineer], got %v", args)
		}
	})

	t.Run("BuildSafe with valid query", func(t *testing.T) {
		b := NewSQLBuilder()
		sql, args, err := b.Select("*").
			From("employees").
			Where("emp_no = ?", 1001).
			Where("gender = ?", "M").
			BuildSafe()

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(args) != 2 {
			t.Errorf("expected 2 args, got %d", len(args))
		}
		if !strings.Contains(sql, "$1") || !strings.Contains(sql, "$2") {
			t.Errorf("expected placeholders $1 and $2 in %s", sql)
		}
	})

	t.Run("Combined Where and Or conditions", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("*").
			From("employees").
			Where("status = ?", "active").
			Or("dept_no = ?", "d001").
			Or("dept_no = ?", "d002").
			Build()

		// Should have WHERE clause with AND condition followed by OR conditions
		if !strings.Contains(query, "WHERE") {
			t.Errorf("expected WHERE clause in %s", query)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})

	t.Run("Multiple WhereRaw conditions", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("*").
			From("employees").
			WhereRaw("salary > ?", 50000).
			WhereRaw("hire_date < ?", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).
			Build()

		if !strings.Contains(query, "WHERE") {
			t.Errorf("expected WHERE clause in %s", query)
		}
		if len(args) != 2 {
			t.Errorf("expected 2 args, got %d: %v", len(args), args)
		}
	})

	t.Run("Complex nested groups", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Select("*").
			From("employees").
			Where("status = ?", "active").
			WhereGroup(func(g *SQLBuilder) *SQLBuilder {
				return g.
					Where("dept_no = ?", "d001").
					Or("dept_no = ?", "d002")
			}).
			Build()

		// Verify the query structure
		if !strings.Contains(query, "WHERE") {
			t.Errorf("expected WHERE clause in %s", query)
		}
		// Should have status condition OR grouped conditions
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})

	t.Run("Update with Or conditions", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Update("employees").
			Set("status", "inactive").
			Or("dept_no = ?", "d001").
			Or("dept_no = ?", "d002").
			Build()

		expected := "UPDATE employees SET status = $1 WHERE dept_no = $2 OR dept_no = $3"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d: %v", len(args), args)
		}
	})

	t.Run("Delete with WhereRaw", func(t *testing.T) {
		b := NewSQLBuilder()
		query, args := b.Delete("employees").
			WhereRaw("created_at < NOW() - INTERVAL '1 year'").
			Build()

		expected := "DELETE FROM employees WHERE created_at < NOW() - INTERVAL '1 year'"
		if query != expected {
			t.Errorf("expected %s, got %s", expected, query)
		}
		if len(args) != 0 {
			t.Errorf("expected 0 args, got %d: %v", len(args), args)
		}
	})
}
