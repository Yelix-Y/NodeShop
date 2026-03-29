package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrProductNotFound    = errors.New("product not found")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrOptimisticConflict = errors.New("optimistic lock conflict")
)

type Product struct {
	ID          uint64         `gorm:"primaryKey" json:"id"`
	SKU         string         `gorm:"column:sku;size:64;not null;uniqueIndex" json:"sku"`
	Name        string         `gorm:"column:name;size:256;not null" json:"name"`
	Description string         `gorm:"column:description;size:1024;not null" json:"description"`
	PriceCent   int64          `gorm:"column:price_cent;not null" json:"price_cent"`
	Stock       int64          `gorm:"column:stock;not null" json:"stock"`
	Version     int64          `gorm:"column:version;not null" json:"version"`
	Status      int8           `gorm:"column:status;not null" json:"status"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Product) TableName() string {
	return "products"
}

type ProductRepository struct {
	db    *gorm.DB
	redis *redis.Client
	rand  *rand.Rand
}

func NewProductRepository(db *gorm.DB, redisClient *redis.Client) *ProductRepository {
	return &ProductRepository{
		db:    db,
		redis: redisClient,
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *ProductRepository) Create(ctx context.Context, p *Product) error {
	if p == nil {
		return errors.New("product is nil")
	}
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *ProductRepository) Update(ctx context.Context, p *Product) error {
	if p == nil || p.ID == 0 {
		return errors.New("invalid product")
	}
	result := r.db.WithContext(ctx).Model(&Product{}).
		Where("id = ? AND deleted_at IS NULL", p.ID).
		Updates(map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"price_cent":  p.PriceCent,
			"stock":       p.Stock,
			"status":      p.Status,
			"updated_at":  time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("update product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uint64) (*Product, error) {
	cacheKey := r.cacheKey(id)
	cacheValue, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if cacheValue == "__nil__" {
			return nil, ErrProductNotFound
		}
		var product Product
		if unmarshalErr := json.Unmarshal([]byte(cacheValue), &product); unmarshalErr == nil {
			return &product, nil
		}
	}

	var product Product
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = r.redis.Set(ctx, cacheKey, "__nil__", 30*time.Second).Err()
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("query product by id: %w", err)
	}

	_ = r.setCache(ctx, &product)
	return &product, nil
}

func (r *ProductRepository) List(ctx context.Context, page, pageSize int) ([]Product, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&Product{}).Where("deleted_at IS NULL")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}
	var list []Product
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	return list, total, nil
}

func (r *ProductRepository) GetByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*Product, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var p Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("select product for update: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) AdjustStockTx(ctx context.Context, tx *gorm.DB, productID uint64, delta int64) (*Product, error) {
	p, err := r.GetByIDForUpdate(ctx, tx, productID)
	if err != nil {
		return nil, err
	}
	if p.Stock+delta < 0 {
		return nil, ErrInsufficientStock
	}
	result := tx.WithContext(ctx).
		Model(&Product{}).
		Where("id = ? AND version = ?", p.ID, p.Version).
		Updates(map[string]any{
			"stock":      gorm.Expr("stock + ?", delta),
			"version":    gorm.Expr("version + 1"),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("adjust stock tx: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrOptimisticConflict
	}
	var updated Product
	if err := tx.WithContext(ctx).First(&updated, "id = ?", productID).Error; err != nil {
		return nil, fmt.Errorf("query updated product: %w", err)
	}
	return &updated, nil
}

func (r *ProductRepository) InvalidateCache(ctx context.Context, id uint64) error {
	if err := r.redis.Del(ctx, r.cacheKey(id)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("invalidate product cache: %w", err)
	}
	return nil
}

func (r *ProductRepository) setCache(ctx context.Context, p *Product) error {
	if p == nil {
		return errors.New("product is nil")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal product: %w", err)
	}
	jitter := r.rand.Intn(180)
	ttl := 10*time.Minute + time.Duration(jitter)*time.Second
	if err := r.redis.Set(ctx, r.cacheKey(p.ID), string(b), ttl).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("set cache: %w", err)
	}
	return nil
}

func (r *ProductRepository) cacheKey(id uint64) string {
	return "product:" + strconv.FormatUint(id, 10)
}
