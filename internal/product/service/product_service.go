package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"eshop/internal/product/repository"
)

const (
	OperationCreateProduct = "create_product"
	OperationUpdateProduct = "update_product"
	OperationAdjustStock   = "adjust_stock"
)

type ProductService struct {
	repo  *repository.ProductRepository
	redis *redis.Client
}

type CreateProductInput struct {
	SKU          string
	Name         string
	Description  string
	PriceCent    int64
	InitialStock int64
}

type UpdateProductInput struct {
	ID          uint64
	Name        string
	Description string
	PriceCent   int64
	Status      int8
}

type ListProductsInput struct {
	Page     int
	PageSize int
	Status   *int8
}

func NewProductService(repo *repository.ProductRepository, redisClient *redis.Client) *ProductService {
	return &ProductService{repo: repo, redis: redisClient}
}

func (s *ProductService) GetProduct(ctx context.Context, id uint64) (*repository.Product, error) {
	if id == 0 {
		return nil, errors.New("id must be positive")
	}
	return s.repo.GetProductByID(ctx, id)
}

func (s *ProductService) ListProducts(ctx context.Context, input ListProductsInput) ([]repository.Product, int64, error) {
	return s.repo.ListProducts(ctx, repository.ProductFilter{
		Page:     input.Page,
		PageSize: input.PageSize,
		Status:   input.Status,
	})
}

func (s *ProductService) CreateProduct(ctx context.Context, idemKey string, input CreateProductInput) (*repository.Product, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}

	if product, done, err := s.tryIdempotentFastReturn(ctx, OperationCreateProduct, idemKey); err != nil {
		return nil, err
	} else if done {
		return product, nil
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := s.repo.RollbackTx(tx)
			if rollbackErr != nil {
				log.Printf("panic rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	product := &repository.Product{
		SKU:         strings.TrimSpace(input.SKU),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		PriceCent:   input.PriceCent,
		Stock:       input.InitialStock,
		Version:     0,
		Status:      1,
	}
	if err := s.repo.CreateProduct(ctx, tx, product); err != nil {
		rollbackErr := s.repo.RollbackTx(tx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("create product failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	if err := s.repo.SaveIdempotencyDone(ctx, tx, OperationCreateProduct, idemKey, product.ID, product); err != nil {
		rollbackErr := s.repo.RollbackTx(tx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("save idempotency failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	if err := s.repo.CommitTx(tx); err != nil {
		return nil, err
	}

	if err := s.repo.InvalidateProductCache(ctx, product.ID); err != nil {
		log.Printf("invalidate cache after create failed, product_id=%d err=%v", product.ID, err)
	}

	return product, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, idemKey string, input UpdateProductInput) (*repository.Product, error) {
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	if product, done, err := s.tryIdempotentFastReturn(ctx, OperationUpdateProduct, idemKey); err != nil {
		return nil, err
	} else if done {
		return product, nil
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			rollbackErr := s.repo.RollbackTx(tx)
			if rollbackErr != nil {
				log.Printf("panic rollback failed: %v", rollbackErr)
			}
			panic(r)
		}
	}()

	product := &repository.Product{
		ID:          input.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		PriceCent:   input.PriceCent,
		Status:      input.Status,
	}
	if err := s.repo.UpdateProduct(ctx, tx, product); err != nil {
		rollbackErr := s.repo.RollbackTx(tx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("update product failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	updated, err := s.repo.GetProductByID(ctx, input.ID)
	if err != nil {
		rollbackErr := s.repo.RollbackTx(tx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("query updated product failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	if err := s.repo.SaveIdempotencyDone(ctx, tx, OperationUpdateProduct, idemKey, input.ID, updated); err != nil {
		rollbackErr := s.repo.RollbackTx(tx)
		if rollbackErr != nil {
			return nil, fmt.Errorf("save idempotency failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	if err := s.repo.CommitTx(tx); err != nil {
		return nil, err
	}
	if err := s.repo.InvalidateProductCache(ctx, input.ID); err != nil {
		log.Printf("invalidate cache after update failed, product_id=%d err=%v", input.ID, err)
	}
	return updated, nil
}

func (s *ProductService) AdjustStock(ctx context.Context, idemKey string, productID uint64, delta int64) (*repository.Product, error) {
	if productID == 0 {
		return nil, errors.New("product id must be positive")
	}
	if delta == 0 {
		return nil, errors.New("delta cannot be 0")
	}

	if product, done, err := s.tryIdempotentFastReturn(ctx, OperationAdjustStock, idemKey); err != nil {
		return nil, err
	} else if done {
		return product, nil
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			return nil, err
		}

		updated, adjustErr := s.repo.AdjustStock(ctx, tx, productID, delta)
		if adjustErr != nil {
			rollbackErr := s.repo.RollbackTx(tx)
			if rollbackErr != nil {
				return nil, fmt.Errorf("adjust stock failed: %w; rollback failed: %v", adjustErr, rollbackErr)
			}
			lastErr = adjustErr
			if errors.Is(adjustErr, repository.ErrIdempotencyConflict) {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			return nil, adjustErr
		}

		if err := s.repo.SaveIdempotencyDone(ctx, tx, OperationAdjustStock, idemKey, productID, updated); err != nil {
			rollbackErr := s.repo.RollbackTx(tx)
			if rollbackErr != nil {
				return nil, fmt.Errorf("save idempotency failed: %w; rollback failed: %v", err, rollbackErr)
			}
			return nil, err
		}

		if err := s.repo.CommitTx(tx); err != nil {
			lastErr = err
			continue
		}
		if err := s.repo.InvalidateProductCache(ctx, productID); err != nil {
			log.Printf("invalidate cache after adjust stock failed, product_id=%d err=%v", productID, err)
		}
		return updated, nil
	}

	if lastErr == nil {
		lastErr = errors.New("stock adjust retries exhausted")
	}
	return nil, lastErr
}

func (s *ProductService) tryIdempotentFastReturn(ctx context.Context, operation, idemKey string) (*repository.Product, bool, error) {
	if strings.TrimSpace(idemKey) == "" {
		return nil, false, errors.New("missing idempotency key")
	}

	ok, err := s.redis.SetNX(ctx, s.idempotencyLockKey(operation, idemKey), "1", 24*time.Hour).Result()
	if err != nil {
		log.Printf("redis idempotency gate unavailable, fallback to mysql unique key, operation=%s err=%v", operation, err)
		return nil, false, nil
	}
	if ok {
		return nil, false, nil
	}

	rec, err := s.repo.GetIdempotencyRecord(ctx, operation, idemKey)
	if err != nil {
		return nil, false, err
	}
	if rec == nil {
		return nil, false, repository.ErrIdempotencyConflict
	}
	var product repository.Product
	if unmarshalErr := json.Unmarshal([]byte(rec.ResponseJSON), &product); unmarshalErr != nil {
		return nil, false, fmt.Errorf("decode idempotency response: %w", unmarshalErr)
	}
	return &product, true, nil
}

func (s *ProductService) idempotencyLockKey(operation, idemKey string) string {
	return "idem:" + operation + ":" + idemKey
}

func validateCreateInput(input CreateProductInput) error {
	if strings.TrimSpace(input.SKU) == "" {
		return errors.New("sku is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("name is required")
	}
	if input.PriceCent <= 0 {
		return errors.New("price_cent must be > 0")
	}
	if input.InitialStock < 0 {
		return errors.New("initial_stock must be >= 0")
	}
	return nil
}

func validateUpdateInput(input UpdateProductInput) error {
	if input.ID == 0 {
		return errors.New("id must be positive")
	}
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("name is required")
	}
	if input.PriceCent <= 0 {
		return errors.New("price_cent must be > 0")
	}
	if input.Status != 1 && input.Status != 2 {
		return errors.New("status must be 1 or 2")
	}
	return nil
}
