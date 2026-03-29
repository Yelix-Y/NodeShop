package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"eshop/internal/repository"
	"eshop/internal/service"
)

type ProductHandler struct {
	service *service.ProductService
}

type createProductRequest struct {
	SKU         string `json:"sku" binding:"required,max=64"`
	Name        string `json:"name" binding:"required,max=256"`
	Description string `json:"description" binding:"max=1024"`
	PriceCent   int64  `json:"price_cent" binding:"required,gt=0"`
	Stock       int64  `json:"stock" binding:"required,gte=0"`
}

type updateProductRequest struct {
	Name        string `json:"name" binding:"required,max=256"`
	Description string `json:"description" binding:"max=1024"`
	PriceCent   int64  `json:"price_cent" binding:"required,gt=0"`
	Stock       int64  `json:"stock" binding:"required,gte=0"`
	Status      int8   `json:"status" binding:"required,oneof=1 2"`
}

func NewProductHandler(s *service.ProductService) *ProductHandler {
	return &ProductHandler{service: s}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req createProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.service.Create(c.Request.Context(), service.CreateProductInput{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Stock:       req.Stock,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req updateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.service.Update(c.Request.Context(), service.UpdateProductInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCent:   req.PriceCent,
		Stock:       req.Stock,
		Status:      req.Status,
	})
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	product, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.service.List(c.Request.Context(), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": list, "total": total, "page": page, "size": size})
}

func parseUintParam(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
