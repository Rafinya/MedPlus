package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	loadEnv()
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "medplus_pos")
	sslmode := getEnv("DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, dbname, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Ping failed: %v", err)
	}

	fmt.Println("==========================================================")
	fmt.Println("   MEDPLUS PHARMACY POS - LIVE POSTGRESQL DATABASE CLIENT  ")
	fmt.Println("==========================================================")
	fmt.Printf("Connected to Database: '%s' on %s:%s\n\n", dbname, host, port)

	tables := []string{
		"users",
		"products",
		"customers",
		"sales",
		"sale_items",
		"prescriptions",
		"suppliers",
		"purchase_orders",
		"mpesa_payments",
	}

	for _, table := range tables {
		fmt.Printf("=== TABLE: %s ===\n", strings.ToUpper(table))
		rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 10", table))
		if err != nil {
			fmt.Printf("  Error querying table %s: %v\n\n", table, err)
			continue
		}
		cols, _ := rows.Columns()
		fmt.Printf("Columns (%d): %s\n", len(cols), strings.Join(cols, " | "))

		count := 0
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range cols {
			valuePtrs[i] = &values[i]
		}

		for rows.Next() {
			count++
			rows.Scan(valuePtrs...)
			rowStr := []string{}
			for _, val := range values {
				if val == nil {
					rowStr = append(rowStr, "NULL")
				} else if b, ok := val.([]byte); ok {
					rowStr = append(rowStr, string(b))
				} else {
					rowStr = append(rowStr, fmt.Sprintf("%v", val))
				}
			}
			fmt.Printf("Row %d: %s\n", count, strings.Join(rowStr, " | "))
		}
		rows.Close()
		if count == 0 {
			fmt.Println("  (Table is empty)")
		}
		fmt.Println()
	}
}

func loadEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	envPath := filepath.Join(dir, ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
