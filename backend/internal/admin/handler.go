package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/asthap08/ecommerce/internal/orders"
	"github.com/asthap08/ecommerce/internal/platform/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool   *pgxpool.Pool
	orders *orders.Service
}

func NewHandler(pool *pgxpool.Pool, ordersSvc *orders.Service) *Handler {
	return &Handler{pool: pool, orders: ordersSvc}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

const lowStockDefault = 5

// Dashboard. Revenue now counts every order that has been (or will be) paid for
// and is not cancelled - paid, processing, shipped, delivered. The old query only
// counted status='paid', so revenue disappeared as soon as an order was packed
// and COD orders were never counted.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	var d struct {
		Orders          int   `json:"orders"`
		OpenOrders      int   `json:"open_orders"`
		RevenueMinor    int64 `json:"revenue_minor"`
		Products        int   `json:"products"`
		ActiveProducts  int   `json:"active_products"`
		LowStock        int   `json:"low_stock"`
		OutOfStock      int   `json:"out_of_stock"`
		Customers       int   `json:"customers"`
		PendingReturns  int   `json:"pending_returns"`
		OrdersToday     int   `json:"orders_today"`
		RevenueTodayMin int64 `json:"revenue_today_minor"`
	}
	err := h.pool.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM orders),
		(SELECT count(*) FROM orders WHERE status IN ('pending','paid','processing')),
		COALESCE((SELECT sum(total_minor) FROM orders WHERE status IN ('paid','processing','shipped','delivered')),0),
		(SELECT count(*) FROM products),
		(SELECT count(*) FROM products WHERE status='active'),
		(SELECT count(*) FROM inventory WHERE quantity-reserved > 0 AND quantity-reserved <= $1),
		(SELECT count(*) FROM inventory WHERE quantity-reserved <= 0),
		(SELECT count(*) FROM users WHERE role='customer'),
		(SELECT count(*) FROM return_requests WHERE status='requested'),
		(SELECT count(*) FROM orders WHERE created_at >= date_trunc('day', now())),
		COALESCE((SELECT sum(total_minor) FROM orders WHERE created_at >= date_trunc('day', now()) AND status IN ('paid','processing','shipped','delivered')),0)`,
		lowStockDefault).Scan(&d.Orders, &d.OpenOrders, &d.RevenueMinor, &d.Products, &d.ActiveProducts, &d.LowStock, &d.OutOfStock,
		&d.Customers, &d.PendingReturns, &d.OrdersToday, &d.RevenueTodayMin)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "dashboard could not be loaded")
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

// Orders supports ?status=, ?q= (id / e-mail / name / tracking), ?limit=, ?offset=.
func (h *Handler) Orders(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.Page(r, 25, 100)
	items, total, err := h.orders.ListAll(r.Context(), orders.ListFilter{
		Status: r.URL.Query().Get("status"), Query: r.URL.Query().Get("q"), Limit: limit, Offset: offset})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "orders could not be loaded")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items, "meta": map[string]int{"limit": limit, "offset": offset, "total": total}})
}

type productRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Status       string `json:"status"`
	Category     string `json:"category"`
	ImageURL     string `json:"image_url"`
	VariantCount int    `json:"variant_count"`
	MinPrice     int64  `json:"min_price_minor"`
	Stock        int    `json:"stock"`
}

func (h *Handler) Products(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.Page(r, 25, 100)
	where, args := "true", []any{}
	if s := r.URL.Query().Get("status"); s != "" {
		args = append(args, s)
		where += " AND p.status=$" + strconv.Itoa(len(args))
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)+"%")
		where += " AND (p.name ILIKE $" + strconv.Itoa(len(args)) + " OR p.slug ILIKE $" + strconv.Itoa(len(args)) + ")"
	}
	var total int
	if err := h.pool.QueryRow(r.Context(), `SELECT count(*) FROM products p WHERE `+where, args...).Scan(&total); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "products could not be loaded")
		return
	}
	args = append(args, limit, offset)
	rows, err := h.pool.Query(r.Context(), `SELECT p.id,p.name,p.slug,p.status,p.category,
		COALESCE((SELECT url FROM product_images WHERE product_id=p.id ORDER BY position LIMIT 1),''),
		(SELECT count(*) FROM product_variants WHERE product_id=p.id),
		COALESCE((SELECT min(price_minor) FROM product_variants WHERE product_id=p.id),0),
		COALESCE((SELECT sum(GREATEST(i.quantity-i.reserved,0)) FROM inventory i JOIN product_variants v ON v.id=i.variant_id WHERE v.product_id=p.id),0)::int
		FROM products p WHERE `+where+` ORDER BY p.created_at DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "products could not be loaded")
		return
	}
	defer rows.Close()
	out := make([]productRow, 0)
	for rows.Next() {
		var p productRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Status, &p.Category, &p.ImageURL, &p.VariantCount, &p.MinPrice, &p.Stock); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "products could not be loaded")
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": out, "meta": map[string]int{"limit": limit, "offset": offset, "total": total}})
}

type inventoryRow struct {
	VariantID   string `json:"variant_id"`
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	Reserved    int    `json:"reserved"`
	Available   int    `json:"available"`
}

// Inventory lists stock per variant; ?low=1 limits to low/out of stock.
func (h *Handler) Inventory(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.Page(r, 50, 200)
	cond := ""
	if r.URL.Query().Get("low") == "1" {
		cond = " WHERE COALESCE(i.quantity-i.reserved,0) <= " + strconv.Itoa(lowStockDefault)
	}
	rows, err := h.pool.Query(r.Context(), `SELECT v.id,v.sku,p.name||CASE WHEN v.name<>'' THEN ' – '||v.name ELSE '' END,
		COALESCE(i.quantity,0),COALESCE(i.reserved,0),GREATEST(COALESCE(i.quantity-i.reserved,0),0)
		FROM product_variants v JOIN products p ON p.id=v.product_id LEFT JOIN inventory i ON i.variant_id=v.id`+cond+
		` ORDER BY COALESCE(i.quantity-i.reserved,0), p.name LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "inventory could not be loaded")
		return
	}
	defer rows.Close()
	out := make([]inventoryRow, 0)
	for rows.Next() {
		var x inventoryRow
		if err := rows.Scan(&x.VariantID, &x.SKU, &x.ProductName, &x.Quantity, &x.Reserved, &x.Available); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "inventory could not be loaded")
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": out, "meta": map[string]int{"limit": limit, "offset": offset, "low_stock_threshold": lowStockDefault}})
}
