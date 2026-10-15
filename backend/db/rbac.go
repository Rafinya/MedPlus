package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"medplus-pos/models"
)

var PermissionCatalog = []models.PermissionDefinition{
	{Key: "pos.access", Label: "POS Access", Description: "Open the point-of-sale screen and perform checkout work.", Category: "Operations"},
	{Key: "inventory.view", Label: "View Inventory", Description: "View stock levels and inventory records.", Category: "Inventory"},
	{Key: "inventory.manage", Label: "Manage Inventory", Description: "Create, update, and archive products.", Category: "Inventory"},
	{Key: "customers.view", Label: "View Customers", Description: "Review customer and patient profiles.", Category: "Customers"},
	{Key: "customers.manage", Label: "Manage Customers", Description: "Create, update, and archive customer profiles.", Category: "Customers"},
	{Key: "dashboard.view", Label: "View Dashboard", Description: "See operational summary metrics.", Category: "Reporting"},
	{Key: "reports.view", Label: "View Reports", Description: "Open reporting summaries and exports.", Category: "Reporting"},
	{Key: "analytics.view", Label: "View Analytics", Description: "Open analytical trend views.", Category: "Reporting"},
	{Key: "database.view", Label: "View Database Overview", Description: "Open the database overview after password verification.", Category: "Administration"},
	{Key: "users.manage", Label: "Manage Users", Description: "Create, edit, and deactivate system users.", Category: "Administration"},
	{Key: "rbac.manage", Label: "Manage RBAC", Description: "Configure role-based access permissions.", Category: "Administration"},
	{Key: "supply.view", Label: "View Supply Chain", Description: "See suppliers and purchase orders.", Category: "Supply Chain"},
	{Key: "supply.manage", Label: "Manage Supply Chain", Description: "Create suppliers, create purchase orders, and receive stock.", Category: "Supply Chain"},
	{Key: "settings.view", Label: "View Settings", Description: "Open system configuration and integration status pages.", Category: "Administration"},
}

func SeedDefaultRolePermissions() error {
	// Clean up and migrate any existing Cashier records to Pharmacist
	if _, err := DB.Exec(`UPDATE users SET role = 'Pharmacist' WHERE role = 'Cashier'`); err != nil {
		return err
	}
	if _, err := DB.Exec(`DELETE FROM role_permissions WHERE role_name = 'Cashier'`); err != nil {
		return err
	}

	allPermissions := []string{
		"pos.access", "inventory.view", "inventory.manage", "customers.view", "customers.manage",
		"dashboard.view", "reports.view", "analytics.view", "database.view", "users.manage",
		"rbac.manage", "supply.view", "supply.manage", "settings.view",
	}

	for _, roleName := range []string{"Admin", "Pharmacist"} {
		for _, permission := range allPermissions {
			if _, err := DB.Exec(`INSERT INTO role_permissions (role_name, permission_key, allowed, updated_at)
				VALUES ($1, $2, TRUE, $3)
				ON CONFLICT (role_name, permission_key) DO UPDATE SET allowed = TRUE`, roleName, permission, time.Now()); err != nil {
				return err
			}
		}
	}
	return nil
}

func HasPermission(roleName, permissionKey string) (bool, error) {
	roleName = strings.TrimSpace(roleName)
	permissionKey = strings.TrimSpace(permissionKey)
	if roleName == "" || permissionKey == "" {
		return false, nil
	}
	if strings.EqualFold(roleName, "Admin") || strings.EqualFold(roleName, "Pharmacist") {
		return true, nil
	}
	var allowed bool
	err := DB.QueryRow(`SELECT allowed FROM role_permissions WHERE role_name = $1 AND permission_key = $2`, roleName, permissionKey).Scan(&allowed)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func PermissionsForRole(roleName string) ([]string, error) {
	rows, err := DB.Query(`SELECT permission_key FROM role_permissions WHERE role_name = $1 AND allowed = TRUE ORDER BY permission_key ASC`, strings.TrimSpace(roleName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		permissions = append(permissions, key)
	}
	return permissions, nil
}

func Roles() ([]string, error) {
	roleSet := map[string]struct{}{}
	rows, err := DB.Query(`SELECT DISTINCT role_name FROM role_permissions
		UNION
		SELECT DISTINCT role FROM users
		ORDER BY 1 ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, err
		}
		roleName = strings.TrimSpace(roleName)
		if roleName != "" {
			roleSet[roleName] = struct{}{}
		}
	}

	roles := make([]string, 0, len(roleSet))
	for roleName := range roleSet {
		roles = append(roles, roleName)
	}
	sort.Strings(roles)
	return roles, nil
}

func RBACMatrix() (map[string]map[string]bool, error) {
	roles, err := Roles()
	if err != nil {
		return nil, err
	}
	matrix := map[string]map[string]bool{}
	for _, roleName := range roles {
		matrix[roleName] = map[string]bool{}
	}
	for _, permission := range PermissionCatalog {
		for _, roleName := range roles {
			matrix[roleName][permission.Key] = false
		}
	}

	rows, err := DB.Query(`SELECT role_name, permission_key, allowed FROM role_permissions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var roleName, permissionKey string
		var allowed bool
		if err := rows.Scan(&roleName, &permissionKey, &allowed); err != nil {
			return nil, err
		}
		if _, ok := matrix[roleName]; !ok {
			matrix[roleName] = map[string]bool{}
		}
		matrix[roleName][permissionKey] = allowed
	}
	return matrix, nil
}

func UpsertRolePermissions(roleName string, permissions map[string]bool) error {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return fmt.Errorf("role_name is required")
	}
	if _, err := DB.Exec(`UPDATE role_permissions SET updated_at = updated_at WHERE role_name = $1`, roleName); err != nil {
		return err
	}
	for _, permission := range PermissionCatalog {
		allowed := permissions[permission.Key]
		if _, err := DB.Exec(`INSERT INTO role_permissions (role_name, permission_key, allowed, updated_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (role_name, permission_key)
			DO UPDATE SET allowed = EXCLUDED.allowed, updated_at = EXCLUDED.updated_at`, roleName, permission.Key, allowed, time.Now()); err != nil {
			return err
		}
	}
	return nil
}
