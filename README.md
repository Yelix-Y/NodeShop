# EShop 微服务电商项目

[English Version](#english-version)

<a id="chinese-version"></a>
## 中文版（默认）

### 项目简介

这是一个基于 **Golang + Gin + GORM + MySQL + Redis** 的电商后端项目，并新增了两端前端示例：
- `frontend/web`：Web 端（HTML/CSS/JS）
- `frontend/android`：Android 端（Kotlin + Compose + Retrofit）

项目覆盖用户登录/注册、商品浏览与详情、购物车、下单结算等核心流程，支持与当前仓库 REST API 联调。

### 功能列表

- 用户管理：注册、登录、JWT 鉴权、查看当前用户
- 商品管理：商品列表、商品详情、商品创建/更新
- 订单管理：下单（幂等键）、我的订单、支付状态更新
- 库存管理：下单扣减库存，事务 + 乐观锁冲突重试
- 前端演示：Web/Android 调后端 API 完整演示链路

### 目录结构（当前仓库实际）

```text
.
├── api
│   ├── proto
│   │   └── product.proto
│   └── v1
│       ├── router.go
│       ├── user_handler.go
│       ├── product_handler.go
│       └── order_handler.go
├── cmd
│   ├── main.go
│   └── product/main.go
├── frontend
│   ├── web
│   │   ├── index.html
│   │   ├── styles.css
│   │   └── app.js
│   └── android
│       ├── README.md
│       └── app/src/main/java/com/eshop/app
│           ├── MainActivity.kt
│           ├── api
│           │   ├── ApiClient.kt
│           │   ├── ApiModels.kt
│           │   └── ApiService.kt
│           ├── data
│           │   └── SessionStore.kt
│           └── ui
│               ├── LoginScreen.kt
│               ├── ProductListScreen.kt
│               ├── ProductDetailScreen.kt
│               ├── CartScreen.kt
│               └── CheckoutScreen.kt
├── internal
│   ├── repository
│   │   ├── user_repository.go
│   │   ├── product_repository.go
│   │   └── order_repository.go
│   ├── service
│   │   ├── user_service.go
│   │   ├── product_service.go
│   │   └── order_service.go
│   ├── utils
│   │   ├── hash.go
│   │   └── jwt.go
│   └── product (历史实现，保留)
├── migrations
│   ├── 001_create_users_table.sql
│   ├── 002_create_products_table.sql
│   ├── 003_create_orders_table.sql
│   └── 001_product.sql
└── README.md
```

### 包职责说明

#### 后端
- `api/v1`: HTTP 接口层（Gin Handler），请求参数校验 + 响应封装
- `internal/service`: 业务编排层（登录、下单、扣库存、订单状态流转）
- `internal/repository`: 数据访问层（MySQL + Redis）
- `internal/utils`: 工具层（密码哈希、JWT）
- `migrations`: 数据库建表脚本
- `cmd/main.go`: 服务启动入口（依赖初始化、路由注册）

#### 前端
- `frontend/web`: 单页联调页面，包含登录/注册、商品列表/详情、购物车、下单
- `frontend/android`: Kotlin 示例代码，包含同等核心流程与 Retrofit API 调用

### 技术栈

- Backend: Golang, Gin, GORM, MySQL, Redis
- Web: HTML + CSS + JavaScript (Fetch API)
- Android: Kotlin, Jetpack Compose, Retrofit

### 启动后端

1. 启动 MySQL / Redis
2. 执行迁移脚本：
   - `migrations/001_create_users_table.sql`
   - `migrations/002_create_products_table.sql`
   - `migrations/003_create_orders_table.sql`
3. 启动服务：

```bash
go run ./cmd
```

默认环境变量：
- `MYSQL_DSN=root:root@tcp(127.0.0.1:3306)/eshop?charset=utf8mb4&parseTime=True&loc=Local`
- `REDIS_ADDR=127.0.0.1:6379`
- `REDIS_PASSWORD=`
- `HTTP_ADDR=:8080`
- `JWT_SECRET=dev-secret-change-me`

### 前端调用后端 API 示例

#### Web (JavaScript)

```javascript
const res = await fetch("http://127.0.0.1:8080/api/v1/products?page=1&page_size=20");
const data = await res.json();
console.log(data.list);
```

#### Android (Kotlin + Retrofit)

```kotlin
val products = ApiClient.service.listProducts(page = 1, pageSize = 20)
println(products.list)
```

### Web 页面本地启动方式

```bash
# 在仓库根目录执行任意静态文件服务器
# Python 示例
python -m http.server 5500

# 浏览器访问
# http://127.0.0.1:5500/frontend/web/
```

[切换到英文](#english-version)

---

<a id="english-version"></a>
## English Version

[Back to Chinese](#chinese-version)

### Overview

This repository is an e-commerce backend built with **Golang + Gin + GORM + MySQL + Redis**, plus frontend samples:
- `frontend/web`: Web demo (HTML/CSS/JS)
- `frontend/android`: Android demo (Kotlin + Compose + Retrofit)

It covers login/register, product list/detail, cart, and checkout against the current REST API.

### Features

- User: register, login, JWT auth, get profile
- Product: list, detail, create, update
- Order: create order with idempotency key, list my orders, pay order
- Inventory: transactional deduction with optimistic-lock retry
- Frontend demos: Web and Android API integration examples

### Backend Structure

- `api/v1`: Gin handlers and route registration
- `internal/service`: business orchestration
- `internal/repository`: MySQL/Redis data access
- `internal/utils`: password hash + JWT helpers
- `cmd/main.go`: service bootstrap

### Frontend Structure

- `frontend/web`: static web demo page + API calls
- `frontend/android`: Kotlin sample screens + Retrofit client

### API Call Example

```bash
curl -X POST http://127.0.0.1:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'
```

### Run Backend

```bash
go run ./cmd
```

### Run Web Demo

Serve repository root as static files and open:
- `http://127.0.0.1:5500/frontend/web/`

[Back to Chinese](#chinese-version)
