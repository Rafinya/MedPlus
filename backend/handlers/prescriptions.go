package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"medplus-pos/db"
	"medplus-pos/models"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type prescriptionItemWriteRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type prescriptionWriteRequest struct {
	PrescriptionNumber string                         `json:"prescription_number"`
	CustomerID         *int                           `json:"customer_id,omitempty"`
	PatientName        string                         `json:"patient_name"`
	DoctorName         string                         `json:"doctor_name,omitempty"`
	Notes              string                         `json:"notes,omitempty"`
	Status             string                         `json:"status,omitempty"`
	Items              []prescriptionItemWriteRequest `json:"items"`
}

type profileUpdateRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password,omitempty"`
}

func validPrescriptionStatus(status string) bool {
	switch status {
	case "active", "dispensed", "void":
		return true
	default:
		return false
	}
}

func generatePrescriptionNumber() string {
	now := time.Now()
	return fmt.Sprintf("RX-%s-%04d", now.Format("20060102"), now.Nanosecond()%10000)
}

func userDisplayName(userID int, fallback string) string {
	var fullName string
	err := db.DB.QueryRow(`SELECT COALESCE(full_name, '') FROM users WHERE id = $1`, userID).Scan(&fullName)
	if err == nil && strings.TrimSpace(fullName) != "" {
		return strings.TrimSpace(fullName)
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return "Pharmacist"
}

func resolvePrescriptionRequest(claimsUserID int, claimsUsername string, req prescriptionWriteRequest) (prescriptionWriteRequest, error) {
	resolved := req
	resolved.PrescriptionNumber = strings.TrimSpace(resolved.PrescriptionNumber)
	resolved.PatientName = strings.TrimSpace(resolved.PatientName)
	resolved.DoctorName = strings.TrimSpace(resolved.DoctorName)
	resolved.Notes = strings.TrimSpace(resolved.Notes)
	resolved.Status = strings.TrimSpace(strings.ToLower(resolved.Status))

	if resolved.Status == "" {
		resolved.Status = "active"
	}
	if !validPrescriptionStatus(resolved.Status) {
		return resolved, fmt.Errorf("invalid prescription status")
	}
	if resolved.PrescriptionNumber == "" {
		resolved.PrescriptionNumber = generatePrescriptionNumber()
	}
	if resolved.CustomerID != nil {
		var customerName string
		err := db.DB.QueryRow(`SELECT full_name FROM customers WHERE id = $1 AND status = 'active'`, *resolved.CustomerID).Scan(&customerName)
		if err == sql.ErrNoRows {
			return resolved, fmt.Errorf("selected customer was not found or is inactive")
		}
		if err != nil {
			return resolved, fmt.Errorf("could not verify customer")
		}
		if resolved.PatientName == "" {
			resolved.PatientName = strings.TrimSpace(customerName)
		}
	}
	if resolved.PatientName == "" {
		return resolved, fmt.Errorf("patient_name is required")
	}
	if resolved.DoctorName == "" {
		resolved.DoctorName = userDisplayName(claimsUserID, claimsUsername)
	}
	if len(resolved.Items) == 0 {
		return resolved, fmt.Errorf("at least one prescription medicine is required")
	}
	return resolved, nil
}

func fetchPrescriptionItemsByIDs(ids []int) (map[int][]models.PrescriptionItem, error) {
	itemsByPrescription := map[int][]models.PrescriptionItem{}
	if len(ids) == 0 {
		return itemsByPrescription, nil
	}
	rows, err := db.DB.Query(`SELECT pi.id, pi.prescription_id, pi.product_id, p.name, COALESCE(p.barcode, ''), pi.quantity, p.rx, pi.created_at
		FROM prescription_items pi
		JOIN products p ON p.id = pi.product_id
		WHERE pi.prescription_id = ANY($1)
		ORDER BY pi.id ASC`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.PrescriptionItem
		if err := rows.Scan(&item.ID, &item.PrescriptionID, &item.ProductID, &item.ProductName, &item.ProductBarcode, &item.Quantity, &item.RequiresRx, &item.CreatedAt); err != nil {
			return nil, err
		}
		itemsByPrescription[item.PrescriptionID] = append(itemsByPrescription[item.PrescriptionID], item)
	}
	return itemsByPrescription, rows.Err()
}

func listPrescriptions(search, status string) ([]models.Prescription, error) {
	query := `SELECT p.id, p.prescription_number, p.patient_name, p.doctor_name,
		p.customer_id, COALESCE(c.full_name, ''), p.created_by, COALESCE(u.full_name, ''),
		COALESCE(p.notes, ''), COALESCE(p.status, 'active'), p.created_at
		FROM prescriptions p
		LEFT JOIN customers c ON c.id = p.customer_id
		LEFT JOIN users u ON u.id = p.created_by
		WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if search = strings.TrimSpace(search); search != "" {
		query += fmt.Sprintf(` AND (LOWER(p.prescription_number) LIKE LOWER($%d) OR LOWER(p.patient_name) LIKE LOWER($%d) OR LOWER(p.doctor_name) LIKE LOWER($%d) OR LOWER(COALESCE(c.full_name, '')) LIKE LOWER($%d))`, idx, idx, idx, idx)
		args = append(args, "%"+search+"%")
		idx++
	}
	if status = strings.TrimSpace(strings.ToLower(status)); status != "" && status != "all" {
		query += fmt.Sprintf(" AND COALESCE(p.status, 'active') = $%d", idx)
		args = append(args, status)
		idx++
	}
	query += " ORDER BY p.created_at DESC, p.id DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptions := []models.Prescription{}
	ids := []int{}
	for rows.Next() {
		var prescription models.Prescription
		var customerID sql.NullInt64
		var createdBy sql.NullInt64
		if err := rows.Scan(
			&prescription.ID,
			&prescription.PrescriptionNumber,
			&prescription.PatientName,
			&prescription.DoctorName,
			&customerID,
			&prescription.CustomerName,
			&createdBy,
			&prescription.CreatedByName,
			&prescription.Notes,
			&prescription.Status,
			&prescription.CreatedAt,
		); err != nil {
			return nil, err
		}
		if customerID.Valid {
			id := int(customerID.Int64)
			prescription.CustomerID = &id
		}
		if createdBy.Valid {
			id := int(createdBy.Int64)
			prescription.CreatedBy = &id
		}
		prescriptions = append(prescriptions, prescription)
		ids = append(ids, prescription.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	itemsByPrescription, err := fetchPrescriptionItemsByIDs(ids)
	if err != nil {
		return nil, err
	}
	for i := range prescriptions {
		prescriptions[i].Items = itemsByPrescription[prescriptions[i].ID]
	}
	return prescriptions, nil
}

func savePrescriptionItems(tx *sql.Tx, prescriptionID int, items []prescriptionItemWriteRequest) error {
	for _, item := range items {
		if item.ProductID <= 0 {
			return fmt.Errorf("invalid prescription product")
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("prescription medicine quantity must be greater than zero")
		}

		var requiresRx bool
		err := tx.QueryRow(`SELECT rx FROM products WHERE id = $1 AND status = 'active'`, item.ProductID).Scan(&requiresRx)
		if err == sql.ErrNoRows {
			return fmt.Errorf("selected prescription medicine was not found")
		}
		if err != nil {
			return err
		}
		if !requiresRx {
			return fmt.Errorf("only medicines marked as prescription-required can be attached to a prescription")
		}

		if _, err := tx.Exec(`INSERT INTO prescription_items (prescription_id, product_id, quantity)
			VALUES ($1, $2, $3)
			ON CONFLICT (prescription_id, product_id)
			DO UPDATE SET quantity = EXCLUDED.quantity`, prescriptionID, item.ProductID, item.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func PrescriptionsCollectionHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ListPrescriptionsHandler(w, r)
	case http.MethodPost:
		CreatePrescriptionHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func PrescriptionItemHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetPrescriptionByIDHandler(w, r)
	case http.MethodPut:
		UpdatePrescriptionHandler(w, r)
	default:
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func ListPrescriptionsHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	prescriptions, err := listPrescriptions(r.URL.Query().Get("search"), r.URL.Query().Get("status"))
	if err != nil {
		log.Printf("ListPrescriptions error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, prescriptions)
}

func GetPrescriptionByIDHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	id, err := parseIDFromPath("/api/prescriptions/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid prescription id")
		return
	}
	prescriptions, err := listPrescriptions("", "all")
	if err != nil {
		log.Printf("GetPrescriptionByID error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	for _, prescription := range prescriptions {
		if prescription.ID == id {
			respondWithJSON(w, http.StatusOK, prescription)
			return
		}
	}
	respondWithError(w, http.StatusNotFound, "Prescription not found")
}

func CreatePrescriptionHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "inventory.manage")
	if !ok {
		return
	}
	var req prescriptionWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	resolved, err := resolvePrescriptionRequest(claims.UserID, claims.Username, req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	var prescriptionID int
	err = tx.QueryRowContext(r.Context(), `INSERT INTO prescriptions (prescription_number, patient_name, doctor_name, customer_id, created_by, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		resolved.PrescriptionNumber,
		resolved.PatientName,
		resolved.DoctorName,
		resolved.CustomerID,
		claims.UserID,
		resolved.Notes,
		resolved.Status,
	).Scan(&prescriptionID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Prescription number already exists")
			return
		}
		log.Printf("CreatePrescription insert error: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err := savePrescriptionItems(tx, prescriptionID, resolved.Items); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	prescriptions, err := listPrescriptions(resolved.PrescriptionNumber, "all")
	if err != nil || len(prescriptions) == 0 {
		respondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"message":             "Prescription created successfully",
			"prescription_number": resolved.PrescriptionNumber,
		})
		return
	}
	respondWithJSON(w, http.StatusCreated, prescriptions[0])
}

func UpdatePrescriptionHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "inventory.manage")
	if !ok {
		return
	}
	id, err := parseIDFromPath("/api/prescriptions/", r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid prescription id")
		return
	}
	var req prescriptionWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	resolved, err := resolvePrescriptionRequest(claims.UserID, claims.Username, req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := db.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(r.Context(), `UPDATE prescriptions
		SET prescription_number = $1, patient_name = $2, doctor_name = $3, customer_id = $4, notes = $5, status = $6, created_by = COALESCE(created_by, $7)
		WHERE id = $8`,
		resolved.PrescriptionNumber,
		resolved.PatientName,
		resolved.DoctorName,
		resolved.CustomerID,
		resolved.Notes,
		resolved.Status,
		claims.UserID,
		id,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			respondWithError(w, http.StatusConflict, "Prescription number already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondWithError(w, http.StatusNotFound, "Prescription not found")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM prescription_items WHERE prescription_id = $1`, id); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err := savePrescriptionItems(tx, id, resolved.Items); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	prescriptions, err := listPrescriptions(resolved.PrescriptionNumber, "all")
	if err != nil {
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "Prescription updated successfully"})
		return
	}
	for _, prescription := range prescriptions {
		if prescription.ID == id {
			respondWithJSON(w, http.StatusOK, prescription)
			return
		}
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Prescription updated successfully"})
}

func SaveProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	claims, ok := currentUser(r)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	var req profileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.FullName = strings.TrimSpace(req.FullName)
	if req.Username == "" || req.FullName == "" {
		respondWithError(w, http.StatusBadRequest, "username and full_name are required")
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
		_, err = db.DB.Exec(`UPDATE users SET username = $1, full_name = $2, password_hash = $3 WHERE id = $4`, req.Username, req.FullName, string(hash), claims.UserID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				respondWithError(w, http.StatusConflict, "Username already exists")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	} else {
		_, err := db.DB.Exec(`UPDATE users SET username = $1, full_name = $2 WHERE id = $3`, req.Username, req.FullName, claims.UserID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				respondWithError(w, http.StatusConflict, "Username already exists")
				return
			}
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	}

	var user models.User
	err := db.DB.QueryRow(`SELECT id, username, full_name, role, status, created_at FROM users WHERE id = $1`, claims.UserID).
		Scan(&user.ID, &user.Username, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	user.Permissions, _ = db.PermissionsForRole(user.Role)
	respondWithJSON(w, http.StatusOK, user)
}
