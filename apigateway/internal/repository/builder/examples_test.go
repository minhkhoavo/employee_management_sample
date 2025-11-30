package builder_test

import (
	"fmt"
	"log"
	"time"

	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
)

// Example1_OrOperator demonstrates using the Or() method for OR conditions
func Example_orOperator() {
	qb := builder.NewSQLBuilder().
		Select("emp_no", "first_name", "last_name").
		From("employees").
		Or("dept_no = ?", "d001").
		Or("dept_no = ?", "d002")

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: SELECT emp_no, first_name, last_name FROM employees WHERE dept_no = $1 OR dept_no = $2
	// Args: [d001 d002]
}

// Example2_WhereGroup demonstrates using WhereGroup() for parenthesized conditions
func Example_whereGroup() {
	testTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	qb := builder.NewSQLBuilder().
		Select("e.emp_no", "e.first_name", "d.dept_name").
		From("employees e").
		Join("INNER", "dept_emp de", "e.emp_no = de.emp_no").
		Join("INNER", "departments d", "de.dept_no = d.dept_no").
		WhereGroup(func(g *builder.SQLBuilder) *builder.SQLBuilder {
			return g.
				Where("gender = ?", "M").
				Where("hire_date > ?", testTime)
		}).
		Or("salary > ?", 80000)

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Number of args: %d\n", len(args))

	// Output:
	// SQL: SELECT e.emp_no, e.first_name, d.dept_name FROM employees e INNER JOIN dept_emp de ON e.emp_no = de.emp_no INNER JOIN departments d ON de.dept_no = d.dept_no WHERE (gender = $1 AND hire_date > $2) OR salary > $3
	// Number of args: 3
}

// Example3_WhereRaw demonstrates using WhereRaw() for complex SQL expressions
func Example_whereRaw() {
	qb := builder.NewSQLBuilder().
		Select("*").
		From("employees").
		WhereRaw("(salary BETWEEN ? AND ?) OR (title = ?)", 50000, 100000, "Senior Engineer")

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: SELECT * FROM employees WHERE (salary BETWEEN $1 AND $2) OR (title = $3)
	// Args: [50000 100000 Senior Engineer]
}

// Example4_BuildSafe demonstrates using BuildSafe() for validation
func Example_buildSafe() {
	// Valid query
	qb1 := builder.NewSQLBuilder().
		Select("*").
		From("employees").
		Where("emp_no = ?", 1001).
		Where("gender = ?", "M")

	sql, args, err := qb1.BuildSafe()
	if err != nil {
		log.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Valid query built successfully")
		fmt.Printf("Number of placeholders matches args: %d\n", len(args))
	}

	fmt.Println("SQL:", sql)

	// Output:
	// Valid query built successfully
	// Number of placeholders matches args: 2
	// SQL: SELECT * FROM employees WHERE emp_no = $1 AND gender = $2
}

// Example5_CombinedConditions demonstrates combining multiple condition types
func Example_combinedConditions() {
	qb := builder.NewSQLBuilder().
		Select("*").
		From("employees").
		Where("status = ?", "active").
		WhereGroup(func(g *builder.SQLBuilder) *builder.SQLBuilder {
			return g.
				Where("dept_no = ?", "d001").
				Or("dept_no = ?", "d002")
		}).
		Or("salary > ?", 100000).
		WhereRaw("performance_score >= ?", 4.5).
		OrderBy("emp_no DESC").
		Limit(10)

	sql, args := qb.Build()
	fmt.Println("Complex query built successfully")
	fmt.Printf("Number of conditions: %d\n", len(args))
	fmt.Println("SQL:", sql)

	// Output:
	// Complex query built successfully
	// Number of conditions: 5
	// SQL: SELECT * FROM employees WHERE status = $1 OR (dept_no = $2 OR dept_no = $3) OR salary > $4 OR performance_score >= $5 ORDER BY emp_no DESC LIMIT 10
}

// Example6_UpdateWithOr demonstrates using Or() with UPDATE queries
func Example_updateWithOr() {
	qb := builder.NewSQLBuilder().
		Update("employees").
		Set("status", "inactive").
		Or("dept_no = ?", "d001").
		Or("dept_no = ?", "d002")

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: UPDATE employees SET status = $1 WHERE dept_no = $2 OR dept_no = $3
	// Args: [inactive d001 d002]
}

// Example7_DeleteWithWhereRaw demonstrates using WhereRaw() with DELETE queries
func Example_deleteWithWhereRaw() {
	qb := builder.NewSQLBuilder().
		Delete("employees").
		WhereRaw("created_at < NOW() - INTERVAL '1 year'")

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Number of args: %d\n", len(args))

	// Output:
	// SQL: DELETE FROM employees WHERE created_at < NOW() - INTERVAL '1 year'
	// Number of args: 0
}

// Example8_Upsert demonstrates using OnConflict() for upsert operations
func Example_upsert() {
	// Upsert with composite primary key
	qb := builder.NewSQLBuilder().
		Insert("dept_emp", "emp_no", "dept_no", "from_date", "to_date").
		Values(10001, "d005", "2023-01-01", "9999-01-01").
		OnConflict("(emp_no, dept_no) DO UPDATE SET from_date = EXCLUDED.from_date, to_date = EXCLUDED.to_date")

	sql, args := qb.Build()
	fmt.Println("SQL:", sql)
	fmt.Printf("Args: %v\n", args)

	// Output:
	// SQL: INSERT INTO dept_emp (emp_no, dept_no, from_date, to_date) VALUES ($1, $2, $3, $4) ON CONFLICT (emp_no, dept_no) DO UPDATE SET from_date = EXCLUDED.from_date, to_date = EXCLUDED.to_date
	// Args: [10001 d005 2023-01-01 9999-01-01]
}
