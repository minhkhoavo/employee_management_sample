package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
)

// EmployeeDoc mirrors your domain.Employee for ES storage.
type EmployeeDoc struct {
	EmpNo     int       `json:"emp_no"`
	BirthDate time.Time `json:"birth_date"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Gender    string    `json:"gender"` // "M" or "F"
	HireDate  time.Time `json:"hire_date"`
}

// ElasticSearchClient wraps olivere/elastic client.
type ElasticSearchClient struct {
	client *elastic.Client
}

// NewElasticSearchClient creates a new client for Elasticsearch 7.x.
func NewElasticSearchClient() (*ElasticSearchClient, error) {
	// Connect to http://localhost:9200 by default
	client, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false), // Essential when using Docker or cloud
		// Add elastic.SetBasicAuth("user", "pass") if security enabled
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	return &ElasticSearchClient{client: client}, nil
}

// IndexEmployee indexes an employee document using emp_no as ID.
func (es *ElasticSearchClient) IndexEmployee(ctx context.Context, emp EmployeeDoc) error {
	_, err := es.client.Index().
		Index("employees").
		Id(fmt.Sprintf("%d", emp.EmpNo)).
		BodyJson(emp).
		Refresh("true"). // Make changes immediately searchable
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to index employee %d: %w", emp.EmpNo, err)
	}
	return nil
}

// GetEmployee retrieves an employee by emp_no.
func (es *ElasticSearchClient) GetEmployee(ctx context.Context, empNo int) (*EmployeeDoc, error) {
	result, err := es.client.Get().
		Index("employees").
		Id(fmt.Sprintf("%d", empNo)).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get employee %d: %w", empNo, err)
	}

	if !result.Found {
		return nil, fmt.Errorf("employee %d not found", empNo)
	}

	var emp EmployeeDoc
	if err := json.Unmarshal(result.Source, &emp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal employee %d: %w", empNo, err)
	}

	return &emp, nil
}

// SearchEmployeesByName performs a full-text match on first_name or last_name.
func (es *ElasticSearchClient) SearchEmployeesByName(ctx context.Context, name string) ([]EmployeeDoc, error) {
	query := elastic.NewMultiMatchQuery(name, "first_name", "last_name")

	searchResult, err := es.client.Search().
		Index("employees").
		Query(query).
		Size(100). // Adjust as needed
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var employees []EmployeeDoc
	for _, item := range searchResult.Hits.Hits {
		var emp EmployeeDoc
		if err := json.Unmarshal(item.Source, &emp); err != nil {
			continue // or return error
		}
		employees = append(employees, emp)
	}

	return employees, nil
}

// BulkIndexEmployees efficiently indexes multiple employees.
func (es *ElasticSearchClient) BulkIndexEmployees(ctx context.Context, employees []EmployeeDoc) error {
	bulkRequest := es.client.Bulk()

	for _, emp := range employees {
		req := elastic.NewBulkIndexRequest().
			Index("employees").
			Id(fmt.Sprintf("%d", emp.EmpNo)).
			Doc(emp)
		bulkRequest = bulkRequest.Add(req)
	}

	if bulkRequest.NumberOfActions() == 0 {
		return nil
	}

	bulkResponse, err := bulkRequest.Refresh("true").Do(ctx)
	if err != nil {
		return fmt.Errorf("bulk index failed: %w", err)
	}

	if bulkResponse.Errors {
		// Log first error for simplicity
		for _, item := range bulkResponse.Items {
			for _, op := range item {
				if op.Error != nil {
					return fmt.Errorf("bulk item failed: %s", op.Error.Reason)
				}
			}
		}
	}

	return nil
}
func (es *ElasticSearchClient) ScrollAllEmployees(ctx context.Context) ([]EmployeeDoc, error) {
	var allEmployees []EmployeeDoc

	// Step 1: Initialize scroll
	scroll := es.client.Scroll("employees").
		Size(1000).      // Fetch 1000 docs per batch
		KeepAlive("2m"). // Scroll context lives for 2 minutes
		Sort("_doc")     // Most efficient sort for scrolling

	for {
		// Step 2: Fetch next batch
		results, err := scroll.Do(ctx)
		if err == io.EOF {
			// No more documents
			break
		}
		if err != nil {
			return nil, fmt.Errorf("scroll error: %w", err)
		}

		// Step 3: Process batch
		for _, hit := range results.Hits.Hits {
			var emp EmployeeDoc
			if err := json.Unmarshal(hit.Source, &emp); err != nil {
				continue // or log and skip
			}
			allEmployees = append(allEmployees, emp)
		}

		// Optional: Add progress logging
		// fmt.Printf("Fetched %d employees so far\n", len(allEmployees))
	}

	// Step 4: Clear scroll (optional; ES auto-cleans after KeepAlive)
	// But good practice in long-running apps
	// es.client.ClearScroll(scroll.ScrollId).Do(ctx)

	return allEmployees, nil
}

type ScrollSession struct {
	ScrollID string
	Index    string
}

func (es *ElasticSearchClient) StartScroll(ctx context.Context, index string, size int) (*ScrollSession, []EmployeeDoc, error) {
	scroll := es.client.Scroll(index).Size(size).KeepAlive("5m")
	res, err := scroll.Do(ctx)
	if err != nil {
		return nil, nil, err
	}

	var docs []EmployeeDoc
	for _, hit := range res.Hits.Hits {
		var emp EmployeeDoc
		json.Unmarshal(hit.Source, &emp)
		docs = append(docs, emp)
	}

	return &ScrollSession{ScrollID: res.ScrollId, Index: index}, docs, nil
}

func (es *ElasticSearchClient) ContinueScroll(ctx context.Context, session *ScrollSession) ([]EmployeeDoc, error) {
	res, err := es.client.Scroll(session.Index).
		ScrollId(session.ScrollID).
		KeepAlive("5m").
		Do(ctx)
	if err != nil {
		return nil, err
	}

	var docs []EmployeeDoc
	for _, hit := range res.Hits.Hits {
		var emp EmployeeDoc
		json.Unmarshal(hit.Source, &emp)
		docs = append(docs, emp)
	}

	// Update scroll ID (it can change!)
	session.ScrollID = res.ScrollId
	return docs, nil
}

// Usage in a paginated API
// session, batch1, _ := client.StartScroll(ctx, "employees", 100)
// batch2, _ := client.ContinueScroll(ctx, session)
