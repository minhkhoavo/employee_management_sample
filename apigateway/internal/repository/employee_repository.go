package repository

import (
	"context"
	"database/sql"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
)

type employeeRepository struct {
	db *sql.DB
}

// NewEmployeeRepository creates a new instance of EmployeeRepository
func NewEmployeeRepository(db *sql.DB) domain.EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) Create(ctx context.Context, e *domain.Employee) error {
	b := builder.NewSQLBuilder()
	query, args := b.Insert("employees", "emp_no", "birth_date", "first_name", "last_name", "gender", "hire_date").
		Values(e.EmpNo, e.BirthDate, e.FirstName, e.LastName, e.Gender, e.HireDate).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) GetByID(ctx context.Context, id int) (*domain.Employee, error) {
	b := builder.NewSQLBuilder()
	query, args := b.Select("emp_no", "birth_date", "first_name", "last_name", "gender", "hire_date").
		From("employees").
		Where("emp_no = ?", id).
		Build()

	row := r.db.QueryRowContext(ctx, query, args...)
	var e domain.Employee
	if err := row.Scan(&e.EmpNo, &e.BirthDate, &e.FirstName, &e.LastName, &e.Gender, &e.HireDate); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *employeeRepository) Update(ctx context.Context, e *domain.Employee) error {
	b := builder.NewSQLBuilder()
	query, args := b.Update("employees").
		Set("first_name", e.FirstName).
		Set("last_name", e.LastName).
		Set("gender", e.Gender).
		Where("emp_no = ?", e.EmpNo).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) Delete(ctx context.Context, id int) error {
	b := builder.NewSQLBuilder()
	query, args := b.Delete("employees").
		Where("emp_no = ?", id).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) List(ctx context.Context, filter domain.EmployeeFilter) ([]domain.Employee, error) {
	b := builder.NewSQLBuilder()
	b.Select("emp_no", "birth_date", "first_name", "last_name", "gender", "hire_date").
		From("employees").
		OrderBy("emp_no ASC")

	if filter.Limit > 0 {
		b.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		b.Offset(filter.Offset)
	}

	query, args := b.Build()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []domain.Employee
	for rows.Next() {
		var e domain.Employee
		if err := rows.Scan(&e.EmpNo, &e.BirthDate, &e.FirstName, &e.LastName, &e.Gender, &e.HireDate); err != nil {
			return nil, err
		}
		employees = append(employees, e)
	}
	return employees, nil
}

func (r *employeeRepository) GetCurrentSalary(ctx context.Context, empID int) (*domain.Salary, error) {
	// Business logic: Current salary has to_date = '9999-01-01'
	b := builder.NewSQLBuilder()
	query, args := b.Select("emp_no", "salary", "from_date", "to_date").
		From("salaries").
		Where("emp_no = ? AND to_date = ?", empID, "9999-01-01").
		Build()

	row := r.db.QueryRowContext(ctx, query, args...)
	var s domain.Salary
	if err := row.Scan(&s.EmpNo, &s.Salary, &s.FromDate, &s.ToDate); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *employeeRepository) GetDepartmentHistory(ctx context.Context, empID int) ([]domain.DeptEmp, error) {
	b := builder.NewSQLBuilder()
	query, args := b.Select("emp_no", "dept_no", "from_date", "to_date").
		From("dept_emp").
		Where("emp_no = ?", empID).
		OrderBy("from_date DESC").
		Build()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.DeptEmp
	for rows.Next() {
		var de domain.DeptEmp
		if err := rows.Scan(&de.EmpNo, &de.DeptNo, &de.FromDate, &de.ToDate); err != nil {
			return nil, err
		}
		history = append(history, de)
	}
	return history, nil
}

func (r *employeeRepository) GetManagers(ctx context.Context, deptNo string) ([]domain.DeptManager, error) {
	b := builder.NewSQLBuilder()
	query, args := b.Select("dept_no", "emp_no", "from_date", "to_date").
		From("dept_manager").
		Where("dept_no = ?", deptNo).
		OrderBy("from_date DESC").
		Build()

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var managers []domain.DeptManager
	for rows.Next() {
		var dm domain.DeptManager
		if err := rows.Scan(&dm.DeptNo, &dm.EmpNo, &dm.FromDate, &dm.ToDate); err != nil {
			return nil, err
		}
		managers = append(managers, dm)
	}
	return managers, nil
}
