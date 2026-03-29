package service

import (
	"context"
	"errors"
	"strings"

	"eshop/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

type CreateProductInput struct {
	SKU         string
	Name        string
	Description string
	PriceCent   int64
	Stock       int64
}

type UpdateProductInput struct {
	ID          uint64
	Name        string
	Description string
	PriceCent   int64
	Stock       int64
	Status      int8
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) Create(ctx context.Context, input CreateProductInput) (*repository.Product, error) {
	if strings.TrimSpace(input.SKU) == "" {
		return nil, errors.New("sku is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	if input.PriceCent <= 0 {
		return nil, errors.New("price_cent must > 0")
	}
	if input.Stock < 0 {
		return nil, errors.New("stock must >= 0")
	}
	p := &repository.Product{
		SKU:         strings.TrimSpace(input.SKU),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		PriceCent:   input.PriceCent,
		Stock:       input.Stock,
		Status:      1,
		Version:     0,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) Update(ctx context.Context, input UpdateProductInput) (*repository.Product, error) {
	if input.ID == 0 {
		return nil, errors.New("id is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, errors.New("name is required")
	}
	if input.PriceCent <= 0 {
		return nil, errors.New("price_cent must > 0")
	}
	if input.Stock < 0 {
		return nil, errors.New("stock must >= 0")
	}
	if input.Status != 1 && input.Status != 2 {
		return nil, errors.New("status must be 1 or 2")
	}
	p := &repository.Product{
		ID:          input.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		PriceCent:   input.PriceCent,
		Stock:       input.Stock,
		Status:      input.Status,
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.InvalidateCache(ctx, input.ID)
	return s.repo.GetByID(ctx, input.ID)
}

func (s *ProductService) Get(ctx context.Context, id uint64) (*repository.Product, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ProductService) List(ctx context.Context, page, pageSize int) ([]repository.Product, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}
