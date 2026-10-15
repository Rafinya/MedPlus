# MedPlus POS Backend

## Overview

This project now runs as a full-stack Go-hosted Pharmacy POS application:

- Go backend API in `backend/`
- Split frontend served by the backend from `web/`
- PostgreSQL database
- Authentication, admin CRUD, customers, inventory, cart/checkout, receipts, supply chain, reports, analytics, database overview, and M-Pesa integration support

## Local setup

1. Ensure PostgreSQL is running.
2. Copy `backend/.env.example` to `backend/.env` if needed.
3. Update `backend/.env` with your local database credentials and JWT secret.
4. If `go` is not recognized in your terminal yet, run `setup-go-path.cmd` from the project root once, then restart Zed or your terminal.
5. Start the backend:

```powershell
cd backend
go run .
```

6. Open the app:

```text
http://localhost:8080/
```

If PATH is still not refreshed, you can use the helper scripts from the project root:

```powershell
.\run-go-version.cmd
.\run-go-build.cmd
```

## Docker setup

From the project root:

```powershell
docker compose up --build
```

This starts:

- PostgreSQL on `localhost:5432`
- App on `http://localhost:8080/`

## Environment variables

Important settings in `backend/.env`:

- `APP_ENV` — `development` or `production`
- `PORT` — HTTP port
- `DATABASE_URL` — optional full PostgreSQL connection string
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` — fallback DB settings
- `JWT_SECRET` — required in production
- `ALLOWED_ORIGINS` — required in production, comma-separated allowed browser origins
- `BOOTSTRAP_DB` — whether the app should try to create the database automatically
- `SEED_DEMO_DATA` — whether demo users/products should be seeded
- `ENABLE_PUBLIC_REGISTRATION` — enable or disable public self-registration
- `DARAJA_BASE_URL` — Daraja API base URL, typically sandbox or production
- `DARAJA_CONSUMER_KEY` and `DARAJA_CONSUMER_SECRET` — Daraja API credentials
- `DARAJA_SHORTCODE` and `DARAJA_PASSKEY` — M-Pesa paybill/till configuration for STK push
- `DARAJA_CALLBACK_URL` — public HTTPS callback endpoint used by Safaricom for payment confirmation

## Daraja / M-Pesa checkout

The backend now supports:

- `GET /api/payments/config`
- `POST /api/payments/mpesa/stk-push`
- `GET /api/payments/mpesa/status/{checkoutRequestId}`
- `POST /api/payments/mpesa/callback`

Checkout behavior:

- When `M-Pesa` is selected in the POS, the frontend requests an STK push from the backend.
- The sale is then saved with a payment reference and an initial `pending` payment status.
- Safaricom callbacks update the persisted payment audit record and mark the sale payment status as `completed` or `failed`.

For production:

- `DARAJA_CALLBACK_URL` must be publicly reachable over HTTPS.
- Use production Daraja credentials and base URL.
- Do not commit real secrets into tracked files.

## Production notes

For production hosting:

1. Set `APP_ENV=production`
2. Set a strong `JWT_SECRET`
3. Set explicit `ALLOWED_ORIGINS`
4. Set `SEED_DEMO_DATA=false`
5. Set `ENABLE_PUBLIC_REGISTRATION=false` unless you explicitly want cashier self-registration
6. Prefer a managed/provisioned PostgreSQL database and set `BOOTSTRAP_DB=false`
7. Use HTTPS at the reverse proxy or hosting layer

## Default demo users

Demo users are only intended for development when `SEED_DEMO_DATA=true`:

- `admin` / `adminpassword`
- `pharmacist` / `pharmpassword`
- `cashier` / `cashierpassword`

## Main API areas

- Authentication: `/api/auth/login`, `/api/auth/register`, `/api/auth/me`, `/api/auth/logout`
- Products and inventory: `/api/products`, `/api/admin/products`, `/api/inventory`
- Sales and receipts: `/api/sales`, `/api/receipts/{saleId}`
- Customers: `/api/customers`, `/api/customers/{id}`
- Admin users CRUD: `/api/users`, `/api/users/{id}`
- Supply chain: `/api/suppliers`, `/api/purchase-orders`, `/api/purchase-orders/{id}/receive`
- Payments: `/api/payments/config`, `/api/payments/mpesa/stk-push`, `/api/payments/mpesa/status/{checkoutRequestId}`, `/api/payments/mpesa/callback`
- Analytics and reports: `/api/dashboard/stats`, `/api/reports`, `/api/analytics`, `/api/database-overview`
- Health endpoints: `/api/health`, `/api/ready`
