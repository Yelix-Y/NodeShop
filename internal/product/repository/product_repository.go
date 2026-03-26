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
	ErrNotFound            = errors.New("product not found")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
	ErrOptimisticConflict  = errors.New("optimistic lock conflict")
)

type Product struct {
	ID          uint64         `gorm:"primaryKey"`
	SKU         string         `gorm:"column:sku;size:64;not null"`
	Name        string         `gorm:"column:name;size:256;not null"`
	Description string         `gorm:"column:description;size:1024;not null"`
	PriceCent   int64          `gorm:"column:price_cent;not null"`
	Stock       int64          `gorm:"column:stock;not null"`
	Version     int64          `gorm:"column:version;not null"`
	Status      int8           `gorm:"column:status;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (Product) TableName() string {
	return "products"
}

type IdempotencyRecord struct {
	ID           uint64         `gorm:"primaryKey"`
	Operation    string         `gorm:"column:operation;size:64;not null"`
	IdemKey      string         `gorm:"column:idem_key;size:128;not null"`
	ResourceID   uint64         `gorm:"column:resource_id;not null"`
	Status       int8           `gorm:"column:status;not null"`
	ResponseJSON string         `gorm:"column:response_json"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (IdempotencyRecord) TableName() string {
	return "idempotency_records"
}

type ProductStockLedger struct {
	ID          uint64         `gorm:"primaryKey"`
	ProductID   uint64         `gorm:"column:product_id;not null"`
	RequestID   string         `gorm:"column:request_id;size:64;not null"`
	Delta       int64          `gorm:"column:delta;not null"`
	BeforeStock int64          `gorm:"column:before_stock;not null"`
	AfterStock  int64          `gorm:"column:after_stock;not null"`
	Reason      string         `gorm:"column:reason;size:64;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (ProductStockLedger) TableName() string {
	return "product_stock_ledger"
}

type OutboxEvent struct {
	ID          uint64         `gorm:"primaryKey"`
	EventID     string         `gorm:"column:event_id;size:64;not null"`
	Aggregate   string         `gorm:"column:aggregate;size:64;not null"`
	AggregateID uint64         `gorm:"column:aggregate_id;not null"`
	EventType   string         `gorm:"column:event_type;size:64;not null"`
	Payload     string         `gorm:"column:payload;type:json;not null"`
	Status      int8           `gorm:"column:status;not null"`
	RetryCount  int            `gorm:"column:retry_count;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

type ProductFilter struct {
	Page     int
	PageSize int
	Status   *int8
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

func (r *ProductRepository) BeginTx(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("begin tx: %w", tx.Error)
	}
	return tx, nil
}

func (r *ProductRepository) CommitTx(tx *gorm.DB) error {
	if tx == nil {
		return errors.New("nil tx")
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *ProductRepository) RollbackTx(tx *gorm.DB) error {
	if tx == nil {
		return nil
	}
	if err := tx.Rollback().Error; err != nil {
		return fmt.Errorf("rollback tx: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetProductByID(ctx context.Context, id uint64) (*Product, error) {
	cacheKey := r.productCacheKey(id)
	cacheValue, err := r.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if cacheValue == "__nil__" {
			return nil, ErrNotFound
		}
		var product Product
		if unmarshalErr := json.Unmarshal([]byte(cacheValue), &product); unmarshalErr == nil {
			return &product, nil
		}
	}
	if err != nil && !errors.Is(err, redis.Nil) {
		// Redis 异常时降级 DB，错误不阻断主链路。
	}

	var product Product
	if dbErr := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error; dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			setErr := r.redis.Set(ctx, cacheKey, "__nil__", 30*time.Second).Err()
			if setErr != nil && !errors.Is(setErr, redis.Nil) {
				// Empty cache 失败不影响结果返回。
			}
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query product by id: %w", dbErr)
	}

	if setErr := r.setProductCache(ctx, &product); setErr != nil {
		// cache 写失败降级
	}
	return &product, nil
}

func (r *ProductRepository) ListProducts(ctx context.Context, f ProductFilter) ([]Product, int64, error) {
	page := f.Page
	if page <= 0 {
		page = 1
	}
	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&Product{}).Where("deleted_at IS NULL")
	if f.Status != nil {
		query = query.Where("status = ?", *f.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count products: %w", err)
	}

	var products []Product
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	return products, total, nil
}

func (r *ProductRepository) GetProductByIDInTx(ctx context.Context, tx *gorm.DB, id uint64) (*Product, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var product Product
	if err := tx.WithContext(ctx).Where("id = ?", id).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query product in tx: %w", err)
	}
	return &product, nil
}

func (r *ProductRepository) CreateProduct(ctx context.Context, tx *gorm.DB, p *Product) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if p == nil {
		return errors.New("product is nil")
	}
	if err := tx.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("create product: %w", err)
	}
	return nil
}

func (r *ProductRepository) UpdateProduct(ctx context.Context, tx *gorm.DB, p *Product) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if p == nil || p.ID == 0 {
		return errors.New("invalid product")
	}

	result := tx.WithContext(ctx).Model(&Product{}).
		Where("id = ? AND deleted_at IS NULL", p.ID).
		Updates(map[string]any{
			"name":        p.Name,
			"description": p.Description,
			"price_cent":  p.PriceCent,
			"status":      p.Status,
			"updated_at":  time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("update product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProductRepository) AdjustStock(ctx context.Context, tx *gorm.DB, productID uint64, delta int64) (*Product, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}

	var current Product
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("select for update product: %w", err)
	}

	if current.Stock+delta < 0 {
		return nil, ErrInsufficientStock
	}

	result := tx.WithContext(ctx).
		Model(&Product{}).
		Where("id = ? AND version = ?", current.ID, current.Version).
		Updates(map[string]any{
			"stock":      gorm.Expr("stock + ?", delta),
			"version":    gorm.Expr("version + 1"),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return nil, fmt.Errorf("adjust stock: %w", result.Error)
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

func (r *ProductRepository) SaveStockLedger(ctx context.Context, tx *gorm.DB, ledger *ProductStockLedger) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if ledger == nil {
		return errors.New("ledger is nil")
	}
	if err := tx.WithContext(ctx).Create(ledger).Error; err != nil {
		return fmt.Errorf("save stock ledger: %w", err)
	}
	return nil
}

func (r *ProductRepository) SaveIdempotencyDone(ctx context.Context, tx *gorm.DB, operation, key string, resourceID uint64, response any) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal idempotency response: %w", err)
	}
	record := IdempotencyRecord{
		Operation:    operation,
		IdemKey:      key,
		ResourceID:   resourceID,
		Status:       1,
		ResponseJSON: string(payload),
	}
	if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "operation"}, {Name: "idem_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"resource_id", "status", "response_json", "updated_at",
		}),
	}).Create(&record).Error; err != nil {
		return fmt.Errorf("save idempotency record: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetIdempotencyRecord(ctx context.Context, operation, key string) (*IdempotencyRecord, error) {
	var rec IdempotencyRecord
	err := r.db.WithContext(ctx).
		Where("operation = ? AND idem_key = ?", operation, key).
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get idempotency record: %w", err)
	}
	return &rec, nil
}

func (r *ProductRepository) InvalidateProductCache(ctx context.Context, id uint64) error {
	if err := r.redis.Del(ctx, r.productCacheKey(id)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("invalidate product cache: %w", err)
	}
	return nil
}

func (r *ProductRepository) SaveOutboxEvent(
	ctx context.Context,
	tx *gorm.DB,
	eventID string,
	aggregate string,
	aggregateID uint64,
	eventType string,
	payload any,
) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}
	event := OutboxEvent{
		EventID:     eventID,
		Aggregate:   aggregate,
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     string(b),
		Status:      0,
		RetryCount:  0,
	}
	if err := tx.WithContext(ctx).Create(&event).Error; err != nil {
		return fmt.Errorf("save outbox event: %w", err)
	}
	return nil
}

func (r *ProductRepository) setProductCache(ctx context.Context, p *Product) error {
	if p == nil {
		return errors.New("product is nil")
	}
	bytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal product cache: %w", err)
	}
	jitterSeconds := r.rand.Intn(120)
	ttl := 10*time.Minute + time.Duration(jitterSeconds)*time.Second
	if err := r.redis.Set(ctx, r.productCacheKey(p.ID), string(bytes), ttl).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("set product cache: %w", err)
	}
	return nil
}

func (r *ProductRepository) productCacheKey(id uint64) string {
	return "product:" + strconv.FormatUint(id, 10)
}
