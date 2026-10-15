package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"medplus-pos/db"
	"medplus-pos/models"
)

func CustomersCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListCustomersHandler(w, r)
	case http.MethodPost:
		CreateCustomerHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func CustomerItemHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetCustomerByIDHandler(w, r)
	case http.MethodPut:
		UpdateCustomerHandler(w, r)
	case http.MethodDelete:
		DeleteCustomerHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListCustomersHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "customers.view"); !ok {
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	query := `SELECT id, full_name, COALESCE(phone, ''), COALESCE(email, ''), COALESCE(loyalty_number, ''), COALESCE(notes, ''), status, created_at
		FROM customers`
	args := []interface{}{}
	idx := 1

	if search != "" {
		query += ` WHERE LOWER(full_name) LIKE LOWER($1) OR phone LIKE $1 OR LOWER(email) LIKE LOWER($1) OR LOWER(COALESCE(loyalty_number, '')) LIKE LOWER($1)`
		args = append(args, "%"+search+"%")
		idx++
	}

	if idx == 1 {
		query += ` WHERE status = 'active'`
	} else {
		query += ` AND status = 'active'`
	}
	query += ` ORDER BY full_name ASC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("ListCustomers query error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	customers := []models.Customer{}
	for rows.Next() {
		var customer models.Customer
		if err := rows.Scan(&customer.ID, &customer.FullName, &customer.Phone, &customer.Email, &customer.LoyaltyNumber, &customer.Notes, &customer.Status, &customer.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		customers = append(customers, customer)
	}
	respondWithJSON(w, http.StatusOK, customers)
}

func GetCustomerByIDHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "customers.view"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/customers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid customer id")
		return
	}

	var customer models.Customer
	err = db.DB.QueryRow(`SELECT id, full_name, COALESCE(phone, ''), COALESCE(email, ''), COALESCE(loyalty_number, ''), COALESCE(notes, ''), status, created_at
		FROM customers WHERE id = $1`, id).
		Scan(&customer.ID, &customer.FullName, &customer.Phone, &customer.Email, &customer.LoyaltyNumber, &customer.Notes, &customer.Status, &customer.CreatedAt)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Customer not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusOK, customer)
}

func CreateCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "customers.manage"); !ok {
		return
	}
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(customer.FullName) == "" {
		respondWithError(w, http.StatusBadRequest, "full_name is required")
		return
	}
	if strings.TrimSpace(customer.Status) == "" {
		customer.Status = "active"
	}

	var created models.Customer
	err := db.DB.QueryRow(`INSERT INTO customers (full_name, phone, email, loyalty_number, notes, status)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
		RETURNING id, full_name, COALESCE(phone, ''), COALESCE(email, ''), COALESCE(loyalty_number, ''), COALESCE(notes, ''), status, created_at`,
		strings.TrimSpace(customer.FullName), strings.TrimSpace(customer.Phone), strings.TrimSpace(customer.Email), strings.TrimSpace(customer.LoyaltyNumber), strings.TrimSpace(customer.Notes), customer.Status,
	).Scan(&created.ID, &created.FullName, &created.Phone, &created.Email, &created.LoyaltyNumber, &created.Notes, &created.Status, &created.CreatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Loyalty number already exists")
			return
		}
		log.Printf("CreateCustomer error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusCreated, created)
}

func UpdateCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "customers.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/customers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid customer id")
		return
	}

	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(customer.FullName) == "" {
		respondWithError(w, http.StatusBadRequest, "full_name is required")
		return
	}
	if strings.TrimSpace(customer.Status) == "" {
		customer.Status = "active"
	}

	result, err := db.DB.Exec(`UPDATE customers
		SET full_name = $1, phone = $2, email = $3, loyalty_number = NULLIF($4, ''), notes = $5, status = $6
		WHERE id = $7`,
		strings.TrimSpace(customer.FullName), strings.TrimSpace(customer.Phone), strings.TrimSpace(customer.Email), strings.TrimSpace(customer.LoyaltyNumber), strings.TrimSpace(customer.Notes), customer.Status, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Loyalty number already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "Customer not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Customer updated successfully"})
}

func DeleteCustomerHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "customers.manage"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/customers/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid customer id")
		return
	}

	result, err := db.DB.Exec("UPDATE customers SET status = 'inactive' WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "Customer not found")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Customer archived successfully"})
}
