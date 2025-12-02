package repository

import (
	"context"
	"database/sql"

	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
)

var (
	employeeTable    = "employees.employee"
	salaryTable      = "employees.salary"
	deptEmpTable     = "employees.dept_emp"
	deptManagerTable = "employees.dept_manager"
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
	query, args := b.Insert(employeeTable, "id", "birth_date", "first_name", "last_name", "gender", "hire_date").
		Values(e.ID, e.BirthDate, e.FirstName, e.LastName, e.Gender, e.HireDate).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) Upsert(ctx context.Context, e *domain.Employee) error {
	b := builder.NewSQLBuilder()
	query, args := b.Insert(employeeTable, "id", "birth_date", "first_name", "last_name", "gender", "hire_date").
		Values(e.ID, e.BirthDate, e.FirstName, e.LastName, e.Gender, e.HireDate).
		OnConflict("(id) DO UPDATE SET birth_date = EXCLUDED.birth_date, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, gender = EXCLUDED.gender, hire_date = EXCLUDED.hire_date").
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) GetByID(ctx context.Context, id int) (*domain.Employee, error) {
	b := builder.NewSQLBuilder()
	query, args := b.Select("id", "birth_date", "first_name", "last_name", "gender", "hire_date").
		From(employeeTable).
		Where("id = ?", id).
		Build()

	row := r.db.QueryRowContext(ctx, query, args...)
	var e domain.Employee
	if err := row.Scan(&e.ID, &e.BirthDate, &e.FirstName, &e.LastName, &e.Gender, &e.HireDate); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *employeeRepository) Update(ctx context.Context, e *domain.Employee) error {
	b := builder.NewSQLBuilder()
	query, args := b.Update(employeeTable).
		Set("first_name", e.FirstName).
		Set("last_name", e.LastName).
		Set("gender", e.Gender).
		Where("id = ?", e.ID).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) Delete(ctx context.Context, id int) error {
	b := builder.NewSQLBuilder()
	query, args := b.Delete(employeeTable).
		Where("id = ?", id).
		Build()

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *employeeRepository) List(ctx context.Context, filter domain.EmployeeFilter) ([]domain.Employee, error) {
	b := builder.NewSQLBuilder()
	b.Select("id", "birth_date", "first_name", "last_name", "gender", "hire_date").
		From(employeeTable).
		OrderBy("id ASC")

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
		if err := rows.Scan(&e.ID, &e.BirthDate, &e.FirstName, &e.LastName, &e.Gender, &e.HireDate); err != nil {
			return nil, err
		}
		employees = append(employees, e)
	}
	return employees, nil
}

func (r *employeeRepository) GetCurrentSalary(ctx context.Context, empID int) (*domain.Salary, error) {
	// Business logic: Current salary has to_date = '9999-01-01'
	b := builder.NewSQLBuilder()
	query, args := b.Select("id", "salary", "from_date", "to_date").
		From(salaryTable).
		Where("employee_id = ? AND to_date = ?", empID, "9999-01-01").
		Build()

	row := r.db.QueryRowContext(ctx, query, args...)
	var s domain.Salary
	if err := row.Scan(&s.EmployeeID, &s.Salary, &s.FromDate, &s.ToDate); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *employeeRepository) GetDepartmentHistory(ctx context.Context, empID int) ([]domain.DeptEmp, error) {
	b := builder.NewSQLBuilder()
	query, args := b.Select("emp_no", "dept_no", "from_date", "to_date").
		From(deptEmpTable).
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
		From(deptManagerTable).
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
