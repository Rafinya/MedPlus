package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"medplus-pos/models"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var schemaSQL string

// DB holds the active database connection
var DB *sql.DB

// InitDB initializes connection to PostgreSQL, migrates tables and optionally seeds demo data.
func InitDB(connStr string) (*sql.DB, error) {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening db connection: %w", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxIdleTime(5 * time.Minute)
	DB.SetConnMaxLifetime(30 * time.Minute)

	if err = DB.Ping(); err != nil {
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database.")

	if _, err = DB.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("error applying database schema: %w", err)
	}
	log.Println("Database schema checked and tables verified.")

	// Migration: update existing product prices to realistic rounded Ksh values
	_, _ = DB.Exec(`
		UPDATE products 
		SET price = ROUND(
			CASE 
				WHEN barcode = '890123456701' THEN 450
				WHEN barcode = '890123456702' THEN 600
				WHEN barcode = '890123456703' THEN 120
				WHEN barcode = '890123456704' THEN 80
				WHEN barcode = '890123456705' THEN 850
				WHEN barcode = '890123456706' THEN 1200
				WHEN barcode = '890123456707' THEN 150
				WHEN barcode = '890123456708' THEN 3500
				WHEN barcode = '890123456709' THEN 350
				WHEN barcode = '890123456710' THEN 1100
				WHEN barcode = '890123456711' THEN 250
				WHEN barcode = '890123456712' THEN 400
				WHEN barcode = '890123456713' THEN 1800
				WHEN barcode = '890123456714' THEN 100
				WHEN barcode = '890123456715' THEN 500
				WHEN barcode = '890123456716' THEN 150
				WHEN barcode = '890123456717' THEN 300
				WHEN barcode = '890123456718' THEN 2200
				WHEN barcode = '890123456719' THEN 5500
				WHEN barcode = '890123456720' THEN 4500
				ELSE price
			END
		);
		UPDATE products SET price = ROUND(price);
	`)

	if shouldSeedDemoData() {
		if err := seedDefaultData(); err != nil {
			return nil, fmt.Errorf("error seeding default data: %w", err)
		}
	} else {
		log.Println("Demo data seeding is disabled.")
	}

	if err := SeedDefaultRolePermissions(); err != nil {
		return nil, fmt.Errorf("error seeding role permissions: %w", err)
	}

	return DB, nil
}

func shouldSeedDemoData() bool {
	flag := strings.TrimSpace(strings.ToLower(os.Getenv("SEED_DEMO_DATA")))
	if flag == "true" || flag == "1" || flag == "yes" {
		return true
	}
	if flag == "false" || flag == "0" || flag == "no" {
		return false
	}
	return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) != "production"
}

// seedDefaultData inserts initial users and products if they don't exist
func seedDefaultData() error {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Seeding demo users...")
		usersToSeed := []models.User{
			{Username: "admin", FullName: "System Administrator", Role: "Admin", Password: "adminpassword"},
			{Username: "pharmacist", FullName: "Dr. Jane Doe", Role: "Pharmacist", Password: "pharmpassword"},
		}

		for _, u := range usersToSeed {
			hashedBytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			_, err = DB.Exec(
				"INSERT INTO users (username, password_hash, full_name, role, status) VALUES ($1, $2, $3, $4, 'active')",
				u.Username, string(hashedBytes), u.FullName, u.Role,
			)
			if err != nil {
				return err
			}
		}
		log.Println("Demo users seeded successfully.")
	}

	err = DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		log.Println("Seeding demo medicines and products...")
		productsToSeed := []models.Product{
			{Barcode: "890123456701", Name: "Amoxicillin 500mg", Brand: "Generic", Price: 450, Stock: 45, Category: "prescription", Icon: "💊", Rx: true, ReorderLevel: 10},
			{Barcode: "890123456702", Name: "Lisinopril 10mg", Brand: "Prinivil", Price: 600, Stock: 32, Category: "prescription", Icon: "💊", Rx: true, ReorderLevel: 10},
			{Barcode: "890123456703", Name: "Ibuprofen 200mg", Brand: "Advil", Price: 120, Stock: 120, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 15},
			{Barcode: "890123456704", Name: "Acetaminophen 500mg", Brand: "Tylenol", Price: 80, Stock: 95, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 15},
			{Barcode: "890123456705", Name: "Vitamin D3 1000IU", Brand: "Nature Made", Price: 850, Stock: 60, Category: "vitamins", Icon: "🍊", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456706", Name: "Multivitamin", Brand: "Centrum", Price: 1200, Stock: 40, Category: "vitamins", Icon: "🍊", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456707", Name: "Band-Aid Assorted", Brand: "Johnson & Johnson", Price: 150, Stock: 8, Category: "medical", Icon: "🩹", Rx: false, ReorderLevel: 5},
			{Barcode: "890123456708", Name: "Digital Thermometer", Brand: "Omron", Price: 3500, Stock: 15, Category: "medical", Icon: "🌡️", Rx: false, ReorderLevel: 5},
			{Barcode: "890123456709", Name: "Metformin 500mg", Brand: "Glucophage", Price: 350, Stock: 3, Category: "prescription", Icon: "💊", Rx: true, ReorderLevel: 5},
			{Barcode: "890123456710", Name: "Atorvastatin 20mg", Brand: "Lipitor", Price: 1100, Stock: 28, Category: "prescription", Icon: "💊", Rx: true, ReorderLevel: 10},
			{Barcode: "890123456711", Name: "Cetirizine 10mg", Brand: "Zyrtec", Price: 250, Stock: 50, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456712", Name: "Omeprazole 20mg", Brand: "Prilosec", Price: 400, Stock: 42, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456713", Name: "Fish Oil 1000mg", Brand: "Nordic Naturals", Price: 1800, Stock: 35, Category: "vitamins", Icon: "🐟", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456714", Name: "Hand Sanitizer", Brand: "Purell", Price: 100, Stock: 2, Category: "personal", Icon: "🧴", Rx: false, ReorderLevel: 5},
			{Barcode: "890123456715", Name: "Face Masks (50pk)", Brand: "Medline", Price: 500, Stock: 25, Category: "medical", Icon: "😷", Rx: false, ReorderLevel: 8},
			{Barcode: "890123456716", Name: "Aspirin 81mg", Brand: "Bayer", Price: 150, Stock: 78, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 15},
			{Barcode: "890123456717", Name: "Loratadine 10mg", Brand: "Claritin", Price: 300, Stock: 55, Category: "otc", Icon: "💊", Rx: false, ReorderLevel: 10},
			{Barcode: "890123456718", Name: "Probiotic Complex", Brand: "Culturelle", Price: 2200, Stock: 20, Category: "vitamins", Icon: "🦠", Rx: false, ReorderLevel: 8},
			{Barcode: "890123456719", Name: "Blood Pressure Monitor", Brand: "Omron", Price: 5500, Stock: 12, Category: "medical", Icon: "❤️", Rx: false, ReorderLevel: 4},
			{Barcode: "890123456720", Name: "Insulin Glargine", Brand: "Lantus", Price: 4500, Stock: 15, Category: "prescription", Icon: "💉", Rx: true, ReorderLevel: 5},
		}

		now := time.Now()
		for i, p := range productsToSeed {
			var expDate time.Time
			if p.Stock <= 5 {
				expDate = now.AddDate(0, 0, 10+i)
			} else {
				expDate = now.AddDate(1+i%2, i%12, 1)
			}

			_, err = DB.Exec(
				`INSERT INTO products (barcode, name, brand, price, stock, category, icon, rx, expiry_date, reorder_level, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active')`,
				p.Barcode, p.Name, p.Brand, p.Price, p.Stock, p.Category, p.Icon, p.Rx, expDate, p.ReorderLevel,
			)
			if err != nil {
				return err
			}
		}
		log.Println("Demo medicines seeded successfully.")
	}

	return nil
}
