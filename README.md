# EShop 微服务电商后端

一个基于 **Golang + Gin + GORM + MySQL + Redis** 的电商后端项目。

## 核心功能

- 用户管理
  - 用户注册
  - 用户登录（JWT）
  - 用户信息查询
- 商品管理
  - 商品创建/更新
  - 商品列表/详情查询
  - 商品缓存（Redis Cache Aside）
- 订单管理
  - 创建订单（`Idempotency-Key` 幂等）
  - 我的订单列表
  - 订单支付（模拟支付状态流转）
- 库存管理
  - 下单时事务内扣减库存
  - 乐观锁冲突重试，避免并发超卖

## 技术栈

- `Gin`：HTTP API 路由与参数校验
- `GORM`：数据访问层 ORM
- `MySQL`：核心业务数据持久化
- `Redis`：商品缓存与读流量削峰
- `JWT`：登录态鉴权

## 项目结构（实际仓库）

```text
.
├── api
│   ├── proto
│   │   └── product.proto
│   └── v1
│       ├── order_handler.go
│       ├── product_handler.go
│       ├── router.go
│       └── user_handler.go
├── cmd
│   ├── main.go
│   └── product
│       └── main.go
├── internal
│   ├── repository
│   │   ├── order_repository.go
│   │   ├── product_repository.go
│   │   └── user_repository.go
│   ├── service
│   │   ├── order_service.go
│   │   ├── product_service.go
│   │   └── user_service.go
│   ├── utils
│   │   ├── hash.go
│   │   └── jwt.go
│   └── product
│       ├── handler
│       ├── repository
│       └── service
├── migrations
│   ├── 001_create_users_table.sql
│   ├── 001_product.sql
│   ├── 002_create_products_table.sql
│   └── 003_create_orders_table.sql
├── go.mod
└── README.md
```

## 包职责说明

### `cmd`

- `cmd/main.go`：当前主入口，初始化 MySQL/Redis，装配依赖，启动 Gin 服务。
- `cmd/product/main.go`：历史入口（保留）。

### `api/v1`

- HTTP Handler 层。
- 负责请求参数校验、错误码映射、调用 Service。
- 各文件对应功能：
  - `user_handler.go`：注册/登录/用户信息。
  - `product_handler.go`：商品增改查。
  - `order_handler.go`：下单/支付/我的订单。
  - `router.go`：统一路由注册与 JWT 鉴权中间件。

### `internal/service`

- 业务编排层。
- 负责业务规则与事务流程：
  - `user_service.go`：密码哈希、登录签发 JWT。
  - `product_service.go`：商品业务校验与缓存失效。
  - `order_service.go`：幂等下单、库存扣减、支付状态流转。

### `internal/repository`

- 数据访问层（DAO）。
- 与 GORM/Redis 直接交互：
  - `user_repository.go`：用户表 CRUD。
  - `product_repository.go`：商品表 CRUD + Redis 缓存 + 库存扣减。
  - `order_repository.go`：订单表 CRUD + 幂等查询。

### `internal/utils`

- 通用工具包。
- `hash.go`：密码哈希与校验。
- `jwt.go`：JWT 生成与解析。

### `migrations`

- 数据库建表脚本。
- 建议按顺序执行：
  1. `001_create_users_table.sql`
  2. `002_create_products_table.sql`
  3. `003_create_orders_table.sql`

### `api/proto`

- gRPC 契约草案（当前 HTTP 版本未启用 gRPC Server）。

## Gin / GORM 在项目中的使用方式

- Gin：
  - `api/v1/router.go` 注册路由与中间件。
  - Handler 使用 `ShouldBindJSON` 做参数校验。
  - 中间件校验 `Authorization: Bearer <token>`。
- GORM：
  - Repository 封装 SQL 细节。
  - Service 只关心业务，不直接拼 SQL。
  - 订单创建使用事务 + `SELECT ... FOR UPDATE` + 乐观锁版本字段。

## 快速启动

1. 启动依赖（MySQL、Redis）
2. 创建数据库并执行迁移脚本
3. 设置环境变量（可选）

```bash
# 默认值如下（不设也可）
# MYSQL_DSN=root:root@tcp(127.0.0.1:3306)/eshop?charset=utf8mb4&parseTime=True&loc=Local
# REDIS_ADDR=127.0.0.1:6379
# REDIS_PASSWORD=
# HTTP_ADDR=:8080
# JWT_SECRET=dev-secret-change-me
```

4. 启动服务

```bash
go run ./cmd
```

## API 示例

### 1) 用户注册

```bash
curl -X POST http://127.0.0.1:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456","nickname":"Alice"}'
```

### 2) 用户登录

```bash
curl -X POST http://127.0.0.1:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'
```

### 3) 创建商品

```bash
curl -X POST http://127.0.0.1:8080/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{"sku":"SKU-1001","name":"Phone Case","description":"Black","price_cent":1999,"stock":100}'
```

### 4) 创建订单（需登录 Token + 幂等键）

```bash
curl -X POST http://127.0.0.1:8080/api/v1/orders \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Idempotency-Key: order-create-0001" \
  -H "Content-Type: application/json" \
  -d '{"product_id":1,"quantity":2}'
```

## 当前状态说明

- 这是一版可运行后端骨架，适合继续扩展为多服务部署。
- 已具备：分层结构、库存并发防护、JWT、幂等下单。
- 可继续增强：网关、独立 gRPC 服务、支付回调、消息队列、完整测试覆盖。
