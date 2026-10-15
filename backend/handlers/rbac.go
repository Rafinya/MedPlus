package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"medplus-pos/db"
	"medplus-pos/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type databaseAccessClaims struct {
	UserID  int    `json:"user_id"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

type databaseAccessRequest struct {
	Password string `json:"password"`
}

type rolePermissionsPayload struct {
	Permissions map[string]bool `json:"permissions"`
}

func VerifyDatabaseAccessHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := requirePermission(w, r, "database.view")
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req databaseAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		respondWithError(w, http.StatusBadRequest, "password is required")
		return
	}

	var hash string
	err := db.DB.QueryRow(`SELECT password_hash FROM users WHERE id = $1`, claims.UserID).Scan(&hash)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusUnauthorized, "User not found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect password")
		return
	}

	token, err := issueDatabaseAccessToken(claims.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not issue access token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"token":              token,
		"expires_in_seconds": 600,
	})
}

func RBACCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "rbac.manage"); !ok {
		return
	}
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	roles, err := db.Roles()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	matrix, err := db.RBACMatrix()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"roles":       roles,
		"permissions": db.PermissionCatalog,
		"matrix":      matrix,
	})
}

func RBACRoleHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "rbac.manage"); !ok {
		return
	}
	if r.Method != http.MethodPut {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	roleName, err := parseRoleName(r.URL.Path)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload rolePermissionsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if payload.Permissions == nil {
		payload.Permissions = map[string]bool{}
	}
	if err := db.UpsertRolePermissions(roleName, payload.Permissions); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update role permissions")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Role permissions updated successfully", "role_name": roleName})
}

func issueDatabaseAccessToken(userID int) (string, error) {
	claims := databaseAccessClaims{
		UserID:  userID,
		Purpose: "database-access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JWTSecret)
}

func validateDatabaseAccessToken(tokenStr string, userID int) error {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return fmt.Errorf("missing access token")
	}
	claims := &databaseAccessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return middleware.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("invalid access token")
	}
	if claims.Purpose != "database-access" || claims.UserID != userID {
		return fmt.Errorf("invalid access token")
	}
	return nil
}

func parseRoleName(path string) (string, error) {
	roleName := strings.Trim(strings.TrimPrefix(path, "/api/rbac/roles/"), "/")
	if roleName == "" {
		return "", fmt.Errorf("missing role name")
	}
	decoded, err := url.PathUnescape(roleName)
	if err != nil {
		return "", fmt.Errorf("invalid role name")
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", fmt.Errorf("invalid role name")
	}
	return decoded, nil
}
