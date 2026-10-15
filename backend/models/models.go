package models

import "time"

// User represents a system user
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Password     string    `json:"password,omitempty"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	Permissions  []string  `json:"permissions,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Product represents a medicine item in inventory
type Product struct {
	ID           int       `json:"id"`
	Barcode      string    `json:"barcode"`
	Name         string    `json:"name"`
	Brand        string    `json:"brand"`
	Price        float64   `json:"price"`
	Stock        int       `json:"stock"`
	Category     string    `json:"category"`
	Icon         string    `json:"icon"`
	Rx           bool      `json:"rx"`
	ExpiryDate   time.Time `json:"expiry_date"`
	ReorderLevel int       `json:"reorder_level"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// Supplier represents a product supplier
type Supplier struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	ContactPerson string    `json:"contact_person"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// PurchaseOrderItem represents a purchase order line
type PurchaseOrderItem struct {
	ID               int     `json:"id"`
	PurchaseOrderID  int     `json:"purchase_order_id"`
	ProductID        int     `json:"product_id"`
	ProductName      string  `json:"product_name,omitempty"`
	Quantity         int     `json:"quantity"`
	UnitCost         float64 `json:"unit_cost"`
	ReceivedQuantity int     `json:"received_quantity"`
}

// PurchaseOrder represents a supply-chain procurement record
type PurchaseOrder struct {
	ID                 int                 `json:"id"`
	PONumber           string              `json:"po_number"`
	SupplierID         int                 `json:"supplier_id"`
	SupplierName       string              `json:"supplier_name,omitempty"`
	Status             string              `json:"status"`
	Notes              string              `json:"notes"`
	OrderedBy          *int                `json:"ordered_by,omitempty"`
	ReceivedBy         *int                `json:"received_by,omitempty"`
	ExpectedDeliveryAt *time.Time          `json:"expected_delivery_at,omitempty"`
	OrderedAt          time.Time           `json:"ordered_at"`
	ReceivedAt         *time.Time          `json:"received_at,omitempty"`
	UpdatedAt          time.Time           `json:"updated_at"`
	ItemCount          int                 `json:"item_count,omitempty"`
	RequestedQuantity  int                 `json:"requested_quantity,omitempty"`
	ReceivedQuantity   int                 `json:"received_quantity,omitempty"`
	Items              []PurchaseOrderItem `json:"items,omitempty"`
}

// PrescriptionItem represents a medicine attached to a prescription
type PrescriptionItem struct {
	ID             int       `json:"id"`
	PrescriptionID int       `json:"prescription_id"`
	ProductID      int       `json:"product_id"`
	ProductName    string    `json:"product_name,omitempty"`
	ProductBarcode string    `json:"product_barcode,omitempty"`
	Quantity       int       `json:"quantity"`
	RequiresRx     bool      `json:"requires_rx,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Prescription represents prescription verification details
type Prescription struct {
	ID                 int                `json:"id"`
	PrescriptionNumber string             `json:"prescription_number"`
	PatientName        string             `json:"patient_name"`
	DoctorName         string             `json:"doctor_name"`
	CustomerID         *int               `json:"customer_id,omitempty"`
	CustomerName       string             `json:"customer_name,omitempty"`
	CreatedBy          *int               `json:"created_by,omitempty"`
	CreatedByName      string             `json:"created_by_name,omitempty"`
	Notes              string             `json:"notes,omitempty"`
	Status             string             `json:"status"`
	Items              []PrescriptionItem `json:"items,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
}

// Customer represents a pharmacy customer or patient profile
type Customer struct {
	ID            int       `json:"id"`
	FullName      string    `json:"full_name"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	LoyaltyNumber string    `json:"loyalty_number"`
	Notes         string    `json:"notes"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// Sale represents a POS transaction
type Sale struct {
	ID               int        `json:"id"`
	TransactionID    string     `json:"transaction_id"`
	Subtotal         float64    `json:"subtotal"`
	Tax              float64    `json:"tax"`
	Total            float64    `json:"total"`
	PaymentMethod    string     `json:"payment_method"`
	PaymentReference string     `json:"payment_reference,omitempty"`
	PaymentStatus    string     `json:"payment_status,omitempty"`
	CashierID        int        `json:"cashier_id"`
	CashierName      string     `json:"cashier_name,omitempty"`
	CustomerID       *int       `json:"customer_id,omitempty"`
	CustomerName     string     `json:"customer_name,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	Items            []SaleItem `json:"items,omitempty"`
}

// SaleItem represents an item within a sale
type SaleItem struct {
	ID             int           `json:"id"`
	SaleID         int           `json:"sale_id"`
	ProductID      int           `json:"product_id"`
	ProductName    string        `json:"product_name,omitempty"`
	ProductBarcode string        `json:"product_barcode,omitempty"`
	Quantity       int           `json:"quantity"`
	Price          float64       `json:"price"`
	PrescriptionID *int          `json:"prescription_id,omitempty"`
	Prescription   *Prescription `json:"prescription,omitempty"`
}

// LoginRequest defines credentials payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest defines user signup payload
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Status   string `json:"status,omitempty"`
}

// LoginResponse returns token and user info
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type PermissionDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type RolePermission struct {
	RoleName      string `json:"role_name"`
	PermissionKey string `json:"permission_key"`
	Allowed       bool   `json:"allowed"`
}

// MPesaPayment represents an M-Pesa STK payment request
type MPesaPayment struct {
	ID                  int       `json:"id"`
	CheckoutRequestID   string    `json:"checkout_request_id"`
	MerchantRequestID   string    `json:"merchant_request_id"`
	SaleReference       string    `json:"sale_reference,omitempty"`
	PhoneNumber         string    `json:"phone_number"`
	Amount              float64   `json:"amount"`
	AccountReference    string    `json:"account_reference"`
	TransactionDesc     string    `json:"transaction_desc"`
	Status              string    `json:"status"`
	ResponseCode        string    `json:"response_code,omitempty"`
	ResponseDescription string    `json:"response_description,omitempty"`
	CustomerMessage     string    `json:"customer_message,omitempty"`
	ReceiptNumber       string    `json:"receipt_number,omitempty"`
	ResultCode          string    `json:"result_code,omitempty"`
	ResultDesc          string    `json:"result_desc,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// DashboardStats holds summary metrics
type DashboardStats struct {
	TodaySales        float64 `json:"today_sales"`
	SalesChange       float64 `json:"sales_change"`
	TransactionCount  int     `json:"transaction_count"`
	TxChange          int     `json:"tx_change"`
	PrescriptionCount int     `json:"prescription_count"`
	RxChange          int     `json:"rx_change"`
	LowStockCount     int     `json:"low_stock_count"`
}
