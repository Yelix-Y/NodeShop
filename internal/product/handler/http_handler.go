package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"eshop/internal/product/repository"
	"eshop/internal/product/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type CreateProductRequest struct {
	SKU          string `json:"sku" binding:"required,max=64"`
	Name         string `json:"name" binding:"required,max=256"`
	Description  string `json:"description" binding:"max=1024"`
	PriceCent    int64  `json:"price_cent" binding:"required,gt=0"`
	InitialStock int64  `json:"initial_stock" binding:"required,gte=0"`
}

type UpdateProductRequest struct {
	Name        string `json:"name" binding:"required,max=256"`
	Description string `json:"description" binding:"max=1024"`
	PriceCent   int64  `json:"price_cent" binding:"required,gt=0"`
	Status      int8   `json:"status" binding:"required,oneof=1 2"`
}

type AdjustStockRequest struct {
	Delta int64 `json:"delta" binding:"required"`
}

func (h *ProductHandler) RegisterRoutes(r *gin.Engine) {
	group := r.Group("/products")
	group.GET("/:id", h.GetProduct)
	group.GET("", h.ListProducts)
	group.POST("", h.CreateProduct)
	group.PUT("/:id", h.UpdateProduct)
	group.POST("/:id/stock", h.AdjustStock)
	r.GET("/health", h.Health)
}

func (h *ProductHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": "product", "status": "ok"})
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id, err := parsePathID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.GetProduct(c.Request.Context(), id)
	if err != nil {
		respondByError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	page, err := parsePositiveIntWithDefault(c.Query("page"), 1)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pageSize, err := parsePositiveIntWithDefault(c.Query("page_size"), 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var status *int8
	if rawStatus := c.Query("status"); rawStatus != "" {
		v, parseErr := strconv.Atoi(rawStatus)
		if parseErr != nil || (v != 1 && v != 2) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 1 or 2"})
			return
		}
		t := int8(v)
		status = &t
	}
	products, total, err := h.svc.ListProducts(c.Request.Context(), service.ListProductsInput{
		Page:     page,
		PageSize: pageSize,
		Status:   status,
	})
	if err != nil {
		respondByError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"list":  products,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	idemKey := c.GetHeader("Idempotency-Key")
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.CreateProduct(c.Request.Context(), idemKey, service.CreateProductInput{
		SKU:          req.SKU,
		Name:         req.Name,
		Description:  req.Description,
		PriceCent:    req.PriceCent,
		InitialStock: req.InitialStock,
	})
	if err != nil {
		respondByError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id, err := parsePathID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idemKey := c.GetHeader("Idempotency-Key")
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.UpdateProduct(c.Request.Context(), idemKey, service.UpdateProductInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Status:      req.Status,
	})
	if err != nil {
		respondByError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) AdjustStock(c *gin.Context) {
	id, err := parsePathID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idemKey := c.GetHeader("Idempotency-Key")
	var req AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.svc.AdjustStock(c.Request.Context(), idemKey, id, req.Delta)
	if err != nil {
		respondByError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

func respondByError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrInsufficientStock),
		errors.Is(err, repository.ErrIdempotencyConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func parsePathID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func parsePositiveIntWithDefault(raw string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, errors.New("invalid positive integer")
	}
	return v, nil
}
