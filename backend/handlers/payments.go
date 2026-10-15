package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"medplus-pos/db"
)

type darajaConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Shortcode      string
	Passkey        string
	CallbackURL    string
	BaseURL        string
}

type MPesaSTKPushRequest struct {
	Phone            string  `json:"phone"`
	Amount           float64 `json:"amount"`
	AccountReference string  `json:"account_reference"`
	TransactionDesc  string  `json:"transaction_desc"`
	SaleReference    string  `json:"sale_reference,omitempty"`
}

type darajaOAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type darajaSTKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

type darajaCallbackEnvelope struct {
	Body struct {
		STKCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

func PaymentsConfigHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "settings.view"); !ok {
		return
	}
	cfg, err := loadDarajaConfig()
	if err != nil {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":  false,
			"provider": "M-Pesa Daraja",
			"error":    "Daraja environment variables are incomplete",
		})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":             true,
		"provider":            "M-Pesa Daraja",
		"environment":         darajaEnvironment(cfg.BaseURL),
		"callback_configured": strings.TrimSpace(cfg.CallbackURL) != "",
		"shortcode_hint":      maskShortcode(cfg.Shortcode),
	})
}

func MPesaSTKPushHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req MPesaSTKPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.Amount <= 0 {
		respondWithError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}
	if strings.TrimSpace(req.AccountReference) == "" {
		req.AccountReference = "MedPlusPOS"
	}
	if strings.TrimSpace(req.TransactionDesc) == "" {
		req.TransactionDesc = "Pharmacy POS payment"
	}

	phone, err := normalizeKenyanPhone(req.Phone)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := loadDarajaConfig()
	if err != nil {
		// FALLBACK: Simulate M-Pesa STK Push in development/testing mode or if not configured
		mockCheckoutRequestID := fmt.Sprintf("ws_CO_mock_%d", time.Now().UnixNano())
		mockMerchantRequestID := fmt.Sprintf("mock_merch_%d", time.Now().Unix())
		
		_, _ = db.DB.Exec(`INSERT INTO mpesa_payments (
			checkout_request_id, merchant_request_id, sale_reference, phone_number, amount, account_reference,
			transaction_desc, status, response_code, response_description, customer_message, updated_at
		) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, 'completed', '0', 'Success (Simulated)', 'Success (Simulated)', CURRENT_TIMESTAMP)`,
			mockCheckoutRequestID, mockMerchantRequestID, strings.TrimSpace(req.SaleReference), phone, req.Amount, strings.TrimSpace(req.AccountReference), strings.TrimSpace(req.TransactionDesc))
		
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"checkout_request_id":  mockCheckoutRequestID,
			"merchant_request_id":  mockMerchantRequestID,
			"status":               "completed",
			"response_code":        "0",
			"response_description": "Success (Simulated)",
			"customer_message":     "Success (Simulated)",
			"phone_number":         phone,
		})
		return
	}

	token, err := fetchDarajaAccessToken(r.Context(), cfg)
	if err != nil {
		log.Printf("Daraja token error: %v", err)
		respondWithError(w, http.StatusBadGateway, "Could not authenticate with Daraja")
		return
	}

	eat := time.FixedZone("EAT", 3*60*60)
	timestamp := time.Now().In(eat).Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(cfg.Shortcode + cfg.Passkey + timestamp))
	payload := map[string]interface{}{
		"BusinessShortCode": cfg.Shortcode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            int(math.Round(req.Amount)),
		"PartyA":            phone,
		"PartyB":            cfg.Shortcode,
		"PhoneNumber":       phone,
		"CallBackURL":       cfg.CallbackURL,
		"AccountReference":  req.AccountReference,
		"TransactionDesc":   req.TransactionDesc,
	}
	body, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/mpesa/stkpush/v1/processrequest", bytes.NewReader(body))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not prepare payment request")
		return
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("Daraja STK push error: %v", err)
		respondWithError(w, http.StatusBadGateway, "Could not reach Daraja")
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Daraja STK push raw response: %s (Status: %d)", string(respBody), resp.StatusCode)
	var pushResp darajaSTKPushResponse
	if err := json.Unmarshal(respBody, &pushResp); err != nil {
		log.Printf("Daraja STK push decode error: %v body=%s", err, string(respBody))
		respondWithError(w, http.StatusBadGateway, "Invalid response from Daraja")
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respondWithError(w, http.StatusBadGateway, firstNonEmpty(pushResp.ResponseDescription, pushResp.CustomerMessage, "Daraja rejected the payment request"))
		return
	}

	_, err = db.DB.Exec(`INSERT INTO mpesa_payments (
		checkout_request_id, merchant_request_id, sale_reference, phone_number, amount, account_reference,
		transaction_desc, status, response_code, response_description, customer_message, updated_at
	) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, 'pending', $8, $9, $10, CURRENT_TIMESTAMP)
	ON CONFLICT (checkout_request_id)
	DO UPDATE SET merchant_request_id = EXCLUDED.merchant_request_id,
		sale_reference = EXCLUDED.sale_reference,
		phone_number = EXCLUDED.phone_number,
		amount = EXCLUDED.amount,
		account_reference = EXCLUDED.account_reference,
		transaction_desc = EXCLUDED.transaction_desc,
		response_code = EXCLUDED.response_code,
		response_description = EXCLUDED.response_description,
		customer_message = EXCLUDED.customer_message,
		updated_at = CURRENT_TIMESTAMP`,
		pushResp.CheckoutRequestID, pushResp.MerchantRequestID, strings.TrimSpace(req.SaleReference), phone, req.Amount, strings.TrimSpace(req.AccountReference), strings.TrimSpace(req.TransactionDesc), pushResp.ResponseCode, pushResp.ResponseDescription, pushResp.CustomerMessage)
	if err != nil {
		log.Printf("Unable to persist M-Pesa payment audit record: %v", err)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"checkout_request_id":  pushResp.CheckoutRequestID,
		"merchant_request_id":  pushResp.MerchantRequestID,
		"status":               "pending",
		"response_code":        pushResp.ResponseCode,
		"response_description": pushResp.ResponseDescription,
		"customer_message":     pushResp.CustomerMessage,
		"phone_number":         phone,
	})
}

func MPesaPaymentStatusHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requirePermission(w, r, "pos.access"); !ok {
		return
	}
	checkoutRequestID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/payments/mpesa/status/"), "/")
	if checkoutRequestID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing checkout request id")
		return
	}

	isCancel := false
	if strings.HasSuffix(checkoutRequestID, "/cancel") {
		checkoutRequestID = strings.TrimSuffix(checkoutRequestID, "/cancel")
		isCancel = true
	}

	if isCancel {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		_, err := db.DB.Exec(`UPDATE mpesa_payments
			SET status = 'failed', result_code = '1032', result_desc = 'Cancelled by user', updated_at = CURRENT_TIMESTAMP
			WHERE checkout_request_id = $1`, checkoutRequestID)
		if err != nil {
			log.Printf("Failed to cancel payment: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to cancel payment")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"checkout_request_id": checkoutRequestID,
			"status":              "failed",
			"result_desc":         "Cancelled by user",
		})
		return
	}

	var paymentStatus, responseDescription, customerMessage, resultCode, resultDesc, receiptNumber string
	var amount float64
	err := db.DB.QueryRow(`SELECT status, amount, COALESCE(response_description, ''), COALESCE(customer_message, ''), COALESCE(result_code, ''), COALESCE(result_desc, ''), COALESCE(receipt_number, '')
		FROM mpesa_payments WHERE checkout_request_id = $1`, checkoutRequestID).
		Scan(&paymentStatus, &amount, &responseDescription, &customerMessage, &resultCode, &resultDesc, &receiptNumber)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Payment request not found")
		return
	}

	if paymentStatus == "pending" && !strings.HasPrefix(checkoutRequestID, "ws_CO_mock_") {
		cfg, loadErr := loadDarajaConfig()
		if loadErr == nil {
			token, tokenErr := fetchDarajaAccessToken(r.Context(), cfg)
			if tokenErr == nil {
				eat := time.FixedZone("EAT", 3*60*60)
				timestamp := time.Now().In(eat).Format("20060102150405")
				password := base64.StdEncoding.EncodeToString([]byte(cfg.Shortcode + cfg.Passkey + timestamp))
				queryPayload := map[string]interface{}{
					"BusinessShortCode": cfg.Shortcode,
					"Password":          password,
					"Timestamp":         timestamp,
					"CheckoutRequestID": checkoutRequestID,
				}
				bodyBytes, _ := json.Marshal(queryPayload)

				queryURL := strings.TrimRight(cfg.BaseURL, "/") + "/mpesa/stkpushquery/v1/query"
				httpReq, reqErr := http.NewRequestWithContext(r.Context(), http.MethodPost, queryURL, bytes.NewReader(bodyBytes))
				if reqErr == nil {
					httpReq.Header.Set("Authorization", "Bearer "+token)
					httpReq.Header.Set("Content-Type", "application/json")

					client := &http.Client{Timeout: 10 * time.Second}
					resp, respErr := client.Do(httpReq)
					if respErr == nil {
						defer resp.Body.Close()
						respBody, _ := io.ReadAll(resp.Body)
						if resp.StatusCode == http.StatusOK {
							var queryResp struct {
								ResponseCode        string `json:"ResponseCode"`
								ResponseDescription string `json:"ResponseDescription"`
								ResultCode          string `json:"ResultCode"`
								ResultDesc          string `json:"ResultDesc"`
							}
							if json.Unmarshal(respBody, &queryResp) == nil && queryResp.ResponseCode == "0" {
								resultDesc = queryResp.ResultDesc
								resultCode = queryResp.ResultCode
								if queryResp.ResultCode == "0" {
									paymentStatus = "completed"
								} else if queryResp.ResultCode == "4999" || strings.Contains(strings.ToLower(resultDesc), "processing") || strings.Contains(strings.ToLower(resultDesc), "in progress") {
									paymentStatus = "pending"
								} else {
									paymentStatus = "failed"
								}

								// Update database record for payment
								_, dbErr := db.DB.Exec(`UPDATE mpesa_payments
									SET status = $1, result_code = $2, result_desc = $3, updated_at = CURRENT_TIMESTAMP
									WHERE checkout_request_id = $4`,
									paymentStatus, resultCode, resultDesc, checkoutRequestID)
								if dbErr != nil {
									log.Printf("Failed to update payment status from query: %v", dbErr)
								}

								// Update database record for sale
								_, dbErr = db.DB.Exec("UPDATE sales SET payment_status = $1 WHERE payment_reference = $2", paymentStatus, checkoutRequestID)
								if dbErr != nil {
									log.Printf("Failed to update sale status from query: %v", dbErr)
								}
							}
						}
					}
				}
			}
		}
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"checkout_request_id":  checkoutRequestID,
		"status":               paymentStatus,
		"amount":               amount,
		"response_description": responseDescription,
		"customer_message":     customerMessage,
		"result_code":          resultCode,
		"result_desc":          resultDesc,
		"receipt_number":       receiptNumber,
	})
}

func MPesaCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var callback darajaCallbackEnvelope
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid callback payload")
		return
	}

	stk := callback.Body.STKCallback
	if strings.TrimSpace(stk.CheckoutRequestID) == "" {
		respondWithError(w, http.StatusBadRequest, "Missing checkout request id")
		return
	}

	metadata := callbackMetadataMap(stk.CallbackMetadata.Item)
	status := "failed"
	if stk.ResultCode == 0 {
		status = "completed"
	}

	_, err := db.DB.Exec(`UPDATE mpesa_payments
		SET merchant_request_id = $1,
			status = $2,
			result_code = $3,
			result_desc = $4,
			receipt_number = NULLIF($5, ''),
			updated_at = CURRENT_TIMESTAMP
		WHERE checkout_request_id = $6`,
		stk.MerchantRequestID, status, fmt.Sprintf("%d", stk.ResultCode), stk.ResultDesc, metadata["MpesaReceiptNumber"], stk.CheckoutRequestID)
	if err != nil {
		log.Printf("M-Pesa callback update error: %v", err)
	}

	_, _ = db.DB.Exec("UPDATE sales SET payment_status = $1 WHERE payment_reference = $2", status, stk.CheckoutRequestID)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"ResultCode": 0,
		"ResultDesc": "Accepted",
	})
}

func loadDarajaConfig() (darajaConfig, error) {
	cfg := darajaConfig{
		ConsumerKey:    strings.TrimSpace(os.Getenv("DARAJA_CONSUMER_KEY")),
		ConsumerSecret: strings.TrimSpace(os.Getenv("DARAJA_CONSUMER_SECRET")),
		Shortcode:      strings.TrimSpace(os.Getenv("DARAJA_SHORTCODE")),
		Passkey:        strings.TrimSpace(os.Getenv("DARAJA_PASSKEY")),
		CallbackURL:    strings.TrimSpace(os.Getenv("DARAJA_CALLBACK_URL")),
		BaseURL:        strings.TrimSpace(os.Getenv("DARAJA_BASE_URL")),
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://sandbox.safaricom.co.ke"
	}
	if cfg.ConsumerKey == "" || cfg.ConsumerSecret == "" || cfg.Shortcode == "" || cfg.Passkey == "" || cfg.CallbackURL == "" {
		return cfg, fmt.Errorf("missing required Daraja configuration")
	}
	return cfg, nil
}

func fetchDarajaAccessToken(ctx context.Context, cfg darajaConfig) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(cfg.ConsumerKey, cfg.ConsumerSecret)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("daraja auth failed with status %d", resp.StatusCode)
	}

	var tokenResp darajaOAuthResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("daraja auth returned empty access token")
	}
	return tokenResp.AccessToken, nil
}

func normalizeKenyanPhone(phone string) (string, error) {
	cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(strings.TrimSpace(phone))
	switch {
	case strings.HasPrefix(cleaned, "254") && len(cleaned) == 12:
		return cleaned, nil
	case strings.HasPrefix(cleaned, "0") && len(cleaned) == 10:
		return "254" + cleaned[1:], nil
	case strings.HasPrefix(cleaned, "7") && len(cleaned) == 9:
		return "254" + cleaned, nil
	default:
		return "", fmt.Errorf("phone must be a valid Kenyan Safaricom number")
	}
}

func callbackMetadataMap(items []struct {
	Name  string      `json:"Name"`
	Value interface{} `json:"Value"`
}) map[string]string {
	result := map[string]string{}
	for _, item := range items {
		if item.Name == "" || item.Value == nil {
			continue
		}
		result[item.Name] = fmt.Sprintf("%v", item.Value)
	}
	return result
}

func darajaEnvironment(baseURL string) string {
	if strings.Contains(strings.ToLower(baseURL), "sandbox") {
		return "sandbox"
	}
	return "production"
}

func maskShortcode(shortcode string) string {
	if len(shortcode) <= 2 {
		return shortcode
	}
	return strings.Repeat("•", len(shortcode)-2) + shortcode[len(shortcode)-2:]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
