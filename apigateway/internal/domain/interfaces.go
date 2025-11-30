package domain

import "context"

// EmployeeFilter defines criteria for listing employees
type EmployeeFilter struct {
	Limit  int
	Offset int
}

// EmployeeRepository defines the interface for employee data access
type EmployeeRepository interface {
	Create(ctx context.Context, e *Employee) error
	GetByID(ctx context.Context, id int) (*Employee, error)
	Update(ctx context.Context, e *Employee) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, filter EmployeeFilter) ([]Employee, error)

	// Advanced Queries
	GetCurrentSalary(ctx context.Context, empID int) (*Salary, error)
	GetDepartmentHistory(ctx context.Context, empID int) ([]DeptEmp, error)
	GetManagers(ctx context.Context, deptNo string) ([]DeptManager, error)
}
