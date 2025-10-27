package order_storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Staspol216/gh1/models"
)

type OrderStorage struct {
	orders map[string]*models.Order
	path string
}

func New(path string) (*OrderStorage, error) {
	b, err := os.ReadFile(path)
	
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}
	
	orders := make(map[string]*models.Order)

	err = json.Unmarshal(b, &orders)
	
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	
	return &OrderStorage{
		orders: orders,
		path: path,
	}, nil
}