package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
	"github.com/locvowork/employee_management_sample/apigateway/internal/domain"
)

// DatastoreClient wraps the cloud datastore client
type DatastoreClient struct {
	client *datastore.Client
}

// NewDatastoreClient creates a new wrapper
func NewDatastoreClient(client *datastore.Client) *DatastoreClient {
	return &DatastoreClient{client: client}
}

// WrapDatastoreClient wraps existing datastore client
func WrapDatastoreClient(client *datastore.Client) *DatastoreClient {
	if client == nil {
		return nil
	}
	return &DatastoreClient{client: client}
}

// BatchSaveProductInfos saves multiple ProductInfo documents
func (dc *DatastoreClient) BatchSaveProductInfos(ctx context.Context, productInfos []domain.ProductInfo) error {
	if dc == nil || dc.client == nil {
		return fmt.Errorf("datastore client is nil")
	}

	if len(productInfos) == 0 {
		return nil
	}

	keys := make([]*datastore.Key, len(productInfos))
	for i := range productInfos {
		keys[i] = datastore.NameKey("ProductInfo",
			fmt.Sprintf("%d-%s-%s-%d",
				productInfos[i].ID,
				productInfos[i].Brand,
				productInfos[i].Country,
				productInfos[i].SubNumber),
			nil)
	}

	_, err := dc.client.PutMulti(ctx, keys, productInfos)
	return err
}

// SaveProductInfo saves a single ProductInfo document
func (dc *DatastoreClient) SaveProductInfo(ctx context.Context, productInfo *domain.ProductInfo) error {
	if dc == nil || dc.client == nil {
		return fmt.Errorf("datastore client is nil")
	}

	key := datastore.NameKey("ProductInfo",
		fmt.Sprintf("%d-%s-%s-%d",
			productInfo.ID,
			productInfo.Brand,
			productInfo.Country,
			productInfo.SubNumber),
		nil)

	_, err := dc.client.Put(ctx, key, productInfo)
	return err
}

// GetProductInfo retrieves a ProductInfo document
func (dc *DatastoreClient) GetProductInfo(ctx context.Context, id int64, brand, country string) (*domain.ProductInfo, error) {
	if dc == nil || dc.client == nil {
		return nil, fmt.Errorf("datastore client is nil")
	}

	var result []domain.ProductInfo
	q := datastore.NewQuery("ProductInfo").
		Filter("ID =", id).
		Filter("Brand =", brand).
		Filter("Country =", country)

	_, err := dc.client.GetAll(ctx, q, &result)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, datastore.ErrNoSuchEntity
	}

	return &result[0], nil
}

// GetProductInfoByBrand retrieves all ProductInfo for a brand
func (dc *DatastoreClient) GetProductInfoByBrand(ctx context.Context, brand string) ([]domain.ProductInfo, error) {
	if dc == nil || dc.client == nil {
		return nil, fmt.Errorf("datastore client is nil")
	}

	var result []domain.ProductInfo
	q := datastore.NewQuery("ProductInfo").Filter("Brand =", brand)

	_, err := dc.client.GetAll(ctx, q, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
