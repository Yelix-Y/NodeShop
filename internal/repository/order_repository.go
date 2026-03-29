package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	OrderStatusPending = "PENDING"
	OrderStatusPaid    = "PAID"
	OrderStatusCancel  = "CANCELLED"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type Order struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	OrderNo        string         `gorm:"column:order_no;size:64;not null;uniqueIndex" json:"order_no"`
	UserID         uint64         `gorm:"column:user_id;not null;index" json:"user_id"`
	ProductID      uint64         `gorm:"column:product_id;not null;index" json:"product_id"`
	Quantity       int64          `gorm:"column:quantity;not null" json:"quantity"`
	TotalPriceCent int64          `gorm:"column:total_price_cent;not null" json:"total_price_cent"`
	Status         string         `gorm:"column:status;size:32;not null" json:"status"`
	IdempotencyKey string         `gorm:"column:idempotency_key;size:128;not null;uniqueIndex" json:"-"`
	CreatedAt      time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) DB() *gorm.DB {
	return r.db
}

func (r *OrderRepository) GetByID(ctx context.Context, id uint64) (*Order, error) {
	var order Order
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("get order by id: %w", err)
	}
	return &order, nil
}

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, idemKey string) (*Order, error) {
	var order Order
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", idemKey).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get order by idem key: %w", err)
	}
	return &order, nil
}

func (r *OrderRepository) CreateTx(ctx context.Context, tx *gorm.DB, order *Order) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if order == nil {
		return errors.New("order is nil")
	}
	if err := tx.WithContext(ctx).Create(order).Error; err != nil {
		return fmt.Errorf("create order tx: %w", err)
	}
	return nil
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID uint64, page, pageSize int) ([]Order, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := r.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}
	var list []Order
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	return list, total, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id uint64, from, to string) error {
	result := r.db.WithContext(ctx).
		Model(&Order{}).
		Where("id = ? AND status = ?", id, from).
		Updates(map[string]any{"status": to, "updated_at": time.Now()})
	if result.Error != nil {
		return fmt.Errorf("update order status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrOrderNotFound
	}
	return nil
}
