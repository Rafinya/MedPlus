package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"medplus-pos/db"
	"medplus-pos/middleware"
	"medplus-pos/models"

	"golang.org/x/crypto/bcrypt"
)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func parseIDFromPath(prefix string, path string) (int, error) {
	value := strings.TrimPrefix(path, prefix)
	value = strings.Trim(value, "/")
	if value == "" {
		return 0, fmt.Errorf("missing id")
	}
	return strconv.Atoi(value)
}

func scanProduct(rows interface {
	Scan(dest ...interface{}) error
}) (models.Product, error) {
	var p models.Product
	var barcode, brand sql.NullString
	var expiry time.Time
	err := rows.Scan(&p.ID, &barcode, &p.Name, &brand, &p.Price, &p.Stock, &p.Category, &p.Icon, &p.Rx, &expiry, &p.ReorderLevel, &p.Status, &p.CreatedAt)
	if err != nil {
		return p, err
	}
	p.Barcode = barcode.String
	p.Brand = brand.String
	p.ExpiryDate = expiry
	return p, nil
}

func validRole(role string) bool {
	role = strings.TrimSpace(role)
	return role == "Admin" || role == "Pharmacist"
}

func requirePermission(w http.ResponseWriter, r *http.Request, permissionKey string) (*middleware.UserClaims, bool) {
	claims, ok := currentUser(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	allowed, err := db.HasPermission(claims.Role, permissionKey)
	if err != nil {
		log.Printf("Permission lookup failed for role=%s permission=%s: %v", claims.Role, permissionKey, err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return nil, false
	}
	if !allowed {
		respondWithError(w, http.StatusForbidden, "Access denied: insufficient permissions")
		return nil, false
	}
	return claims, true
}

type productWriteRequest struct {
	Barcode      *string  `json:"barcode"`
	Name         *string  `json:"name"`
	Brand        *string  `json:"brand"`
	Price        *float64 `json:"price"`
	Stock        *int     `json:"stock"`
	Category     *string  `json:"category"`
	Icon         *string  `json:"icon"`
	Rx           *bool    `json:"rx"`
	ExpiryDate   *string  `json:"expiry_date"`
	ReorderLevel *int     `json:"reorder_level"`
	Status       *string  `json:"status"`
}

func parseFlexibleDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format")
}

func productFromRequest(req productWriteRequest) (models.Product, error) {
	var expiryDate time.Time
	var err error
	if req.ExpiryDate != nil {
		expiryDate, err = parseFlexibleDate(*req.ExpiryDate)
		if err != nil {
			return models.Product{}, err
		}
	}

	var barcode string
	if req.Barcode != nil {
		barcode = strings.TrimSpace(*req.Barcode)
	}

	var name string
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	var brand string
	if req.Brand != nil {
		brand = strings.TrimSpace(*req.Brand)
	}

	var price float64
	if req.Price != nil {
		price = *req.Price
	}

	var stock int
	if req.Stock != nil {
		stock = *req.Stock
	}

	var category string
	if req.Category != nil {
		category = strings.TrimSpace(*req.Category)
	}

	var icon string
	if req.Icon != nil {
		icon = strings.TrimSpace(*req.Icon)
	}

	var rx bool
	if req.Rx != nil {
		rx = *req.Rx
	}

	var reorderLevel int
	if req.ReorderLevel != nil {
		reorderLevel = *req.ReorderLevel
	} else {
		reorderLevel = 10
	}

	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}

	return models.Product{
		Barcode:      barcode,
		Name:         name,
		Brand:        brand,
		Price:        price,
		Stock:        stock,
		Category:     category,
		Icon:         icon,
		Rx:           rx,
		ExpiryDate:   expiryDate,
		ReorderLevel: reorderLevel,
		Status:       status,
	}, nil
}

func validPaymentMethod(method string) bool {
	switch method {
	case "Cash", "M-Pesa":
		return true
	default:
		return false
	}
}

func validPaymentStatus(status string) bool {
	switch status {
	case "pending", "completed", "failed":
		return true
	default:
		return false
	}
}

func currentUser(r *http.Request) (*middleware.UserClaims, bool) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*middleware.UserClaims)
	return claims, ok && claims != nil
}

func isPublicRegistrationEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("ENABLE_PUBLIC_REGISTRATION")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) != "production"
	}
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	return nil
}

// Auth
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var user models.User
	err := db.DB.QueryRow(
		"SELECT id, username, password_hash, full_name, role, status, created_at FROM users WHERE username = $1",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	if err != nil {
		log.Printf("Database error in login: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if user.Status != "active" {
		respondWithError(w, http.StatusForbidden, "User account is suspended")
		return
	}

	
	// MAGIC BCRYPT CHECK: This is where we verify the password securely!
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// The password didn't match the math equation, so we reject them.
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		log.Printf("JWT generation error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusOK, models.LoginResponse{Token: token, User: user})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !isPublicRegistrationEnabled() {
		respondWithError(w, http.StatusForbidden, "Public registration is disabled")
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.FullName) == "" {
		respondWithError(w, http.StatusBadRequest, "username, password and full_name are required")
		return
	}

	req.Role = "Pharmacist"
	if err := validatePassword(req.Password); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}


	// SECURE PASSWORD GENERATION: This is where the magic happens!
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		// If the computer fails to scramble the password, we stop here.
		respondWithError(w, http.StatusInternalServerError, "Could not secure password")
		return
	}

	var user models.User
	err = db.DB.QueryRow(
		`INSERT INTO users (username, password_hash, full_name, role, status)
		 VALUES ($1, $2, $3, $4, 'active')
		 RETURNING id, username, full_name, role, status, created_at`,
		req.Username, string(hashedBytes), req.FullName, req.Role,
	).Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Username already exists")
			return
		}
		log.Printf("Register error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusCreated, models.LoginResponse{Token: token, User: user})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := currentUser(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var user models.User
	err := db.DB.QueryRow(
		"SELECT id, username, full_name, role, status, created_at FROM users WHERE id = $1",
		claims.UserID,
	).Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err == nil {
		user.Permissions, _ = db.PermissionsForRole(user.Role)
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, user)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Logout successful. Discard token on client."})
}

// Products and inventory
func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	category := r.URL.Query().Get("category")
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	query := `SELECT id, barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status, created_at
		FROM products WHERE status = 'active'`
	args := []interface{}{}
	idx := 1

	if category != "" && category != "all" {
		query += fmt.Sprintf(" AND category = $%d", idx)
		args = append(args, category)
		idx++
	}
	if search != "" {
		query += fmt.Sprintf(" AND (LOWER(name) LIKE LOWER($%d) OR LOWER(brand) LIKE LOWER($%d) OR barcode LIKE $%d)", idx, idx, idx)
		args = append(args, "%"+search+"%")
		idx++
	}
	query += " ORDER BY name ASC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Query error in GetProducts: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		products = append(products, p)
	}
	respondWithJSON(w, http.StatusOK, products)
}

func GetInventoryHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "inventory.view"); !ok {
		return
	}
	rows, err := db.DB.Query(`SELECT id, barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status, created_at
		FROM products ORDER BY stock ASC, name ASC`)
	if err != nil {
		log.Printf("Query error in GetInventory: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	inventory := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		inventory = append(inventory, p)
	}
	respondWithJSON(w, http.StatusOK, inventory)
}

func ProductCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetInventoryHandler(w, r)
	case http.MethodPost:
		CreateProductHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ProductItemHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetProductByIDHandler(w, r)
	case http.MethodPut:
		UpdateProductHandler(w, r)
	case http.MethodDelete:
		DeleteProductHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "inventory.manage"); !ok {
		return
	}
	var req productWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	product, err := productFromRequest(req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid expiry date format")
		return
	}

	if strings.TrimSpace(product.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "Product name is required")
		return
	}
	if product.Status == "" {
		product.Status = "active"
	}
	if product.Icon == "" {
		product.Icon = "💊"
	}
	if product.ExpiryDate.IsZero() {
		product.ExpiryDate = time.Now().AddDate(1, 0, 0)
	}

	var barcodeVal interface{}
	if product.Barcode != "" {
		barcodeVal = product.Barcode
	}

	var brandVal interface{}
	if product.Brand != "" {
		brandVal = product.Brand
	}

	err = db.DB.QueryRow(
		`INSERT INTO products (barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, created_at`,
		barcodeVal, product.Name, brandVal, product.Price, product.Stock, product.Category, product.Icon, product.Rx, product.ExpiryDate, product.ReorderLevel, product.Status,
	).Scan(&product.ID, &product.CreatedAt)
	if err != nil {
		log.Printf("Create product error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusCreated, product)
}

func GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath("/api/products/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product id")
		return
	}

	row := db.DB.QueryRow(`SELECT id, barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status, created_at
		FROM products WHERE id = $1`, id)
	product, err := scanProduct(row)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Product not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, product)
}

func UpdateProductHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "inventory.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/products/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product id")
		return
	}

	// Fetch current product from database to merge values and prevent wiping them out
	var currentBarcode sql.NullString
	var currentName string
	var currentBrand sql.NullString
	var currentPrice float64
	var currentStock int
	var currentCategory string
	var currentIcon string
	var currentRx bool
	var currentExpiryDate time.Time
	var currentReorderLevel int
	var currentStatus string

	err = db.DB.QueryRow(`SELECT barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status
		FROM products WHERE id = $1`, id).Scan(
		&currentBarcode, &currentName, &currentBrand, &currentPrice, &currentStock, &currentCategory, &currentIcon, &currentRx, &currentExpiryDate, &currentReorderLevel, &currentStatus,
	)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Product not found")
		return
	} else if err != nil {
		log.Printf("Update product error fetching existing: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var req productWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Merge incoming request with existing product data
	barcode := currentBarcode.String
	if req.Barcode != nil {
		barcode = *req.Barcode
	}
	var barcodeVal interface{}
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		barcodeVal = nil
	} else {
		barcodeVal = barcode
	}

	name := currentName
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	brand := currentBrand.String
	if req.Brand != nil {
		brand = *req.Brand
	}
	var brandVal interface{}
	brand = strings.TrimSpace(brand)
	if brand == "" {
		brandVal = nil
	} else {
		brandVal = brand
	}

	price := currentPrice
	if req.Price != nil {
		price = *req.Price
	}

	stock := currentStock
	if req.Stock != nil {
		stock = *req.Stock
	}

	category := currentCategory
	if req.Category != nil {
		category = strings.TrimSpace(*req.Category)
	}

	icon := currentIcon
	if req.Icon != nil {
		icon = strings.TrimSpace(*req.Icon)
	}

	rx := currentRx
	if req.Rx != nil {
		rx = *req.Rx
	}

	expiryDate := currentExpiryDate
	if req.ExpiryDate != nil {
		parsed, err := parseFlexibleDate(*req.ExpiryDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid expiry date format")
			return
		}
		expiryDate = parsed
	}

	reorderLevel := currentReorderLevel
	if req.ReorderLevel != nil {
		reorderLevel = *req.ReorderLevel
	}

	status := currentStatus
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}

	result, err := db.DB.Exec(`UPDATE products
		SET barcode = $1, name = $2, brand = $3, price = $4, stock = $5, category = $6, icon = $7, rx = $8, expiry_date = $9, reorder_level = $10, status = $11
		WHERE id = $12`,
		barcodeVal, name, brandVal, price, stock, category, icon, rx, expiryDate, reorderLevel, status, id)
	if err != nil {
		log.Printf("Update product error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "Product not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Product updated successfully"})
}

func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "inventory.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/products/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product id")
		return
	}
	result, err := db.DB.Exec("UPDATE products SET status = 'inactive' WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "Product not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Product archived successfully"})
}

// Users admin CRUD
func UsersCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListUsersHandler(w, r)
	case http.MethodPost:
		AdminCreateUserHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func UserItemHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetUserByIDHandler(w, r)
	case http.MethodPut:
		UpdateUserHandler(w, r)
	case http.MethodDelete:
		DeleteUserHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "users.manage"); !ok {
		return
	}
	rows, err := db.DB.Query("SELECT id, username, full_name, role, status, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		users = append(users, user)
	}
	respondWithJSON(w, http.StatusOK, users)
}

func AdminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "users.manage"); !ok {
		return
	}
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if !validRole(req.Role) {
		respondWithError(w, http.StatusBadRequest, "Invalid role")
		return
	}
	if err := validatePassword(req.Password); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not secure password")
		return
	}

	var user models.User
	err = db.DB.QueryRow(`INSERT INTO users (username, password_hash, full_name, role, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id, username, full_name, role, status, created_at`, req.Username, string(hash), req.FullName, req.Role).
		Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Username already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusCreated, user)
}

func GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "users.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/users/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}
	var user models.User
	err = db.DB.QueryRow("SELECT id, username, full_name, role, status, created_at FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, user)
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "users.manage")
	if !ok {
		return
	}
	id, err := parseIDFromPath("/api/users/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = strings.TrimSpace(r.URL.Query().Get("status"))
	}
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "inactive" {
		status = "active"
	}

	if req.Role != "" && !validRole(req.Role) {
		respondWithError(w, http.StatusBadRequest, "Invalid role")
		return
	}

	if claims != nil && claims.UserID == id && status == "inactive" {
		respondWithError(w, http.StatusBadRequest, "You cannot deactivate your own logged-in admin account")
		return
	}

	if strings.TrimSpace(req.Password) != "" {
		if err := validatePassword(req.Password); err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not secure password")
			return
		}
		_, err = db.DB.Exec("UPDATE users SET username = COALESCE(NULLIF($1, ''), username), full_name = COALESCE(NULLIF($2, ''), full_name), role = COALESCE(NULLIF($3, ''), role), status = $4, password_hash = $5 WHERE id = $6", req.Username, req.FullName, req.Role, status, string(hash), id)
		if err != nil {
			log.Printf("UpdateUser error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error updating user")
			return
		}
	} else {
		_, err = db.DB.Exec("UPDATE users SET username = COALESCE(NULLIF($1, ''), username), full_name = COALESCE(NULLIF($2, ''), full_name), role = COALESCE(NULLIF($3, ''), role), status = $4 WHERE id = $5", req.Username, req.FullName, req.Role, status, id)
		if err != nil {
			log.Printf("UpdateUser error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error updating user")
			return
		}
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "users.manage")
	if !ok {
		return
	}
	id, err := parseIDFromPath("/api/users/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	if claims != nil && claims.UserID == id {
		respondWithError(w, http.StatusBadRequest, "You cannot delete your own logged-in admin account")
		return
	}

	// Nullify user foreign key references in historical records so deletion can proceed
	db.DB.Exec("UPDATE sales SET cashier_id = NULL WHERE cashier_id = $1", id)
	db.DB.Exec("UPDATE prescriptions SET created_by = NULL WHERE created_by = $1", id)
	db.DB.Exec("UPDATE purchase_orders SET ordered_by = NULL WHERE ordered_by = $1", id)
	db.DB.Exec("UPDATE purchase_orders SET received_by = NULL WHERE received_by = $1", id)

	res, err := db.DB.Exec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		log.Printf("DeleteUser raw SQL failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "User deleted permanently"})
}

// POS / cart checkout

type CreateSaleRequest struct {
	Items            []SaleItemRequest `json:"items"`
	PaymentMethod    string            `json:"payment_method"`
	PaymentReference string            `json:"payment_reference,omitempty"`
	PaymentStatus    string            `json:"payment_status,omitempty"`
	CustomerID       *int              `json:"customer_id,omitempty"`
}

type SaleItemRequest struct {
	ProductID          int    `json:"product_id"`
	Quantity           int    `json:"quantity"`
	PrescriptionNumber string `json:"rx_number,omitempty"`
	PatientName        string `json:"patient,omitempty"`
	DoctorName         string `json:"doctor,omitempty"`
}

func ProcessSaleHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "pos.access")
	if !ok {
		return
	}

	var req CreateSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Items) == 0 {
		respondWithError(w, http.StatusBadRequest, "Cart cannot be empty")
		return
	}
	if !validPaymentMethod(req.PaymentMethod) {
		respondWithError(w, http.StatusBadRequest, "Invalid payment method")
		return
	}

	paymentStatus := strings.TrimSpace(strings.ToLower(req.PaymentStatus))
	if paymentStatus == "" {
		if req.PaymentMethod == "M-Pesa" {
			paymentStatus = "pending"
			if strings.TrimSpace(req.PaymentReference) != "" {
				var status string
				err := db.DB.QueryRow("SELECT status FROM mpesa_payments WHERE checkout_request_id = $1", strings.TrimSpace(req.PaymentReference)).Scan(&status)
				if err == nil && status == "completed" {
					paymentStatus = "completed"
				}
			}
		} else {
			paymentStatus = "completed"
		}
	}
	if !validPaymentStatus(paymentStatus) {
		respondWithError(w, http.StatusBadRequest, "Invalid payment status")
		return
	}
	if req.PaymentMethod == "M-Pesa" && strings.TrimSpace(req.PaymentReference) == "" {
		respondWithError(w, http.StatusBadRequest, "payment_reference is required for M-Pesa transactions")
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	var subtotal float64
	var itemsToInsert []models.SaleItem
	usedPrescriptionIDs := make(map[int]bool)

	if req.CustomerID != nil {
		var exists bool
		err = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1 AND status = 'active')", *req.CustomerID).Scan(&exists)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		if !exists {
			respondWithError(w, http.StatusBadRequest, "Selected customer is invalid or inactive")
			return
		}
	}

	for _, item := range req.Items {
		if item.Quantity <= 0 {
			respondWithError(w, http.StatusBadRequest, "Item quantity must be greater than zero")
			return
		}

		var currentStock int
		var price float64
		var name string
		var isRx bool
		var expiryDate time.Time
		err := tx.QueryRowContext(r.Context(), "SELECT name, price, stock, rx, expiry_date FROM products WHERE id = $1 AND status = 'active' FOR UPDATE", item.ProductID).
			Scan(&name, &price, &currentStock, &isRx, &expiryDate)
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Product ID %d not found", item.ProductID))
			return
		}
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		if !expiryDate.IsZero() && time.Now().After(expiryDate) {
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Cannot sell expired medicine '%s' (Expired on %s)", name, expiryDate.Format("2006-01-02")))
			return
		}
		if currentStock < item.Quantity {
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Insufficient stock for %s. Available: %d, Requested: %d", name, currentStock, item.Quantity))
			return
		}

		var rxID *int
		if isRx {
			if item.PrescriptionNumber == "" || item.PatientName == "" || item.DoctorName == "" {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Prescription details required for %s", name))
				return
			}
			
			var pID int
			var pStatus string
			err = tx.QueryRowContext(r.Context(), `SELECT id, status FROM prescriptions WHERE LOWER(prescription_number) = LOWER($1)`, strings.TrimSpace(item.PrescriptionNumber)).Scan(&pID, &pStatus)
			if err == sql.ErrNoRows {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Prescription '%s' not found", item.PrescriptionNumber))
				return
			} else if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Internal server error")
				return
			}

			if strings.ToLower(pStatus) != "active" {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Prescription '%s' is already %s", item.PrescriptionNumber, pStatus))
				return
			}

			var prescribedQty int
			err = tx.QueryRowContext(r.Context(), `SELECT quantity FROM prescription_items WHERE prescription_id = $1 AND product_id = $2`, pID, item.ProductID).Scan(&prescribedQty)
			if err == sql.ErrNoRows {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Prescription '%s' does not contain product %s", item.PrescriptionNumber, name))
				return
			} else if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Internal server error")
				return
			}

			if item.Quantity > prescribedQty {
				respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Requested quantity (%d) for %s exceeds the prescribed quantity (%d)", item.Quantity, name, prescribedQty))
				return
			}

			rxID = &pID
			usedPrescriptionIDs[pID] = true
		}

		_, err = tx.ExecContext(r.Context(), "UPDATE products SET stock = stock - $1 WHERE id = $2", item.Quantity, item.ProductID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		subtotal += price * float64(item.Quantity)
		itemsToInsert = append(itemsToInsert, models.SaleItem{
			ProductID:      item.ProductID,
			ProductName:    name,
			Quantity:       item.Quantity,
			Price:          price,
			PrescriptionID: rxID,
		})
	}

	tax := subtotal * 0.085
	total := subtotal + tax
	transactionID := "TXN-" + strconv.FormatInt(time.Now().UnixNano(), 10)[7:]

	var saleID int
	err = tx.QueryRowContext(r.Context(), `INSERT INTO sales (transaction_id, subtotal, tax, total, payment_method, payment_reference, payment_status, cashier_id, customer_id)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9) RETURNING id`, transactionID, subtotal, tax, total, req.PaymentMethod, strings.TrimSpace(req.PaymentReference), paymentStatus, claims.UserID, req.CustomerID).Scan(&saleID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	for _, item := range itemsToInsert {
		_, err = tx.ExecContext(r.Context(), `INSERT INTO sale_items (sale_id, product_id, quantity, price, prescription_id)
			VALUES ($1, $2, $3, $4, $5)`, saleID, item.ProductID, item.Quantity, item.Price, item.PrescriptionID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	for rxID := range usedPrescriptionIDs {
		_, err = tx.ExecContext(r.Context(), `UPDATE prescriptions SET status = 'dispensed' WHERE id = $1`, rxID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":           "Transaction completed successfully",
		"sale_id":           saleID,
		"transaction_id":    transactionID,
		"subtotal":          subtotal,
		"tax":               tax,
		"total":             total,
		"items":             itemsToInsert,
		"cashier":           claims.Username,
		"payment_reference": strings.TrimSpace(req.PaymentReference),
		"payment_status":    paymentStatus,
	})
}

func ListSalesHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "reports.view"); !ok {
		return
	}
	rows, err := db.DB.Query(`SELECT s.id, s.transaction_id, s.subtotal, s.tax, s.total, s.payment_method, COALESCE(s.payment_reference, ''), COALESCE(s.payment_status, 'completed'), COALESCE(s.cashier_id, 0), COALESCE(u.full_name, ''), s.customer_id, COALESCE(c.full_name, ''), s.created_at
		FROM sales s
		LEFT JOIN users u ON u.id = s.cashier_id
		LEFT JOIN customers c ON c.id = s.customer_id
		ORDER BY s.created_at DESC`)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	sales := []models.Sale{}
	for rows.Next() {
		var sale models.Sale
		if err := rows.Scan(&sale.ID, &sale.TransactionID, &sale.Subtotal, &sale.Tax, &sale.Total, &sale.PaymentMethod, &sale.PaymentReference, &sale.PaymentStatus, &sale.CashierID, &sale.CashierName, &sale.CustomerID, &sale.CustomerName, &sale.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		sales = append(sales, sale)
	}
	respondWithJSON(w, http.StatusOK, sales)
}

func GetReceiptHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/receipts/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid sale id")
		return
	}

	var sale models.Sale
	err = db.DB.QueryRow(`SELECT s.id, s.transaction_id, s.subtotal, s.tax, s.total, s.payment_method, COALESCE(s.payment_reference, ''), COALESCE(s.payment_status, 'completed'), COALESCE(s.cashier_id, 0), COALESCE(u.full_name, ''), s.customer_id, COALESCE(c.full_name, ''), s.created_at
		FROM sales s
		LEFT JOIN users u ON u.id = s.cashier_id
		LEFT JOIN customers c ON c.id = s.customer_id
		WHERE s.id = $1`, id).
		Scan(&sale.ID, &sale.TransactionID, &sale.Subtotal, &sale.Tax, &sale.Total, &sale.PaymentMethod, &sale.PaymentReference, &sale.PaymentStatus, &sale.CashierID, &sale.CashierName, &sale.CustomerID, &sale.CustomerName, &sale.CreatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Sale not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	rows, err := db.DB.Query(`SELECT si.id, si.sale_id, si.product_id, p.name, p.barcode, si.quantity, si.price, si.prescription_id
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1 ORDER BY si.id ASC`, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	items := []models.SaleItem{}
	for rows.Next() {
		var item models.SaleItem
		if err := rows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.ProductName, &item.ProductBarcode, &item.Quantity, &item.Price, &item.PrescriptionID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		items = append(items, item)
	}
	sale.Items = items

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"receipt_title": "MedPlus POS Receipt",
		"sale":          sale,
	})
}

// Supply chain
func SuppliersCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListSuppliersHandler(w, r)
	case http.MethodPost:
		CreateSupplierHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func SupplierItemHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetSupplierByIDHandler(w, r)
	case http.MethodPut:
		UpdateSupplierHandler(w, r)
	case http.MethodDelete:
		DeleteSupplierHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListSuppliersHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.view"); !ok {
		return
	}
	rows, err := db.DB.Query("SELECT id, name, contact_person, phone, email, address, status, created_at FROM suppliers ORDER BY name ASC")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	suppliers := []models.Supplier{}
	for rows.Next() {
		var supplier models.Supplier
		if err := rows.Scan(&supplier.ID, &supplier.Name, &supplier.ContactPerson, &supplier.Phone, &supplier.Email, &supplier.Address, &supplier.Status, &supplier.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		suppliers = append(suppliers, supplier)
	}
	respondWithJSON(w, http.StatusOK, suppliers)
}

func CreateSupplierHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.manage"); !ok {
		return
	}
	var supplier models.Supplier
	if err := json.NewDecoder(r.Body).Decode(&supplier); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if supplier.Status == "" {
		supplier.Status = "active"
	}
	err := db.DB.QueryRow(`INSERT INTO suppliers (name, contact_person, phone, email, address, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`, supplier.Name, supplier.ContactPerson, supplier.Phone, supplier.Email, supplier.Address, supplier.Status).
		Scan(&supplier.ID, &supplier.CreatedAt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusCreated, supplier)
}

func GetSupplierByIDHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.view"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/suppliers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid supplier id")
		return
	}
	var supplier models.Supplier
	err = db.DB.QueryRow("SELECT id, name, contact_person, phone, email, address, status, created_at FROM suppliers WHERE id = $1", id).
		Scan(&supplier.ID, &supplier.Name, &supplier.ContactPerson, &supplier.Phone, &supplier.Email, &supplier.Address, &supplier.Status, &supplier.CreatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Supplier not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, supplier)
}

func UpdateSupplierHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/suppliers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid supplier id")
		return
	}
	var supplier models.Supplier
	if err := json.NewDecoder(r.Body).Decode(&supplier); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	_, err = db.DB.Exec("UPDATE suppliers SET name = $1, contact_person = $2, phone = $3, email = $4, address = $5, status = $6 WHERE id = $7", supplier.Name, supplier.ContactPerson, supplier.Phone, supplier.Email, supplier.Address, supplier.Status, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Supplier updated successfully"})
}

func DeleteSupplierHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/suppliers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid supplier id")
		return
	}
	_, err = db.DB.Exec("UPDATE suppliers SET status = 'inactive' WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Supplier archived successfully"})
}

type purchaseOrderRequest struct {
	SupplierID         int                        `json:"supplier_id"`
	Status             string                     `json:"status"`
	Notes              string                     `json:"notes"`
	ExpectedDeliveryAt *time.Time                 `json:"expected_delivery_at,omitempty"`
	Items              []models.PurchaseOrderItem `json:"items"`
}

func validPurchaseOrderStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "draft", "submitted", "requested", "received", "cancelled":
		return true
	default:
		return false
	}
}

func PurchaseOrdersCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListPurchaseOrdersHandler(w, r)
	case http.MethodPost:
		CreatePurchaseOrderHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func PurchaseOrderItemHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/receive") {
		ReceivePurchaseOrderHandler(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		GetPurchaseOrderByIDHandler(w, r)
	case http.MethodPut:
		UpdatePurchaseOrderHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListPurchaseOrdersHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.view"); !ok {
		return
	}
	rows, err := db.DB.Query(`SELECT po.id, po.po_number, po.supplier_id, s.name, po.status, COALESCE(po.notes, ''), po.ordered_by, po.received_by, po.expected_delivery_at, po.ordered_at, po.received_at, po.updated_at,
			COALESCE(COUNT(poi.id), 0) AS item_count,
			COALESCE(SUM(poi.quantity), 0) AS requested_quantity,
			COALESCE(SUM(poi.received_quantity), 0) AS received_quantity
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		LEFT JOIN purchase_order_items poi ON poi.purchase_order_id = po.id
		GROUP BY po.id, po.po_number, po.supplier_id, s.name, po.status, po.notes, po.ordered_by, po.received_by, po.expected_delivery_at, po.ordered_at, po.received_at, po.updated_at
		ORDER BY po.ordered_at DESC`)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	orders := []models.PurchaseOrder{}
	for rows.Next() {
		var po models.PurchaseOrder
		if err := rows.Scan(&po.ID, &po.PONumber, &po.SupplierID, &po.SupplierName, &po.Status, &po.Notes, &po.OrderedBy, &po.ReceivedBy, &po.ExpectedDeliveryAt, &po.OrderedAt, &po.ReceivedAt, &po.UpdatedAt, &po.ItemCount, &po.RequestedQuantity, &po.ReceivedQuantity); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		orders = append(orders, po)
	}
	respondWithJSON(w, http.StatusOK, orders)
}

func CreatePurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "supply.manage")
	if !ok {
		return
	}
	var req purchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.SupplierID == 0 || len(req.Items) == 0 {
		respondWithError(w, http.StatusBadRequest, "supplier_id and items are required")
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "submitted"
	}
	if !validPurchaseOrderStatus(req.Status) || strings.EqualFold(req.Status, "received") {
		respondWithError(w, http.StatusBadRequest, "Invalid purchase order status")
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	poNumber := "PO-" + strconv.FormatInt(time.Now().UnixNano(), 10)[8:]
	var poID int
	err = tx.QueryRowContext(r.Context(), `INSERT INTO purchase_orders (po_number, supplier_id, status, notes, ordered_by, expected_delivery_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING id`, poNumber, req.SupplierID, strings.ToLower(req.Status), req.Notes, claims.UserID, req.ExpectedDeliveryAt).Scan(&poID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	for _, item := range req.Items {
		if item.ProductID == 0 || item.Quantity <= 0 {
			respondWithError(w, http.StatusBadRequest, "Each item must include product_id and positive quantity")
			return
		}
		_, err = tx.ExecContext(r.Context(), `INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost, received_quantity)
			VALUES ($1, $2, $3, $4, 0)`, poID, item.ProductID, item.Quantity, item.UnitCost)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{"message": "Purchase order created", "purchase_order_id": poID, "po_number": poNumber})
}

func GetPurchaseOrderByIDHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.view"); !ok {
		return
	}
	path := r.URL.Path
	if strings.Contains(path, "/receive") {
		path = strings.TrimSuffix(path, "/receive")
	}
	id, err := parseIDFromPath("/api/purchase-orders/", path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid purchase order id")
		return
	}

	var po models.PurchaseOrder
	err = db.DB.QueryRow(`SELECT po.id, po.po_number, po.supplier_id, s.name, po.status, COALESCE(po.notes, ''), po.ordered_by, po.received_by, po.expected_delivery_at, po.ordered_at, po.received_at, po.updated_at
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE po.id = $1`, id).
		Scan(&po.ID, &po.PONumber, &po.SupplierID, &po.SupplierName, &po.Status, &po.Notes, &po.OrderedBy, &po.ReceivedBy, &po.ExpectedDeliveryAt, &po.OrderedAt, &po.ReceivedAt, &po.UpdatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Purchase order not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	rows, err := db.DB.Query(`SELECT poi.id, poi.purchase_order_id, poi.product_id, p.name, poi.quantity, poi.unit_cost, poi.received_quantity
		FROM purchase_order_items poi
		JOIN products p ON p.id = poi.product_id
		WHERE poi.purchase_order_id = $1 ORDER BY poi.id ASC`, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	items := []models.PurchaseOrderItem{}
	for rows.Next() {
		var item models.PurchaseOrderItem
		if err := rows.Scan(&item.ID, &item.PurchaseOrderID, &item.ProductID, &item.ProductName, &item.Quantity, &item.UnitCost, &item.ReceivedQuantity); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		items = append(items, item)
	}
	po.Items = items
	respondWithJSON(w, http.StatusOK, po)
}

func UpdatePurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "supply.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/purchase-orders/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid purchase order id")
		return
	}

	var req purchaseOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.SupplierID == 0 || len(req.Items) == 0 {
		respondWithError(w, http.StatusBadRequest, "supplier_id and items are required")
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "draft"
	}
	if !validPurchaseOrderStatus(req.Status) || strings.EqualFold(req.Status, "received") {
		respondWithError(w, http.StatusBadRequest, "Invalid purchase order status")
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRowContext(r.Context(), `SELECT status FROM purchase_orders WHERE id = $1 FOR UPDATE`, id).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Purchase order not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if strings.EqualFold(currentStatus, "received") {
		respondWithError(w, http.StatusConflict, "Received purchase orders cannot be modified")
		return
	}

	_, err = tx.ExecContext(r.Context(), `UPDATE purchase_orders
		SET supplier_id = $1, status = $2, notes = $3, expected_delivery_at = $4, updated_at = NOW()
		WHERE id = $5`, req.SupplierID, strings.ToLower(req.Status), req.Notes, req.ExpectedDeliveryAt, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	_, err = tx.ExecContext(r.Context(), `DELETE FROM purchase_order_items WHERE purchase_order_id = $1`, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	for _, item := range req.Items {
		if item.ProductID == 0 || item.Quantity <= 0 {
			respondWithError(w, http.StatusBadRequest, "Each item must include product_id and positive quantity")
			return
		}
		_, err = tx.ExecContext(r.Context(), `INSERT INTO purchase_order_items (purchase_order_id, product_id, quantity, unit_cost, received_quantity)
			VALUES ($1, $2, $3, $4, 0)`, id, item.ProductID, item.Quantity, item.UnitCost)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "Purchase order updated successfully", "purchase_order_id": id})
}

func ReceivePurchaseOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	claims, ok := requirePermission(w, r, "supply.manage")
	if !ok {
		return
	}

	id, err := parseIDFromPath("/api/purchase-orders/", strings.TrimSuffix(r.URL.Path, "/receive"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid purchase order id")
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("ReceivePO begin tx error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error starting transaction")
		return
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(r.Context(), "SELECT status FROM purchase_orders WHERE id = $1 FOR UPDATE", id).Scan(&status)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Purchase order not found")
		return
	}
	if err != nil {
		log.Printf("ReceivePO query status error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error fetching status")
		return
	}
	if status == "received" {
		respondWithError(w, http.StatusConflict, "Purchase order has already been received")
		return
	}
	if status == "cancelled" {
		respondWithError(w, http.StatusBadRequest, "Cancelled purchase orders cannot be received")
		return
	}

	type itemToReceive struct {
		productID int
		quantity  int
	}
	var itemsToReceive []itemToReceive

	rows, err := tx.QueryContext(r.Context(), "SELECT product_id, quantity FROM purchase_order_items WHERE purchase_order_id = $1", id)
	if err != nil {
		log.Printf("ReceivePO query items error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error fetching order items")
		return
	}
	for rows.Next() {
		var pID, qty int
		if err := rows.Scan(&pID, &qty); err != nil {
			rows.Close()
			log.Printf("ReceivePO scan item error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error scanning items")
			return
		}
		itemsToReceive = append(itemsToReceive, itemToReceive{productID: pID, quantity: qty})
	}
	rows.Close()

	for _, item := range itemsToReceive {
		_, err = tx.ExecContext(r.Context(), "UPDATE products SET stock = stock + $1, status = CASE WHEN (stock + $1) > 0 THEN 'active' ELSE status END WHERE id = $2", item.quantity, item.productID)
		if err != nil {
			log.Printf("ReceivePO update product stock error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error updating product stock")
			return
		}
		_, err = tx.ExecContext(r.Context(), "UPDATE purchase_order_items SET received_quantity = quantity WHERE purchase_order_id = $1 AND product_id = $2", id, item.productID)
		if err != nil {
			log.Printf("ReceivePO update PO item error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal server error updating purchase order item")
			return
		}
	}

	var receivedByPtr *int
	if claims != nil && claims.UserID > 0 {
		var exists bool
		_ = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", claims.UserID).Scan(&exists)
		if exists {
			receivedByPtr = &claims.UserID
		}
	}

	_, err = tx.ExecContext(r.Context(), "UPDATE purchase_orders SET status = 'received', received_by = $1, received_at = NOW(), updated_at = NOW() WHERE id = $2", receivedByPtr, id)
	if err != nil {
		log.Printf("ReceivePO update PO status error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error updating purchase order status")
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("ReceivePO commit error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error committing receive transaction")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Purchase order received and inventory updated"})
}

// Reporting and analytics
func GetDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "dashboard.view"); !ok {
		return
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	var stats models.DashboardStats
	_ = db.DB.QueryRow("SELECT COALESCE(SUM(total), 0) FROM sales WHERE created_at >= $1", todayStart).Scan(&stats.TodaySales)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM sales WHERE created_at >= $1", todayStart).Scan(&stats.TransactionCount)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM prescriptions WHERE created_at >= $1", todayStart).Scan(&stats.PrescriptionCount)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM products WHERE stock <= reorder_level AND status = 'active'").Scan(&stats.LowStockCount)

	var yesterdaySales float64
	var yesterdayTx int
	var yesterdayRx int
	_ = db.DB.QueryRow("SELECT COALESCE(SUM(total), 0) FROM sales WHERE created_at >= $1 AND created_at < $2", yesterdayStart, todayStart).Scan(&yesterdaySales)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM sales WHERE created_at >= $1 AND created_at < $2", yesterdayStart, todayStart).Scan(&yesterdayTx)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM prescriptions WHERE created_at >= $1 AND created_at < $2", yesterdayStart, todayStart).Scan(&yesterdayRx)

	if yesterdaySales > 0 {
		stats.SalesChange = ((stats.TodaySales - yesterdaySales) / yesterdaySales) * 100
	}
	stats.TxChange = stats.TransactionCount - yesterdayTx
	stats.RxChange = stats.PrescriptionCount - yesterdayRx

	respondWithJSON(w, http.StatusOK, stats)
}

func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "reports.view"); !ok {
		return
	}
	from, to := reportRange(r)

	var summary struct {
		From             string  `json:"from"`
		To               string  `json:"to"`
		Revenue          float64 `json:"revenue"`
		Transactions     int     `json:"transactions"`
		UnitsSold        int     `json:"units_sold"`
		Prescriptions    int     `json:"prescriptions"`
		LowStockItems    int     `json:"low_stock_items"`
		InventoryValue   float64 `json:"inventory_value"`
		PurchaseOrderCnt int     `json:"purchase_order_count"`
	}
	summary.From = from.Format(time.RFC3339)
	summary.To = to.Format(time.RFC3339)

	_ = db.DB.QueryRow("SELECT COALESCE(SUM(total), 0), COUNT(*) FROM sales WHERE created_at >= $1 AND created_at <= $2", from, to).Scan(&summary.Revenue, &summary.Transactions)
	_ = db.DB.QueryRow(`SELECT COALESCE(SUM(quantity), 0) FROM sale_items si
		JOIN sales s ON s.id = si.sale_id WHERE s.created_at >= $1 AND s.created_at <= $2`, from, to).Scan(&summary.UnitsSold)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM prescriptions WHERE created_at >= $1 AND created_at <= $2", from, to).Scan(&summary.Prescriptions)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM products WHERE stock <= reorder_level AND status = 'active'").Scan(&summary.LowStockItems)
	_ = db.DB.QueryRow("SELECT COALESCE(SUM(price * stock), 0) FROM products WHERE status = 'active'").Scan(&summary.InventoryValue)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE ordered_at >= $1 AND ordered_at <= $2", from, to).Scan(&summary.PurchaseOrderCnt)

	topProducts := []map[string]interface{}{}
	rows, err := db.DB.Query(`SELECT p.name, COALESCE(SUM(si.quantity), 0) AS units, COALESCE(SUM(si.quantity * si.price), 0) AS revenue
		FROM products p
		LEFT JOIN sale_items si ON si.product_id = p.id
		LEFT JOIN sales s ON s.id = si.sale_id AND s.created_at >= $1 AND s.created_at <= $2
		GROUP BY p.name
		ORDER BY units DESC, revenue DESC
		LIMIT 10`, from, to)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			var units int
			var revenue float64
			if err := rows.Scan(&name, &units, &revenue); err == nil {
				topProducts = append(topProducts, map[string]interface{}{"name": name, "units": units, "revenue": revenue})
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"summary":      summary,
		"top_products": topProducts,
	})
}

func AnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "analytics.view"); !ok {
		return
	}
	daily := []map[string]interface{}{}
	rows, err := db.DB.Query(`SELECT TO_CHAR(created_at, 'YYYY-MM-DD') AS day, COALESCE(SUM(total), 0), COUNT(*)
		FROM sales
		WHERE created_at >= NOW() - INTERVAL '14 days'
		GROUP BY day
		ORDER BY day ASC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var revenue float64
			var transactions int
			if err := rows.Scan(&day, &revenue, &transactions); err == nil {
				daily = append(daily, map[string]interface{}{"day": day, "revenue": revenue, "transactions": transactions})
			}
		}
	}

	categoryBreakdown := []map[string]interface{}{}
	rows, err = db.DB.Query(`SELECT p.category, COALESCE(SUM(si.quantity), 0), COALESCE(SUM(si.quantity * si.price), 0)
		FROM products p
		LEFT JOIN sale_items si ON si.product_id = p.id
		GROUP BY p.category
		ORDER BY p.category ASC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var units int
			var revenue float64
			if err := rows.Scan(&category, &units, &revenue); err == nil {
				categoryBreakdown = append(categoryBreakdown, map[string]interface{}{"category": category, "units": units, "revenue": revenue})
			}
		}
	}

	paymentMethods := []map[string]interface{}{}
	rows, err = db.DB.Query(`SELECT payment_method, COUNT(*), COALESCE(SUM(total), 0)
		FROM sales
		GROUP BY payment_method
		ORDER BY payment_method ASC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var method string
			var count int
			var revenue float64
			if err := rows.Scan(&method, &count, &revenue); err == nil {
				paymentMethods = append(paymentMethods, map[string]interface{}{"method": method, "count": count, "revenue": revenue})
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"daily_sales":        daily,
		"category_breakdown": categoryBreakdown,
		"payment_methods":    paymentMethods,
		"generated_at":       time.Now().Format(time.RFC3339),
	})
}

func DatabaseOverviewHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "database.view")
	if !ok {
		return
	}
	accessToken := strings.TrimSpace(r.Header.Get("X-Database-Access-Token"))
	if err := validateDatabaseAccessToken(accessToken, claims.UserID); err != nil {
		respondWithError(w, http.StatusForbidden, "Database access verification required")
		return
	}

	var users, activeProducts, suppliers, purchaseOrders, sales, lowStock, customers, pendingPayments int
	var inventoryValue float64
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&users)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM products WHERE status = 'active'").Scan(&activeProducts)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM suppliers WHERE status = 'active'").Scan(&suppliers)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM purchase_orders").Scan(&purchaseOrders)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM sales").Scan(&sales)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM customers WHERE status = 'active'").Scan(&customers)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM mpesa_payments WHERE status = 'pending'").Scan(&pendingPayments)
	_ = db.DB.QueryRow("SELECT COUNT(*) FROM products WHERE stock <= reorder_level AND status = 'active'").Scan(&lowStock)
	_ = db.DB.QueryRow("SELECT COALESCE(SUM(price * stock), 0) FROM products WHERE status = 'active'").Scan(&inventoryValue)

	tableDescriptions := map[string]string{
		"customers":            "Customer and patient profiles used at checkout and for repeat service.",
		"mpesa_payments":       "Audit trail for M-Pesa STK push requests and callbacks.",
		"prescriptions":        "Prescription details captured for regulated medicines.",
		"products":             "Medicine and retail product master records.",
		"purchase_order_items": "Line items that belong to a purchase order.",
		"purchase_orders":      "Restocking orders sent to suppliers.",
		"role_permissions":     "Dynamic permission matrix that decides which roles can access which features.",
		"sale_items":           "Line items sold within each POS transaction.",
		"sales":                "Completed or pending checkout records.",
		"suppliers":            "Supplier master records and contact details.",
		"users":                "Staff accounts that sign in to the system.",
	}

	tables := []map[string]interface{}{}
	tableRows, err := db.DB.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE' ORDER BY table_name ASC`)
	if err == nil {
		defer tableRows.Close()
		for tableRows.Next() {
			var tableName string
			if err := tableRows.Scan(&tableName); err != nil {
				continue
			}

			columns := []map[string]interface{}{}
			columnRows, colErr := db.DB.Query(`SELECT column_name, data_type, is_nullable
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = $1
				ORDER BY ordinal_position ASC`, tableName)
			if colErr == nil {
				for columnRows.Next() {
					var columnName, dataType, isNullable string
					if err := columnRows.Scan(&columnName, &dataType, &isNullable); err == nil {
						columns = append(columns, map[string]interface{}{
							"name":     columnName,
							"type":     dataType,
							"nullable": strings.EqualFold(isNullable, "YES"),
						})
					}
				}
				columnRows.Close()
			}

			quotedTable := strings.ReplaceAll(tableName, `"`, `""`)
			var rowCount int
			_ = db.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, quotedTable)).Scan(&rowCount)

			var sizePretty string
			_ = db.DB.QueryRow(`SELECT pg_size_pretty(pg_total_relation_size(('public.' || quote_ident($1))::regclass))`, tableName).Scan(&sizePretty)
			if strings.TrimSpace(sizePretty) == "" {
				sizePretty = "Unknown"
			}

			tables = append(tables, map[string]interface{}{
				"table_name":   tableName,
				"description":  tableDescriptions[tableName],
				"row_count":    rowCount,
				"column_count": len(columns),
				"size_pretty":  sizePretty,
				"columns":      columns,
			})
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"headline": "Simple system overview for non-technical users",
		"verified": true,
		"overview": []map[string]interface{}{
			{"label": "User accounts", "value": users, "description": "Total staff accounts that can access the system."},
			{"label": "Active products", "value": activeProducts, "description": "Products currently available in the catalog."},
			{"label": "Suppliers", "value": suppliers, "description": "Active suppliers your pharmacy can order from."},
			{"label": "Purchase orders", "value": purchaseOrders, "description": "Orders placed with suppliers for stock replenishment."},
			{"label": "Customer profiles", "value": customers, "description": "Saved customer or patient records available at checkout."},
			{"label": "Completed sales", "value": sales, "description": "Transactions successfully processed through the POS."},
			{"label": "Pending M-Pesa payments", "value": pendingPayments, "description": "Mobile-money checkouts still awaiting callback confirmation."},
			{"label": "Low stock alerts", "value": lowStock, "description": "Products that should be restocked soon."},
			{"label": "Estimated inventory value", "value": inventoryValue, "description": "Approximate retail value of current stock on hand."},
		},
		"tables": tables,
	})
}

func reportRange(r *http.Request) (time.Time, time.Time) {
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	if q := strings.TrimSpace(r.URL.Query().Get("from")); q != "" {
		if parsed, err := time.Parse(time.RFC3339, q); err == nil {
			from = parsed
		}
	}
	if q := strings.TrimSpace(r.URL.Query().Get("to")); q != "" {
		if parsed, err := time.Parse(time.RFC3339, q); err == nil {
			to = parsed
		}
	}
	return from, to
}
