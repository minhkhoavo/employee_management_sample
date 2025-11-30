package domain

import "time"

// Employee represents the employees table
type Employee struct {
	EmpNo     int       `json:"emp_no" db:"emp_no"`
	BirthDate time.Time `json:"birth_date" db:"birth_date"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Gender    string    `json:"gender" db:"gender"`
	HireDate  time.Time `json:"hire_date" db:"hire_date"`
}

// Department represents the departments table
type Department struct {
	DeptNo   string `json:"dept_no" db:"dept_no"`
	DeptName string `json:"dept_name" db:"dept_name"`
}

// DeptEmp represents the dept_emp table (junction table)
type DeptEmp struct {
	EmpNo    int       `json:"emp_no" db:"emp_no"`
	DeptNo   string    `json:"dept_no" db:"dept_no"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}

// DeptManager represents the dept_manager table
type DeptManager struct {
	DeptNo   string    `json:"dept_no" db:"dept_no"`
	EmpNo    int       `json:"emp_no" db:"emp_no"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}

// Salary represents the salaries table
type Salary struct {
	EmpNo    int       `json:"emp_no" db:"emp_no"`
	Salary   int       `json:"salary" db:"salary"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}

// Title represents the titles table
type Title struct {
	EmpNo    int       `json:"emp_no" db:"emp_no"`
	Title    string    `json:"title" db:"title"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}
