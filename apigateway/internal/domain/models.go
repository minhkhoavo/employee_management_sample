package domain

import "time"

// ==================== PRODUCT MANAGEMENT ====================

// Product represents the product table in SQL DB
type Product struct {
	ID       int64  `json:"id" db:"id"`
	Brand    string `json:"brand" db:"brand"`
	Revision int64  `json:"revision" db:"revision"`
}

// Feature represents the feature table in SQL DB
type Feature struct {
	ID        int64  `json:"id" db:"id"`
	Brand     string `json:"brand" db:"brand"`
	Country   string `json:"country" db:"country"`
	Content   string `json:"content" db:"content"`
	SubNumber int    `json:"sub_number" db:"sub_number"`
}

// ProductInfo represents the product info document in GCP Datastore ONLY
type ProductInfo struct {
	ID        int64  `datastore:"ID" json:"id"`
	Brand     string `datastore:"Brand" json:"brand"`
	Country   string `datastore:"Country" json:"country"`
	Place     string `datastore:"Place" json:"place"`
	Year      int    `datastore:"Year" json:"year"`
	SubNumber int    `datastore:"SubNumber" json:"sub_number"`
}

// ProductDetailResponse response chính
type ProductDetailResponse struct {
	Item    ProductItemDTO     `json:"item"`
	Details []ProductDetailDTO `json:"details"`
}

// ProductItemDTO chứa info cơ bản
type ProductItemDTO struct {
	ID    int64  `json:"id"`
	Brand string `json:"brand"`
}

// ProductDetailDTO chứa merged info
type ProductDetailDTO struct {
	ID        int64  `json:"id"`
	Brand     string `json:"brand"`
	Country   string `json:"country"`
	Place     string `json:"place"`
	Year      int    `json:"year"`
	SubNumber int    `json:"sub_number"`
	Content   string `json:"content"`
}

// ==================== LEGACY MODELS ====================

// Employee represents the employees table
type Employee struct {
	ID        int       `json:"id" db:"id"`
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
	EmployeeID int       `json:"employee_id" db:"employee_id"`
	Salary     int       `json:"salary" db:"salary"`
	FromDate   time.Time `json:"from_date" db:"from_date"`
	ToDate     time.Time `json:"to_date" db:"to_date"`
}

// Title represents the titles table
type Title struct {
	EmpNo    int       `json:"emp_no" db:"emp_no"`
	Title    string    `json:"title" db:"title"`
	FromDate time.Time `json:"from_date" db:"from_date"`
	ToDate   time.Time `json:"to_date" db:"to_date"`
}

// EmployeeReport represents the aggregated employee data for reporting
type EmployeeReport struct {
	Employee          Employee      `json:"employee"`
	CurrentSalary     Salary        `json:"current_salary"`
	CurrentTitle      Title         `json:"current_title"`
	DepartmentHistory []DeptEmp     `json:"department_history"`
	ManagementHistory []DeptManager `json:"management_history"`
}
