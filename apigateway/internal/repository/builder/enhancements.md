# Query Builder Enhancements – Sample Usage

This document demonstrates how the **SQLBuilder** could be extended with the following features:

1. **Logical OR operator** – `Or(condition, args…)`
2. **Parentheses / grouping** – `WhereGroup(func(*SQLBuilder) *SQLBuilder)`
3. **Raw condition support** – `WhereRaw(sql string, args …interface{})`
4. **Safety checks** – validate that the number of `?` placeholders matches the length of `args`.

---

## 1. Logical OR Operator (`Or`)
```go
qb := builder.NewSQLBuilder().
    Select("emp_no", "first_name", "last_name").
    From("employees").
    // Employees either in department 1 **OR** department 2
    Or("dept_no = ?", "d001").
    Or("dept_no = ?", "d002")

sql, args := qb.Build()
// Expected SQL:
// SELECT emp_no, first_name, last_name FROM employees WHERE dept_no = $1 OR dept_no = $2
// args => []interface{}{"d001", "d002"}
```

The `Or` method stores its conditions in a separate slice and joins them with `OR` when building the final query.

---

## 2. Parentheses / Grouping (`WhereGroup`)
```go
qb := builder.NewSQLBuilder().
    Select("e.emp_no", "e.first_name", "d.dept_name").
    From("employees e").
    Join("INNER", "dept_emp de", "e.emp_no = de.emp_no").
    Join("INNER", "departments d", "de.dept_no = d.dept_no").
    // Grouped conditions: (gender = ? AND hire_date > ?) OR (salary > ?)
    WhereGroup(func(g *builder.SQLBuilder) *builder.SQLBuilder {
        return g.
            Where("gender = ?", "M").
            Where("hire_date > ?", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
    }).
    Or("salary > ?", 80000)

sql, args := qb.Build()
/* Expected SQL (formatted for readability):
SELECT e.emp_no, e.first_name, d.dept_name FROM employees e
INNER JOIN dept_emp de ON e.emp_no = de.emp_no
INNER JOIN departments d ON de.dept_no = d.dept_no
WHERE (gender = $1 AND hire_date > $2) OR salary > $3
*/
// args => []interface{}{ "M", time.Time{...}, 80000 }
```

`WhereGroup` creates a sub‑builder, builds its clause, and wraps it in parentheses before appending it to the main query.

---

## 3. Raw Condition Support (`WhereRaw`)
```go
qb := builder.NewSQLBuilder().
    Select("*`).
    From("employees").
    // Use a complex expression that the builder does not parse automatically
    WhereRaw("(salary BETWEEN ? AND ?) OR (title = ?)", 50000, 100000, "Senior Engineer")

sql, args := qb.Build()
// Expected SQL:
// SELECT * FROM employees WHERE (salary BETWEEN $1 AND $2) OR (title = $3)
// args => []interface{}{50000, 100000, "Senior Engineer"}
```

`WhereRaw` passes the condition verbatim to the builder while still handling placeholder conversion and argument ordering.

---

## 4. Safety Checks – Placeholder‑Argument Validation
```go
qb := builder.NewSQLBuilder().
    Select("*`).
    From("employees").
    // Intentional mismatch – two placeholders but only one argument
    Where("emp_no = ? AND gender = ?", 1001)

sql, args, err := qb.BuildSafe() // hypothetical method that returns an error on mismatch
if err != nil {
    // err => "placeholder count (2) does not match argument count (1)"
    log.Fatalf("invalid query: %v", err)
}
```

`BuildSafe` (or an internal validation step) would count the `?` tokens in the assembled `WHERE` clause and compare them to the length of `b.args`. If they differ, it returns an error before any SQL is sent to the database, preventing runtime failures.

---

### How These Enhancements Fit Together
* **Or** and **WhereGroup** give you expressive logical composition without manually concatenating strings.
* **WhereRaw** lets you fall back to raw SQL for edge‑cases while still benefitting from automatic placeholder handling.
* **Safety checks** protect you from mismatched placeholders, a common source of bugs when building queries dynamically.

Feel free to copy these snippets into your codebase and adapt the builder implementation accordingly.
