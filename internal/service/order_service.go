package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"eshop/internal/repository"
)

var ErrOrderForbidden = errors.New("order access forbidden")

type OrderService struct {
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
	rand        *rand.Rand
}

type CreateOrderInput struct {
	UserID    uint64
	ProductID uint64
	Quantity  int64
	IdemKey   string
}

func NewOrderService(orderRepo *repository.OrderRepository, productRepo *repository.ProductRepository) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*repository.Order, error) {
	if input.UserID == 0 {
		return nil, errors.New("user id is required")
	}
	if input.ProductID == 0 {
		return nil, errors.New("product id is required")
	}
	if input.Quantity <= 0 {
		return nil, errors.New("quantity must > 0")
	}
	if strings.TrimSpace(input.IdemKey) == "" {
		return nil, errors.New("idempotency key is required")
	}

	exists, err := s.orderRepo.GetByIdempotencyKey(ctx, input.IdemKey)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return exists, nil
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		tx := s.orderRepo.DB().WithContext(ctx).Begin()
		if tx.Error != nil {
			return nil, fmt.Errorf("begin tx: %w", tx.Error)
		}

		updatedProduct, err := s.productRepo.AdjustStockTx(ctx, tx, input.ProductID, -input.Quantity)
		if err != nil {
			_ = tx.Rollback().Error
			lastErr = err
			if errors.Is(err, repository.ErrOptimisticConflict) {
				time.Sleep(15 * time.Millisecond)
				continue
			}
			return nil, err
		}

		order := &repository.Order{
			OrderNo:        s.generateOrderNo(),
			UserID:         input.UserID,
			ProductID:      input.ProductID,
			Quantity:       input.Quantity,
			TotalPriceCent: updatedProduct.PriceCent * input.Quantity,
			Status:         repository.OrderStatusPending,
			IdempotencyKey: input.IdemKey,
		}
		if err := s.orderRepo.CreateTx(ctx, tx, order); err != nil {
			_ = tx.Rollback().Error
			lastErr = err
			existsOrder, findErr := s.orderRepo.GetByIdempotencyKey(ctx, input.IdemKey)
			if findErr == nil && existsOrder != nil {
				return existsOrder, nil
			}
			return nil, err
		}

		if err := tx.Commit().Error; err != nil {
			lastErr = err
			continue
		}

		_ = s.productRepo.InvalidateCache(ctx, input.ProductID)
		return order, nil
	}

	if lastErr == nil {
		lastErr = errors.New("create order retries exhausted")
	}
	return nil, lastErr
}

func (s *OrderService) PayOrder(ctx context.Context, userID, orderID uint64) (*repository.Order, error) {
	if userID == 0 || orderID == 0 {
		return nil, errors.New("invalid user or order id")
	}
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrOrderForbidden
	}
	if order.Status == repository.OrderStatusPaid {
		return order, nil
	}
	if order.Status != repository.OrderStatusPending {
		return nil, errors.New("order status cannot be paid")
	}
	if err := s.orderRepo.UpdateStatus(ctx, orderID, repository.OrderStatusPending, repository.OrderStatusPaid); err != nil {
		return nil, err
	}
	return s.orderRepo.GetByID(ctx, orderID)
}

func (s *OrderService) ListMyOrders(ctx context.Context, userID uint64, page, pageSize int) ([]repository.Order, int64, error) {
	if userID == 0 {
		return nil, 0, errors.New("invalid user id")
	}
	return s.orderRepo.ListByUser(ctx, userID, page, pageSize)
}

func (s *OrderService) generateOrderNo() string {
	return fmt.Sprintf("OD%s%04d", time.Now().Format("20060102150405"), s.rand.Intn(10000))
}
