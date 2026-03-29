const API_BASE = window.API_BASE || "http://127.0.0.1:8080";
const TOKEN_KEY = "eshop_token";
const CART_KEY = "eshop_cart";

const byId = (id) => document.getElementById(id);

function toast(msg) {
  const el = byId("toast");
  el.textContent = msg;
  el.style.display = "block";
  setTimeout(() => {
    el.style.display = "none";
  }, 2200);
}

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}

function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token);
  renderAuthStatus();
}

function getCart() {
  try {
    return JSON.parse(localStorage.getItem(CART_KEY) || "[]");
  } catch {
    return [];
  }
}

function saveCart(list) {
  localStorage.setItem(CART_KEY, JSON.stringify(list));
}

async function request(path, opts = {}) {
  const headers = { "Content-Type": "application/json", ...(opts.headers || {}) };
  const token = getToken();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, { ...opts, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

function renderAuthStatus() {
  byId("auth-status").textContent = getToken() ? "已登录" : "未登录";
}

function addCart(product) {
  const cart = getCart();
  const idx = cart.findIndex((it) => it.product_id === product.id);
  if (idx >= 0) {
    cart[idx].quantity += 1;
  } else {
    cart.push({
      product_id: product.id,
      name: product.name,
      price_cent: product.price_cent,
      quantity: 1,
    });
  }
  saveCart(cart);
  renderCart();
  toast("已加入购物车");
}

function renderCart() {
  const cart = getCart();
  const box = byId("cart-list");
  if (!cart.length) {
    box.innerHTML = "<small>购物车为空</small>";
    return;
  }
  box.innerHTML = cart
    .map(
      (it, i) => `
      <div class="item">
        <div>
          <div>${it.name}</div>
          <small>￥${(it.price_cent / 100).toFixed(2)} x ${it.quantity}</small>
        </div>
        <div>
          <button onclick="changeQty(${i}, -1)">-</button>
          <button onclick="changeQty(${i}, 1)">+</button>
        </div>
      </div>`
    )
    .join("");
}

window.changeQty = function changeQty(i, delta) {
  const cart = getCart();
  if (!cart[i]) return;
  cart[i].quantity += delta;
  if (cart[i].quantity <= 0) cart.splice(i, 1);
  saveCart(cart);
  renderCart();
};

async function loadProducts() {
  try {
    const data = await request("/api/v1/products?page=1&page_size=20", { method: "GET" });
    const list = data.list || [];
    const box = byId("product-list");
    box.innerHTML = list
      .map(
        (p) => `
      <div class="item">
        <div>
          <div><b>${p.name}</b></div>
          <small>ID:${p.id} SKU:${p.sku || "-"} 库存:${p.stock}</small><br />
          <small>￥${(p.price_cent / 100).toFixed(2)}</small>
        </div>
        <div>
          <button onclick="viewProduct(${p.id})">详情</button>
          <button onclick='addProductToCart(${JSON.stringify(p)})'>加购</button>
        </div>
      </div>`
      )
      .join("");
  } catch (err) {
    toast(err.message);
  }
}

window.addProductToCart = function addProductToCart(product) {
  addCart(product);
};

window.viewProduct = async function viewProduct(id) {
  try {
    const product = await request(`/api/v1/products/${id}`, { method: "GET" });
    byId("product-detail-card").hidden = false;
    byId("product-detail").innerHTML = `
      <div><b>${product.name}</b></div>
      <div>${product.description || "暂无描述"}</div>
      <small>价格：￥${(product.price_cent / 100).toFixed(2)} | 库存：${product.stock}</small>
      <div style="margin-top:8px;">
        <button onclick='addProductToCart(${JSON.stringify(product)})'>加入购物车</button>
      </div>
    `;
  } catch (err) {
    toast(err.message);
  }
};

async function loadMyOrders() {
  try {
    const data = await request("/api/v1/orders/my?page=1&page_size=20", { method: "GET" });
    const list = data.list || [];
    byId("order-list").innerHTML =
      list.length === 0
        ? "<small>暂无订单</small>"
        : list
            .map(
              (o) => `
      <div class="item">
        <div>
          <div><b>${o.order_no}</b></div>
          <small>商品ID:${o.product_id} 数量:${o.quantity} 状态:${o.status}</small>
        </div>
        <small>￥${(o.total_price_cent / 100).toFixed(2)}</small>
      </div>`
            )
            .join("");
  } catch (err) {
    toast(err.message);
  }
}

async function checkout() {
  const token = getToken();
  if (!token) {
    toast("请先登录");
    return;
  }
  const cart = getCart();
  if (!cart.length) {
    toast("购物车为空");
    return;
  }

  try {
    for (const item of cart) {
      await request("/api/v1/orders", {
        method: "POST",
        headers: {
          "Idempotency-Key": `web-order-${item.product_id}-${Date.now()}`,
        },
        body: JSON.stringify({
          product_id: item.product_id,
          quantity: item.quantity,
        }),
      });
    }
    saveCart([]);
    renderCart();
    toast("下单成功");
    await loadMyOrders();
  } catch (err) {
    toast(err.message);
  }
}

byId("register-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  try {
    await request("/api/v1/users/register", {
      method: "POST",
      body: JSON.stringify({
        username: String(fd.get("username") || ""),
        password: String(fd.get("password") || ""),
        nickname: String(fd.get("nickname") || ""),
      }),
    });
    toast("注册成功");
  } catch (err) {
    toast(err.message);
  }
});

byId("login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  try {
    const data = await request("/api/v1/users/login", {
      method: "POST",
      body: JSON.stringify({
        username: String(fd.get("username") || ""),
        password: String(fd.get("password") || ""),
      }),
    });
    setToken(data.token);
    toast("登录成功");
  } catch (err) {
    toast(err.message);
  }
});

byId("btn-load-products").addEventListener("click", loadProducts);
byId("btn-load-orders").addEventListener("click", loadMyOrders);
byId("btn-cart").addEventListener("click", renderCart);
byId("btn-checkout").addEventListener("click", checkout);

renderAuthStatus();
renderCart();
loadProducts();
