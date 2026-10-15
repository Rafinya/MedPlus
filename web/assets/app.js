function getInitialApiBaseUrl() {
    const saved = localStorage.getItem("medplus_api_base_url");
    if (window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1") {
        if (!saved || saved.startsWith("http://localhost:") || saved.startsWith("http://127.0.0.1:")) {
            return "";
        }
        return saved;
    }
    return saved || "";
}

const state = {
    apiBaseUrl: getInitialApiBaseUrl(),
                token: localStorage.getItem("medplus_token") || "",
                user: JSON.parse(
                    localStorage.getItem("medplus_user") || "null",
                ),
                authMode: "login",
                cart: [],
                currentCategory: "all",
                pendingProduct: null,
                products: [],
                inventory: [],
                users: [],
                suppliers: [],
                purchaseOrders: [],
                reports: null,
                analytics: null,
                databaseOverview: null,
                databaseAccessToken: localStorage.getItem("medplus_db_access_token") || "",
                dashboard: null,
                currentView: "pos",
                lastReceiptSaleId: null,
                prescriptions: [],
                customers: [],
            };

            document.addEventListener("DOMContentLoaded", async function () {
                updateDate();
                const apiInput = document.getElementById("apiBaseUrlInput");
                if (apiInput) {
                    apiInput.value = state.apiBaseUrl;
                }
                const authInput = document.getElementById("authApiBaseUrlInput");
                if (authInput) {
                    authInput.value = state.apiBaseUrl;
                }
                document
                    .getElementById("searchInput")
                    .addEventListener("input", handleSearch);
                document
                    .getElementById("authForm")
                    .addEventListener("submit", submitAuthForm);
                document
                    .getElementById("userForm")
                    .addEventListener("submit", submitUserForm);
                document
                    .getElementById("supplierForm")
                    .addEventListener("submit", submitSupplierForm);
                document
                    .getElementById("purchaseOrderForm")
                    .addEventListener("submit", submitPurchaseOrderForm);
                document
                    .getElementById("prescriptionForm")
                    .addEventListener("submit", submitPrescriptionForm);
                document
                    .getElementById("customerForm")
                    .addEventListener("submit", submitCustomerForm);
                document
                    .getElementById("profileForm")
                    .addEventListener("submit", submitProfileForm);
                bindModalClose();
                if (state.token) {
                    try {
                        await bootstrapAuthenticatedApp();
                    } catch (error) {
                        showToast(
                            "Session Expired",
                            error.message || "Please login again.",
                            "warning",
                        );
                        clearSession();
                        showAuthScreen();
                    }
                } else {
                    showAuthScreen();
                }
            });

            function bindModalClose() {
                document
                    .querySelectorAll(".modal-overlay")
                    .forEach((overlay) => {
                        overlay.addEventListener("click", function (e) {
                            if (e.target === this)
                                this.classList.remove("active");
                        });
                    });
            }

            function updateDate() {
                const now = new Date();
                const options = {
                    weekday: "short",
                    year: "numeric",
                    month: "short",
                    day: "numeric",
                };
                document.getElementById("currentDate").textContent =
                    now.toLocaleDateString("en-US", options);
            }

            function switchAuthMode(mode) {
                state.authMode = mode;
                document
                    .getElementById("loginTab")
                    .classList.toggle("active", mode === "login");
                document
                    .getElementById("registerTab")
                    .classList.toggle("active", mode === "register");
                document
                    .getElementById("registerFields")
                    .classList.toggle("hidden", mode !== "register");
                document.getElementById("authSubmitBtn").textContent =
                    mode === "login" ? "Login" : "Create Account";
            }

            async function submitAuthForm(event) {
                event.preventDefault();
                const username = document
                    .getElementById("authUsername")
                    .value.trim();
                const password = document.getElementById("authPassword").value;
                try {
                    if (state.authMode === "login") {
                        const response = await apiFetch(
                            "/api/auth/login",
                            {
                                method: "POST",
                                body: JSON.stringify({ username, password }),
                            },
                            false,
                        );
                        saveSession(response);
                    } else {
                        const payload = {
                            username,
                            password,
                            full_name: document
                                .getElementById("registerFullName")
                                .value.trim(),
                        };
                        const response = await apiFetch(
                            "/api/auth/register",
                            { method: "POST", body: JSON.stringify(payload) },
                            false,
                        );
                        saveSession(response);
                    }
                    await bootstrapAuthenticatedApp();
                    
                    // Show successful authentication toast
                    showToast(
                        "Authentication Successful",
                        "Welcome to MedPlus Pharmacy POS.",
                        "success",
                    );
                    
                    // Show central welcome modal popup
                    const welcomeModalMessage = document.getElementById("welcomeModalMessage");
                    if (welcomeModalMessage) {
                        const displayName = state.user?.full_name || state.user?.username || username;
                        welcomeModalMessage.innerHTML = `Welcome back, <strong style="color: var(--brand);">${escapeHtml(displayName)}</strong>!`;
                    }
                    const welcomeModal = document.getElementById("welcomeModal");
                    if (welcomeModal) {
                        welcomeModal.classList.add("active");
                    }
                } catch (error) {
                    showToast(
                        "Authentication Failed",
                        error.message || "Unable to continue.",
                        "error",
                    );
                }
            }

            function saveSession(response) {
                state.token = response.token;
                state.user = response.user;
                localStorage.setItem("medplus_token", state.token);
                localStorage.setItem(
                    "medplus_user",
                    JSON.stringify(state.user),
                );
            }

            function clearSession() {
                state.token = "";
                state.user = null;
                state.databaseAccessToken = "";
                localStorage.removeItem("medplus_token");
                localStorage.removeItem("medplus_user");
                localStorage.removeItem("medplus_db_access_token");
            }

            function showAuthScreen() {
                document
                    .getElementById("authScreen")
                    .classList.remove("hidden");
                document.getElementById("appShell").classList.add("hidden");
            }

            function showAppShell() {
                document.getElementById("authScreen").classList.add("hidden");
                document.getElementById("appShell").classList.remove("hidden");
            }

            async function bootstrapAuthenticatedApp() {
                const me = await apiFetch("/api/auth/me");
                state.user = me;
                localStorage.setItem("medplus_user", JSON.stringify(me));
                hydrateUserProfile();
                applyRolePermissions();
                showAppShell();
                await loadCoreData();
                showView(state.currentView || "pos");
            }

            function hydrateUserProfile() {
                const name =
                    state.user?.full_name || state.user?.username || "User";
                document.getElementById("userName").textContent = name;
                document.getElementById("userRole").textContent =
                    state.user?.role || "Role";
                document.getElementById("userAvatar").textContent = name
                    .split(" ")
                    .map((part) => part[0])
                    .join("")
                    .slice(0, 2)
                    .toUpperCase();
                populateProfileFields();
            }

            async function loadCoreData() {
                await Promise.all([
                    loadProducts(),
                    loadInventory(),
                    loadDashboard(),
                    loadReports(),
                    loadAnalytics(),
                    loadDatabaseOverview(),
                    loadSuppliers(),
                    loadPurchaseOrders(),
                    loadUsersSafe(),
                    loadPrescriptions(),
                    loadCustomers(),
                    loadSales(),
                ]);
                renderProducts();
                renderInventory();
                renderDashboard();
                renderReports();
                renderAnalytics();
                renderDatabaseOverview();
                renderSuppliers();
                renderPurchaseOrders();
                renderUsers();
                renderPrescriptions();
                renderPrescriptionProductOptions();
                renderCustomers();
                updateCart();
                renderDashboardFeeds();
            }

            async function refreshActiveView() {
                await loadCoreData();
                showToast(
                    "Refreshed",
                    "Latest backend data loaded.",
                    "success",
                );
            }

            function saveApiBaseUrl() {
                const apiInput = document.getElementById("apiBaseUrlInput");
                state.apiBaseUrl = (apiInput && apiInput.value.trim()) || "";
                localStorage.setItem("medplus_api_base_url", state.apiBaseUrl);
                const authInput = document.getElementById("authApiBaseUrlInput");
                if (authInput) {
                    authInput.value = state.apiBaseUrl;
                }
                showToast("Saved", "API base URL updated.", "success");
            }

            function toggleAuthApiConfig() {
                const fields = document.getElementById("authApiConfigFields");
                const icon = document.getElementById("authApiConfigIcon");
                if (!fields || !icon) return;
                if (fields.classList.contains("hidden")) {
                    fields.classList.remove("hidden");
                    icon.className = "fas fa-chevron-up";
                } else {
                    fields.classList.add("hidden");
                    icon.className = "fas fa-chevron-down";
                }
            }

            function saveAuthApiBaseUrl() {
                const authInput = document.getElementById("authApiBaseUrlInput");
                const val = (authInput && authInput.value.trim()) || "";
                state.apiBaseUrl = val;
                localStorage.setItem("medplus_api_base_url", state.apiBaseUrl);
                const apiInput = document.getElementById("apiBaseUrlInput");
                if (apiInput) {
                    apiInput.value = state.apiBaseUrl;
                }
                showToast("Saved", "API base URL updated.", "success");
            }

            async function logout() {
                try {
                    if (state.token)
                        await apiFetch("/api/auth/logout", { method: "POST" });
                } catch (_) {}
                clearSession();
                state.cart = [];
                showAuthScreen();
                showToast(
                    "Logged Out",
                    "You have been signed out successfully.",
                    "success",
                );
            }

            async function apiFetch(path, options = {}, auth = true) {
                const headers = {
                    "Content-Type": "application/json",
                    ...(options.headers || {}),
                };
                if (auth && state.token)
                    headers.Authorization = `Bearer ${state.token}`;
                let baseUrl = state.apiBaseUrl || "";
                if (window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1") {
                    if (baseUrl.startsWith("http://localhost:") || baseUrl.startsWith("http://127.0.0.1:")) {
                        baseUrl = "";
                    }
                }
                if (baseUrl.endsWith("/")) {
                    baseUrl = baseUrl.slice(0, -1);
                }
                const response = await fetch(baseUrl + path, {
                    ...options,
                    headers,
                });
                const contentType = response.headers.get("content-type") || "";
                const payload = contentType.includes("application/json")
                    ? await response.json()
                    : await response.text();
                if (!response.ok) {
                    const message =
                        typeof payload === "string"
                            ? payload
                            : payload.error ||
                              payload.message ||
                              "Request failed";
                    throw new Error(message);
                }
                return payload;
            }

            async function loadSales() {
                try {
                    state.sales = await apiFetch("/api/sales");
                } catch (error) {
                    console.error("Failed to load sales history:", error);
                    state.sales = [];
                }
            }

            function renderDashboardWelcome() {
                const nameEl = document.getElementById("welcomeGreeting");
                if (!nameEl) return;
                const userFullName = state.user?.full_name || state.user?.username || "Pharmacist";
                const userRole = state.user?.role || "Pharmacist";
                nameEl.innerHTML = `<i class="fas fa-hand-holding-medical" style="color: #2dd4bf; margin-right: 6px;"></i> Welcome Back, ${escapeHtml(userFullName)}!`;
                
                const descEl = document.getElementById("welcomeDescription");
                if (descEl) {
                    descEl.textContent = `Logged in as ${userRole}. Here is the real-time operational status for MedPlus Pharmacy today.`;
                }

                // Show/hide admin database quick task card based on role
                const qaAdminDb = document.getElementById("qaAdminDb");
                if (qaAdminDb) {
                    qaAdminDb.style.display = userRole === "Admin" ? "flex" : "none";
                }
            }

            function renderDashboardFeeds() {
                // 1. Render Recent Transactions
                const salesFeed = document.getElementById("recentSalesFeedList");
                const salesCountEl = document.getElementById("recentSalesCount");
                if (salesFeed) {
                    const sales = state.sales || [];
                    salesCountEl.textContent = `${sales.length} total`;
                    if (!sales.length) {
                        salesFeed.innerHTML = `<div class="empty-state" style="padding: 20px;">No recent sales.</div>`;
                    } else {
                        // Limit to top 5 recent sales
                        salesFeed.innerHTML = sales.slice(0, 5).map(sale => {
                            const date = new Date(sale.created_at);
                            const formattedTime = date.toLocaleTimeString("en-US", { hour: '2-digit', minute: '2-digit' });
                            return `
                                <div class="feed-item sale-item">
                                    <div class="feed-details">
                                        <div class="feed-main-txt">Sale ID: ${escapeHtml(sale.transaction_id.slice(0, 8))}...</div>
                                        <div class="feed-sub-txt">${escapeHtml(sale.payment_method)} payment · Cashier: ${escapeHtml(sale.cashier_name || "N/A")}</div>
                                    </div>
                                    <div class="feed-right-txt">
                                        <div class="feed-value">${formatCurrency(sale.total)}</div>
                                        <div class="feed-meta">${formattedTime}</div>
                                    </div>
                                </div>
                            `;
                        }).join("");
                    }
                }

                // 2. Render Expiring Medicine Alerts
                const expiryFeed = document.getElementById("expiringMedicinesFeedList");
                const expiryCountEl = document.getElementById("expiringMedicinesCount");
                if (expiryFeed) {
                    const products = state.inventory || [];
                    const now = new Date();
                    
                    // Filter products with valid expiry date
                    const expiringProducts = products.filter(p => {
                        if (!p.expiry_date || p.status !== "active") return false;
                        const expDate = new Date(p.expiry_date);
                        if (Number.isNaN(expDate.getTime())) return false;
                        // Expiring in the next 180 days (or already expired)
                        const daysLeft = Math.ceil((expDate - now) / (1000 * 60 * 60 * 24));
                        return daysLeft <= 180;
                    }).map(p => {
                        const expDate = new Date(p.expiry_date);
                        const daysLeft = Math.ceil((expDate - now) / (1000 * 60 * 60 * 24));
                        return { ...p, daysLeft };
                    });

                    // Sort by days left (closest first)
                    expiringProducts.sort((a, b) => a.daysLeft - b.daysLeft);

                    expiryCountEl.textContent = `${expiringProducts.length} alert${expiringProducts.length !== 1 ? 's' : ''}`;

                    if (!expiringProducts.length) {
                        expiryFeed.innerHTML = `<div class="empty-state" style="padding: 20px;">No expiry alerts. All products valid.</div>`;
                    } else {
                        // Limit to top 5 expiry alerts
                        expiryFeed.innerHTML = expiringProducts.slice(0, 5).map(p => {
                            const isCritical = p.daysLeft <= 30;
                            const isExpired = p.daysLeft <= 0;
                            const alertClass = isCritical ? 'expiry-critical' : 'expiry-warning';
                            const badgeColor = isCritical ? '#ef4444' : '#f59e0b';
                            
                            let timeStatusText = "";
                            if (isExpired) {
                                timeStatusText = `Expired (${Math.abs(p.daysLeft)} days ago)`;
                            } else {
                                timeStatusText = `Expires in ${p.daysLeft} days`;
                            }

                            return `
                                <div class="feed-item ${alertClass}">
                                    <div class="feed-details">
                                        <div class="feed-main-txt">${escapeHtml(p.name)}</div>
                                        <div class="feed-sub-txt">Stock: ${p.stock} units · ${escapeHtml(p.brand || "Generic")}</div>
                                    </div>
                                    <div class="feed-right-txt">
                                        <div class="feed-value" style="color: ${badgeColor}; font-size: 12px; font-weight: 700;">
                                            ${timeStatusText}
                                        </div>
                                        <div class="feed-meta">Exp: ${formatDate(p.expiry_date)}</div>
                                    </div>
                                </div>
                            `;
                        }).join("");
                    }
                }
            }

            function showView(view, event) {
                if (!canAccessView(view)) {
                    showToast(
                        "Access Denied",
                        "Your account does not have permission to open this module.",
                        "warning",
                    );
                    view = "pos";
                }
                state.currentView = view;
                
                // Show floating pill page loader
                const loader = document.getElementById("pageLoader");
                if (loader) {
                    loader.classList.add("active");
                }
                
                // Instantly update active sidebar nav link for immediate feedback
                document
                    .querySelectorAll(".nav-item")
                    .forEach((item) => item.classList.remove("active"));
                const navItem =
                    event?.currentTarget ||
                    document.querySelector(`.nav-item[data-view="${view}"]`);
                if (navItem) navItem.classList.add("active");

                // Execute actual view transition after 350ms (mid-point of animation)
                setTimeout(() => {
                    [
                        "pos",
                        "inventory",
                        "dashboard",
                        "reports",
                        "analytics",
                        "database",
                        "admin",
                        "supply",
                        "prescriptions",
                        "customers",
                        "settings",
                    ].forEach((name) => {
                        const viewEl = document.getElementById(`${name}View`);
                        if (viewEl) {
                            viewEl.classList.add("hidden");
                        }
                    });
                    
                    const newViewEl = document.getElementById(`${view}View`);
                    if (newViewEl) {
                        newViewEl.classList.remove("hidden");
                    }
                    
                    document.getElementById("cartSection").style.display =
                        view === "pos" ? "flex" : "none";
                    
                    // Render dashboard special items if it is the active view
                    if (view === "dashboard") {
                        renderDashboardWelcome();
                        renderDashboardFeeds();
                    }
                }, 350);
                
                // Hide page loader after 700ms total transition time
                setTimeout(() => {
                    if (loader) {
                        loader.classList.remove("active");
                    }
                }, 700);
            }

            function handleSearch(event) {
                renderProducts(event.target.value.trim().toLowerCase());
            }

            async function loadProducts() {
                const search = document
                    .getElementById("searchInput")
                    ?.value?.trim();
                const query = new URLSearchParams();
                if (state.currentCategory !== "all")
                    query.set("category", state.currentCategory);
                if (search) query.set("search", search);
                const suffix = query.toString() ? `?${query.toString()}` : "";
                state.products = await apiFetch(`/api/products${suffix}`);
                document.getElementById("inventoryBadge").textContent =
                    state.products.filter(
                        (p) => p.stock <= p.reorder_level,
                    ).length;
            }

            async function loadInventory() {
                state.inventory = await apiFetch("/api/inventory");
            }
            async function loadDashboard() {
                state.dashboard = await apiFetch("/api/dashboard/stats");
            }
            async function loadReports() {
                try {
                    state.reports = await apiFetch("/api/reports");
                    renderReports();
                } catch (error) {
                    state.reports = { summary: null, top_products: [] };
                    renderReports();
                }
            }
            async function loadAnalytics() {
                try {
                    state.analytics = await apiFetch("/api/analytics");
                    renderAnalytics();
                } catch (error) {
                    state.analytics = {
                        daily_sales: [],
                        category_breakdown: [],
                    };
                    renderAnalytics();
                }
            }
            async function loadDatabaseOverview() {
                try {
                    const headers = {};
                    if (state.databaseAccessToken) {
                        headers["X-Database-Access-Token"] = state.databaseAccessToken;
                    }
                    state.databaseOverview = await apiFetch("/api/database-overview", { headers });
                    
                    document.getElementById("databaseUnlockContainer").classList.add("hidden");
                    document.getElementById("databaseOverviewContent").classList.remove("hidden");
                    document.getElementById("refreshDbOverviewBtn").style.display = "block";
                    
                    renderDatabaseOverview();
                } catch (error) {
                    if (error.message.includes("verification required") || error.message.includes("Forbidden") || error.message.includes("403")) {
                        document.getElementById("databaseUnlockContainer").classList.remove("hidden");
                        document.getElementById("databaseOverviewContent").classList.add("hidden");
                        document.getElementById("refreshDbOverviewBtn").style.display = "none";
                    } else {
                        showToast("Error", error.message, "error");
                        state.databaseOverview = { overview: [] };
                        renderDatabaseOverview();
                    }
                }
            }
            async function loadCustomers() {
                try {
                    state.customers = await apiFetch("/api/customers");
                    renderCustomers();
                    renderCustomerOptions();
                } catch (error) {
                    state.customers = [];
                    renderCustomers();
                    renderCustomerOptions();
                }
            }

            function renderCustomerOptions() {
                const datalist = document.getElementById("customerList");
                if (!datalist) return;
                if (!state.customers.length) {
                    datalist.innerHTML = '';
                    return;
                }
                datalist.innerHTML = state.customers
                    .filter((c) => c.status === "active")
                    .map(
                        (c) => `<option value="${escapeHtml(c.full_name)}">${escapeHtml(c.phone || "")}</option>`
                    )
                    .join("");
            }

            function renderCustomers() {
                const tbody = document.getElementById("customersTable");
                if (!tbody) return;

                const query = (document.getElementById("customerSearchInput")?.value || "").trim().toLowerCase();
                const filtered = (state.customers || []).filter((c) => {
                    if (!query) return true;
                    return (
                        (c.full_name || "").toLowerCase().includes(query) ||
                        (c.phone || "").toLowerCase().includes(query) ||
                        (c.email || "").toLowerCase().includes(query)
                    );
                });

                if (!filtered.length) {
                    tbody.innerHTML = `<tr><td colspan="6" class="empty-state" style="text-align: center; padding: 24px; color: var(--muted);">No customer profiles found.</td></tr>`;
                    return;
                }

                tbody.innerHTML = filtered
                    .map(
                        (c) => `
                    <tr style="border-bottom: 1px solid var(--border);">
                        <td style="padding: 12px 16px; font-weight: 600;">
                            <div style="display: flex; align-items: center; gap: 10px;">
                                <div style="width: 32px; height: 32px; border-radius: 50%; background: rgba(16,185,129,0.15); color: #10b981; display: flex; align-items: center; justify-content: center; font-weight: bold; font-size: 13px;">
                                    ${escapeHtml((c.full_name || "C").charAt(0).toUpperCase())}
                                </div>
                                <div>${escapeHtml(c.full_name)}</div>
                            </div>
                        </td>
                        <td style="padding: 12px 16px; color: var(--muted); font-size: 13px;">
                            ${c.phone ? `<span style="font-family: monospace;">${escapeHtml(c.phone)}</span>` : '<span style="opacity: 0.5;">—</span>'}
                        </td>
                        <td style="padding: 12px 16px; color: var(--muted); font-size: 13px;">
                            ${c.email ? escapeHtml(c.email) : '<span style="opacity: 0.5;">—</span>'}
                        </td>
                        <td style="padding: 12px 16px; max-width: 240px; font-size: 12px; color: var(--muted);">
                            ${c.notes ? escapeHtml(c.notes) : '<span style="opacity: 0.4;">No notes</span>'}
                        </td>
                        <td style="padding: 12px 16px;">
                            <span class="badge ${c.status === "inactive" ? "danger" : "success"}">${c.status === "inactive" ? "Inactive" : "Active"}</span>
                        </td>
                        <td style="padding: 12px 16px; text-align: right;">
                            <div class="inline-actions" style="justify-content: flex-end; gap: 6px;">
                                <button class="small-btn" onclick="editCustomer(${c.id})"><i class="fas fa-edit"></i> Edit</button>
                                <button class="danger-btn small-btn" onclick="deactivateCustomer(${c.id})"><i class="fas fa-archive"></i> Archive</button>
                            </div>
                        </td>
                    </tr>
                `,
                    )
                    .join("");
            }

            function toggleCustomerModal() {
                resetCustomerForm();
                document.getElementById("customerModal").classList.add("active");
            }

            function resetCustomerForm() {
                const form = document.getElementById("customerForm");
                if (form) form.reset();
                const idInput = document.getElementById("customerId");
                if (idInput) idInput.value = "";
                const title = document.getElementById("customerFormTitle");
                if (title) title.textContent = "New Customer Profile";
                const statusSelect = document.getElementById("customerStatus");
                if (statusSelect) statusSelect.value = "active";
            }

            function editCustomer(id) {
                const c = state.customers.find((entry) => entry.id === id);
                if (!c) return;
                document.getElementById("customerId").value = c.id;
                document.getElementById("customerFullName").value = c.full_name || "";
                document.getElementById("customerPhone").value = c.phone || "";
                document.getElementById("customerEmail").value = c.email || "";
                document.getElementById("customerNotes").value = c.notes || "";
                document.getElementById("customerStatus").value = c.status || "active";
                document.getElementById("customerFormTitle").textContent = "Edit Customer Profile";
                document.getElementById("customerModal").classList.add("active");
            }

            async function submitCustomerForm(event) {
                event.preventDefault();
                const id = document.getElementById("customerId").value;
                const payload = {
                    full_name: document.getElementById("customerFullName").value.trim(),
                    phone: document.getElementById("customerPhone").value.trim(),
                    email: document.getElementById("customerEmail").value.trim(),
                    notes: document.getElementById("customerNotes").value.trim(),
                    status: document.getElementById("customerStatus").value,
                };
                try {
                    if (id) {
                        await apiFetch(`/api/customers/${id}`, {
                            method: "PUT",
                            body: JSON.stringify(payload),
                        });
                        showToast("Updated", "Customer profile updated successfully.", "success");
                    } else {
                        await apiFetch("/api/customers", {
                            method: "POST",
                            body: JSON.stringify(payload),
                        });
                        showToast("Created", "Customer profile created successfully.", "success");
                    }
                    closeModal("customerModal");
                    resetCustomerForm();
                    await loadCustomers();
                } catch (error) {
                    showToast("Error", error.message || "Unable to save customer.", "error");
                }
            }

            async function deactivateCustomer(id) {
                if (!confirm("Deactivate/Archive this customer profile?")) return;
                try {
                    await apiFetch(`/api/customers/${id}`, {
                        method: "DELETE",
                    });
                    showToast("Archived", "Customer profile deactivated.", "success");
                    await loadCustomers();
                } catch (error) {
                    showToast("Error", error.message || "Unable to archive customer.", "error");
                }
            }

            async function submitProfileForm(event) {
                event.preventDefault();
                const fullName = document.getElementById("profileFullName").value.trim();
                const username = document.getElementById("profileUsername").value.trim();
                const password = document.getElementById("profilePassword").value;
                const confirmPassword = document.getElementById("profileConfirmPassword").value;

                if (password && password !== confirmPassword) {
                    return showToast("Validation Error", "Passwords do not match.", "error");
                }

                const payload = {
                    full_name: fullName,
                    username: username,
                };
                if (password) payload.password = password;

                try {
                    const updatedUser = await apiFetch("/api/profile", {
                        method: "PUT",
                        body: JSON.stringify(payload),
                    });
                    
                    state.user = updatedUser;
                    localStorage.setItem("medplus_user", JSON.stringify(updatedUser));
                    
                    hydrateUserProfile();
                    
                    document.getElementById("profilePassword").value = "";
                    document.getElementById("profileConfirmPassword").value = "";
                    
                    showToast("Profile Updated", "Your information has been successfully updated.", "success");
                } catch (error) {
                    showToast("Update Failed", error.message || "Unable to update profile.", "error");
                }
            }

            function populateProfileFields() {
                if (state.user) {
                    const fullNameInput = document.getElementById("profileFullName");
                    const usernameInput = document.getElementById("profileUsername");
                    if (fullNameInput) fullNameInput.value = state.user.full_name || "";
                    if (usernameInput) usernameInput.value = state.user.username || "";
                }
            }

            async function loadPrescriptions() {
                try {
                    state.prescriptions = await apiFetch("/api/prescriptions");
                    renderPrescriptions();
                } catch (error) {
                    state.prescriptions = [];
                    renderPrescriptions();
                }
            }

            function renderPrescriptions() {
                const container = document.getElementById("prescriptionsListTable");
                if (!container) return;
                if (!state.prescriptions || !state.prescriptions.length) {
                    container.innerHTML = `<div class="panel empty-state">No prescriptions registered.</div>`;
                    return;
                }
                container.innerHTML = state.prescriptions
                    .map((rx) => {
                        const statusClass =
                            rx.status === "active"
                                ? "status-active"
                                : rx.status === "dispensed"
                                  ? "status-expired"
                                  : "status-expired";
                        const itemsList = rx.items
                            ? rx.items.map(it => `${it.product_name} (Qty: ${it.quantity})`).join(", ")
                            : "No items";
                        return `
                        <div class="list-card">
                            <div style="display:flex; justify-content:space-between; align-items:flex-start;">
                                <strong>${escapeHtml(rx.prescription_number)}</strong>
                                <span class="status-badge ${statusClass}">${escapeHtml(rx.status)}</span>
                            </div>
                            <div class="muted" style="margin-top:6px;">Patient: <strong>${escapeHtml(rx.patient_name)}</strong></div>
                            <div class="muted">Doctor: ${escapeHtml(rx.doctor_name || "N/A")}</div>
                            <div class="muted" style="margin-top:4px; font-size:13px; color:#0f766e;">Med: ${escapeHtml(itemsList)}</div>
                            ${rx.notes ? `<div class="muted" style="font-size:12px; font-style:italic; margin-top:4px;">Notes: ${escapeHtml(rx.notes)}</div>` : ""}
                            <div class="muted" style="font-size:11px; margin-top:6px;">Date: ${formatDate(rx.created_at)}</div>
                        </div>
                    `;
                    })
                    .join("");
            }

            async function renderPrescriptionProductOptions() {
                const select = document.getElementById("rxProductId");
                if (!select) return;
                try {
                    const allProducts = await apiFetch("/api/products");
                    const rxProducts = allProducts.filter(p => p.rx);
                    if (!rxProducts.length) {
                        select.innerHTML = '<option value="">No prescription medicines available</option>';
                        return;
                    }
                    select.innerHTML = rxProducts
                        .map(
                            (p) => `<option value="${p.id}">${escapeHtml(p.name)} (${escapeHtml(p.brand || "Generic")})</option>`
                        )
                        .join("");
                } catch (error) {
                    select.innerHTML = '<option value="">Failed to load medicines</option>';
                }
            }

            async function submitPrescriptionForm(event) {
                event.preventDefault();
                const patientName = document.getElementById("rxPatientName").value.trim();
                const doctorName = document.getElementById("rxDoctorName").value.trim();
                const productId = Number(document.getElementById("rxProductId").value);
                const quantity = Number(document.getElementById("rxQuantity").value);
                const notes = document.getElementById("rxNotes").value.trim();

                const payload = {
                    patient_name: patientName,
                    doctor_name: doctorName,
                    notes: notes,
                    status: "active",
                    items: [
                        {
                            product_id: productId,
                            quantity: quantity
                        }
                    ]
                };

                try {
                    const response = await apiFetch("/api/prescriptions", {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    showToast(
                        "Created",
                        `Prescription ${response.prescription_number} created successfully.`,
                        "success",
                    );
                    document.getElementById("prescriptionForm").reset();
                    await loadPrescriptions();
                    await loadDashboard();
                } catch (error) {
                    showToast(
                        "Prescription Error",
                        error.message || "Unable to save prescription.",
                        "error",
                    );
                }
            }
            async function loadSuppliers() {
                try {
                    state.suppliers = await apiFetch("/api/suppliers");
                    renderSupplierOptions();
                    renderSuppliers();
                } catch (error) {
                    state.suppliers = [];
                    renderSupplierOptions();
                    renderSuppliers();
                }
            }
            async function loadPurchaseOrders() {
                try {
                    state.purchaseOrders = await apiFetch(
                        "/api/purchase-orders",
                    );
                    renderPurchaseOrders();
                } catch (error) {
                    state.purchaseOrders = [];
                    renderPurchaseOrders();
                }
            }
            async function loadUsersSafe() {
                try {
                    state.users = await apiFetch("/api/users");
                    renderUsers();
                } catch (error) {
                    state.users = [];
                    renderUsers();
                }
            }

            function renderProducts(searchTerm = "") {
                const grid = document.getElementById("productsGrid");
                const filtered = state.products.filter((product) => {
                    if (!searchTerm) return true;
                    return [product.name, product.brand, product.barcode]
                        .filter(Boolean)
                        .some((value) =>
                            String(value).toLowerCase().includes(searchTerm),
                        );
                });
                if (!filtered.length) {
                    grid.innerHTML = `<div class="panel empty-state" style="grid-column:1/-1;">No products found for the current filters.</div>`;
                    return;
                }
                grid.innerHTML = filtered
                    .map((product) => {
                        const todayStr = new Date().toISOString().slice(0, 10);
                        const expStr = (product.expiry_date || "").slice(0, 10);
                        const isExpired = expStr !== "" && expStr <= todayStr;

                        const stockClass = isExpired
                            ? "stock-critical"
                            : product.stock > 20
                                ? "stock-good"
                                : product.stock > 5
                                  ? "stock-low"
                                  : "stock-critical";
                        const stockText = isExpired
                            ? "EXPIRED"
                            : product.stock > 20
                                ? "In Stock"
                                : product.stock > 5
                                  ? "Low Stock"
                                  : "Critical";

                        const expiredBadge = isExpired
                            ? `<div class="expired-badge-chip" style="background-color: #ef4444; color: #ffffff; font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px; display: inline-flex; align-items: center; gap: 4px; margin-bottom: 6px; box-shadow: 0 2px 4px rgba(239,68,68,0.3);"><i class="fas fa-exclamation-triangle"></i> EXPIRED</div>`
                            : "";

                        return `
                    <div class="product-card ${product.rx ? "prescription" : ""} ${isExpired ? "expired-card" : ""}" style="${isExpired ? "border: 2px solid #ef4444; background: rgba(239, 68, 68, 0.05);" : ""}" onclick="addToCart(${product.id})">
                        <div class="product-image">${product.icon || "💊"}</div>
                        ${expiredBadge}
                        <div class="product-name">${escapeHtml(product.name)}</div>
                        <div class="product-footer">
                            <div class="product-price">${formatCurrency(product.price)}</div>
                            <div class="product-stock ${stockClass}" style="${isExpired ? "color: #ef4444; font-weight: 700;" : ""}">${stockText}</div>
                        </div>
                    </div>
                `;
                    })
                    .join("");
            }

            function filterCategory(category, event) {
                state.currentCategory = category;
                document
                    .querySelectorAll("[data-category]")
                    .forEach((chip) => chip.classList.remove("active"));
                if (event?.currentTarget)
                    event.currentTarget.classList.add("active");
                loadProducts().then(() =>
                    renderProducts(
                        document
                            .getElementById("searchInput")
                            .value.trim()
                            .toLowerCase(),
                    ),
                );
            }

            function populateRxModalOptions(productId) {
                const rxInput = document.getElementById("rxNumber");
                if (!rxInput) return;
                
                const matchingRxs = state.prescriptions.filter(rx => 
                    rx.status === "active" && 
                    rx.items && 
                    rx.items.some(item => item.product_id === productId)
                );
                
                let datalist = document.getElementById("activeRxNumbers");
                if (!datalist) {
                    datalist = document.createElement("datalist");
                    datalist.id = "activeRxNumbers";
                    document.body.appendChild(datalist);
                    rxInput.setAttribute("list", "activeRxNumbers");
                }
                
                datalist.innerHTML = matchingRxs
                    .map(rx => `<option value="${escapeHtml(rx.prescription_number)}">${escapeHtml(rx.patient_name)} (Qty: ${rx.items.find(i => i.product_id === productId)?.quantity || 0})</option>`)
                    .join("");
                
                const onInputHandler = function() {
                    const typed = this.value.trim().toLowerCase();
                    const matched = matchingRxs.find(rx => rx.prescription_number.toLowerCase() === typed);
                    if (matched) {
                        document.getElementById("rxPatient").value = matched.patient_name;
                        document.getElementById("rxDoctor").value = matched.doctor_name || "";
                    }
                };
                
                rxInput.removeEventListener("input", rxInput._onInputHandler);
                rxInput._onInputHandler = onInputHandler;
                rxInput.addEventListener("input", onInputHandler);
            }

            function addToCart(productId) {
                const product =
                    state.products.find((p) => p.id === productId) ||
                    state.inventory.find((p) => p.id === productId);
                if (!product) return;

                const todayStr = new Date().toISOString().slice(0, 10);
                const expStr = (product.expiry_date || "").slice(0, 10);
                if (expStr !== "" && expStr <= todayStr) {
                    return showToast(
                        "Expired Medicine Blocked",
                        `Cannot sell '${product.name}' because it has expired (${formatDate(product.expiry_date)}).`,
                        "error"
                    );
                }

                if (product.rx) {
                    state.pendingProduct = product;
                    populateRxModalOptions(product.id);
                    document.getElementById("rxModal").classList.add("active");
                    return;
                }
                const existing = state.cart.find(
                    (item) => item.id === productId,
                );
                if (existing) {
                    if (existing.quantity + 1 > product.stock)
                        return showToast(
                            "Stock Limit",
                            "Cannot add more than available stock.",
                            "warning",
                        );
                    existing.quantity += 1;
                } else {
                    state.cart.push({ ...product, quantity: 1 });
                }
                updateCart();
                showToast(
                    "Added to Cart",
                    `${product.name} added successfully.`,
                    "success",
                );
            }

            function verifyPrescription() {
                const rxNumber = document
                    .getElementById("rxNumber")
                    .value.trim();
                const patient = document
                    .getElementById("rxPatient")
                    .value.trim();
                const doctor = document.getElementById("rxDoctor").value.trim();
                if (!rxNumber || !patient || !doctor || !state.pendingProduct)
                    return showToast(
                        "Error",
                        "Please fill in all prescription details.",
                        "error",
                    );
                
                const matchedRx = state.prescriptions.find(rx => rx.prescription_number.toLowerCase() === rxNumber.toLowerCase());
                if (!matchedRx) {
                    return showToast(
                        "Invalid Prescription",
                        `No prescription with number '${rxNumber}' found in the system.`,
                        "error"
                    );
                }
                
                if (matchedRx.status !== "active") {
                    return showToast(
                        "Prescription Inactive",
                        `Prescription '${rxNumber}' is already ${matchedRx.status}.`,
                        "error"
                    );
                }
                
                const rxItem = matchedRx.items ? matchedRx.items.find(it => it.product_id === state.pendingProduct.id) : null;
                if (!rxItem) {
                    return showToast(
                        "Medicine Mismatch",
                        `Prescription '${rxNumber}' does not prescribe ${state.pendingProduct.name}.`,
                        "error"
                    );
                }
                
                if (matchedRx.patient_name.toLowerCase() !== patient.toLowerCase()) {
                    return showToast(
                        "Patient Mismatch",
                        `Prescription '${rxNumber}' is for patient '${matchedRx.patient_name}', but input was '${patient}'.`,
                        "warning"
                    );
                }
                
                const existing = state.cart.find(
                    (item) => item.id === state.pendingProduct.id,
                );
                const nextQty = existing ? existing.quantity + 1 : 1;
                if (nextQty > rxItem.quantity) {
                    return showToast(
                        "Prescription Limit",
                        `Cannot add more than the prescribed quantity of ${rxItem.quantity}.`,
                        "error"
                    );
                }

                if (existing) {
                    if (existing.quantity + 1 > state.pendingProduct.stock)
                        return showToast(
                            "Stock Limit",
                            "Cannot add more than available stock.",
                            "warning",
                        );
                    existing.quantity += 1;
                    existing.rxNumber = rxNumber;
                    existing.patient = patient;
                    existing.doctor = doctor;
                } else {
                    state.cart.push({
                        ...state.pendingProduct,
                        quantity: 1,
                        rxNumber,
                        patient,
                        doctor,
                    });
                }
                closeModal("rxModal");
                document.getElementById("rxNumber").value = "";
                document.getElementById("rxPatient").value = "";
                document.getElementById("rxDoctor").value = "";
                showToast(
                    "Prescription Verified",
                    `${state.pendingProduct.name} added with valid prescription details.`,
                    "success",
                );
                state.pendingProduct = null;
                updateCart();
            }

            function updateCart() {
                const cartItems = document.getElementById("cartItems");
                const cartCount = document.getElementById("cartCount");
                const checkoutBtn = document.getElementById("checkoutBtn");
                cartCount.textContent = state.cart.reduce(
                    (sum, item) => sum + item.quantity,
                    0,
                );
                if (!state.cart.length) {
                    cartItems.innerHTML = `<div class="cart-empty"><i class="fas fa-shopping-basket"></i><p>Your cart is empty</p><p style="font-size:12px; margin-top:4px;">Add products to begin checkout</p></div>`;
                    checkoutBtn.disabled = true;
                } else {
                    cartItems.innerHTML = state.cart
                        .map(
                            (item) => `
                    <div class="cart-item">
                        <div class="cart-item-icon">${item.icon || "💊"}</div>
                        <div class="cart-item-details">
                            <div class="cart-item-name">${escapeHtml(item.name)}</div>
                            <div class="cart-item-price">${formatCurrency(item.price)} each</div>
                            ${item.rxNumber ? `<div style="font-size:11px; color:#f59e0b;">Rx: ${escapeHtml(item.rxNumber)}</div>` : ""}
                        </div>
                        <div class="cart-item-qty">
                            <button class="qty-btn" onclick="updateQuantity(${item.id}, -1)">-</button>
                            <span style="font-weight:600; min-width:20px; text-align:center;">${item.quantity}</span>
                            <button class="qty-btn" onclick="updateQuantity(${item.id}, 1)">+</button>
                        </div>
                        <div class="cart-item-total">${formatCurrency(item.price * item.quantity)}</div>
                        <button class="remove-item" onclick="removeFromCart(${item.id})"><i class="fas fa-times"></i></button>
                    </div>
                `,
                        )
                        .join("");
                    checkoutBtn.disabled = false;
                }
                const subtotal = state.cart.reduce(
                    (sum, item) => sum + item.price * item.quantity,
                    0,
                );
                const tax = subtotal * 0.085;
                const total = subtotal + tax;
                document.getElementById("subtotal").textContent =
                    formatCurrency(subtotal);
                document.getElementById("tax").textContent =
                    formatCurrency(tax);
                document.getElementById("total").textContent =
                    formatCurrency(total);
            }

            function updateQuantity(productId, change) {
                const item = state.cart.find((entry) => entry.id === productId);
                const product =
                    state.products.find((entry) => entry.id === productId) ||
                    item;
                if (!item || !product) return;
                const nextQuantity = item.quantity + change;
                if (nextQuantity <= 0) return removeFromCart(productId);
                if (nextQuantity > product.stock)
                    return showToast(
                        "Stock Limit",
                        "Quantity exceeds available stock.",
                        "warning",
                    );
                if (product.rx && item.rxNumber) {
                    const matchedRx = state.prescriptions.find(rx => rx.prescription_number.toLowerCase() === item.rxNumber.toLowerCase());
                    if (matchedRx) {
                        const rxItem = matchedRx.items ? matchedRx.items.find(it => it.product_id === productId) : null;
                        if (rxItem && nextQuantity > rxItem.quantity) {
                            return showToast(
                                "Prescription Limit",
                                `Cannot exceed the prescribed quantity of ${rxItem.quantity} for this prescription.`,
                                "error"
                            );
                        }
                    }
                }
                item.quantity = nextQuantity;
                updateCart();
            }

            function removeFromCart(productId) {
                state.cart = state.cart.filter((item) => item.id !== productId);
                updateCart();
            }
            function clearCart() {
                state.cart = [];
                updateCart();
                showToast(
                    "Cart Cleared",
                    "All items removed from cart.",
                    "warning",
                );
            }

            async function processCheckout() {
                if (!state.cart.length) return;
                const method = document.getElementById("paymentMethod").value;
                if (method === "M-Pesa") {
                    document.getElementById("mpesaPhone").value = "";
                    document.getElementById("mpesaModal").classList.add("active");
                    return;
                }
                await doCheckout();
            }

            async function doCheckout(paymentReference = "") {
                if (!state.cart.length) return;
                try {
                    const payload = {
                        payment_method:
                            document.getElementById("paymentMethod").value,
                        payment_reference: paymentReference,
                        items: state.cart.map((item) => ({
                            product_id: item.id,
                            quantity: item.quantity,
                            rx_number: item.rxNumber,
                            patient: item.patient,
                            doctor: item.doctor,
                        })),
                    };
                    const sale = await apiFetch("/api/sales", {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    state.lastReceiptSaleId = sale.sale_id;
                    await Promise.allSettled([
                        loadProducts(),
                        loadInventory(),
                        loadDashboard(),
                        loadReports(),
                        loadAnalytics(),
                        loadDatabaseOverview(),
                        loadPrescriptions(),
                        loadSales()
                    ]);
                    await loadReceipt(sale.sale_id);
                    renderProducts(
                        document
                            .getElementById("searchInput")
                            .value.trim()
                            .toLowerCase(),
                    );
                    renderInventory();
                    renderDashboard();
                    renderReports();
                    renderAnalytics();
                    renderDatabaseOverview();
                    renderDashboardFeeds();
                    document
                        .getElementById("checkoutModal")
                        .classList.add("active");
                } catch (error) {
                    showToast(
                        "Checkout Failed",
                        error.message || "Could not process transaction.",
                        "error",
                    );
                }
            }

            async function loadReceipt(saleId) {
                const receipt = await apiFetch(`/api/receipts/${saleId}`);
                document.getElementById("receiptContent").innerHTML =
                    renderReceiptHtml(receipt.sale);
            }

            function renderReceiptHtml(sale) {
                return `
                <div class="receipt-header">
                    <div class="receipt-title">MedPlus Pharmacy</div>
                    <div class="receipt-meta">Generated from live backend receipt endpoint</div>
                    <div class="receipt-meta" style="margin-top:8px;">${new Date(sale.created_at).toLocaleString()}</div>
                    <div class="receipt-meta">Transaction: ${escapeHtml(sale.transaction_id)}</div>
                    <div class="receipt-meta">Cashier: ${escapeHtml(sale.cashier_name || "N/A")}</div>
                </div>
                <div class="receipt-items">
                    ${sale.items.map((item) => `<div class="receipt-item"><span>${escapeHtml(item.product_name)} x${item.quantity}</span><span>${formatCurrency(item.price * item.quantity)}</span></div>`).join("")}
                </div>
                <div class="receipt-total"><span>Subtotal</span><span>${formatCurrency(sale.subtotal)}</span></div>
                <div class="receipt-total" style="font-weight:400; font-size:13px;"><span>Tax (8.5%)</span><span>${formatCurrency(sale.tax)}</span></div>
                <div class="receipt-total" style="font-size:18px; margin-top:8px;"><span>TOTAL</span><span>${formatCurrency(sale.total)}</span></div>
                <div class="receipt-footer"><p>Thank you for choosing MedPlus Pharmacy</p><p style="margin-top:4px;">Payment Method: ${escapeHtml(sale.payment_method)}</p></div>
            `;
            }

            function completeTransaction() {
                closeModal("checkoutModal");
                state.cart = [];
                updateCart();
                showToast("Transaction Complete", "New sale ready.", "success");
            }

            function selectPaymentMethod(method, btn) {
                document.getElementById("paymentMethod").value = method;
                document.querySelectorAll(".payment-method-btns .pay-btn").forEach(b => b.classList.remove("active"));
                btn.classList.add("active");
            }

            function toggleSidebar() {
                const sidebar = document.getElementById("sidebarEl");
                const overlay = document.getElementById("sidebarOverlay");
                sidebar.classList.toggle("open");
                overlay.classList.toggle("active");
            }

            async function confirmMpesaPayment() {
                const phone = document.getElementById("mpesaPhone").value.trim();
                if (!phone || phone.length < 10) {
                    showToast("Invalid Phone", "Please enter a valid M-Pesa phone number.", "warning");
                    return;
                }
                closeModal("mpesaModal");
                showToast("M-Pesa", `Initiating STK push to ${phone}...`, "success");
                
                const subtotal = state.cart.reduce(
                    (sum, item) => sum + item.price * item.quantity,
                    0,
                );
                const tax = subtotal * 0.085;
                const total = subtotal + tax;
                
                try {
                    const stkResponse = await apiFetch("/api/payments/mpesa/stk-push", {
                        method: "POST",
                        body: JSON.stringify({
                            phone: phone,
                            amount: total,
                            account_reference: "MedPlusPOS",
                            transaction_desc: "MedPlus POS Checkout"
                        })
                    });
                    
                    if (stkResponse.checkout_request_id) {
                        const checkoutId = stkResponse.checkout_request_id;
                        
                        // If the backend already confirmed payment (e.g. simulated/mock mode), skip polling
                        if (stkResponse.status === "completed") {
                            closeModal("mpesaModal");
                            showToast("M-Pesa Paid", "Payment confirmed! Completing transaction.", "success");
                            await doCheckout(checkoutId);
                            return;
                        }
                        
                        // Show loading/awaiting payment modal
                        closeModal("mpesaModal");
                        const loadingModal = document.getElementById("mpesaLoadingModal");
                        loadingModal.classList.add("active");
                        
                        let pollInterval = null;
                        let pollTimeout = null;
                        let consecutiveErrors = 0;
                        
                        const cleanup = () => {
                            if (pollInterval) clearInterval(pollInterval);
                            if (pollTimeout) clearTimeout(pollTimeout);
                            closeModal("mpesaLoadingModal");
                        };
                        
                        // Bind cancel button
                        const cancelBtn = document.getElementById("cancelMpesaPollBtn");
                        cancelBtn.onclick = async function() {
                            cleanup();
                            try {
                                await apiFetch(`/api/payments/mpesa/status/${checkoutId}/cancel`, {
                                    method: "POST"
                                });
                            } catch (err) {
                                console.error("Error cancelling transaction:", err);
                            }
                            showToast("M-Pesa Cancelled", "Payment request was cancelled by the operator.", "warning");
                        };
                        
                        // Start polling loop every 3 seconds
                        pollInterval = setInterval(async function() {
                            try {
                                const statusResp = await apiFetch(`/api/payments/mpesa/status/${checkoutId}`);
                                consecutiveErrors = 0; // Reset on success
                                if (statusResp.status === "completed") {
                                    cleanup();
                                    showToast("M-Pesa Paid", "Payment confirmed! Completing transaction.", "success");
                                    await doCheckout(checkoutId);
                                } else if (statusResp.status === "failed") {
                                    cleanup();
                                    showToast("M-Pesa Rejected", statusResp.result_desc || "Payment was rejected or cancelled.", "error");
                                }
                            } catch (err) {
                                consecutiveErrors++;
                                console.error(`Error polling payment status (attempt ${consecutiveErrors}):`, err);
                                if (consecutiveErrors >= 5) {
                                    cleanup();
                                    showToast("M-Pesa Error", "Lost connection while checking payment. Please verify manually.", "error");
                                }
                            }
                        }, 3000);
                        
                        // Timeout after 80 seconds
                        pollTimeout = setTimeout(async function() {
                            cleanup();
                            showToast("M-Pesa Timeout", "No response from customer or payment unconfirmed. Please verify manually.", "warning");
                        }, 80000);
                        
                    } else {
                        showToast("M-Pesa STK Failed", stkResponse.response_description || "STK push could not be initiated.", "error");
                    }
                } catch (error) {
                    showToast("M-Pesa STK Error", error.message || "Failed to trigger M-Pesa payment.", "error");
                }
            }

            async function confirmCardPayment() {
                const cardNum = document.getElementById("cardNumber").value.trim();
                const expiry = document.getElementById("cardExpiry").value.trim();
                const cvv = document.getElementById("cardCvv").value.trim();
                const holder = document.getElementById("cardHolder").value.trim();
                if (!cardNum || cardNum.length < 16) {
                    showToast("Invalid Card", "Please enter a valid card number.", "warning");
                    return;
                }
                if (!expiry || !cvv || !holder) {
                    showToast("Missing Details", "Please fill in all card fields.", "warning");
                    return;
                }
                closeModal("cardModal");
                showToast("Card Payment", "Processing card payment...", "success");
                await doCheckout();
            }

            function printReceipt() {
                window.print();
            }
            function closeModal(modalId) {
                document.getElementById(modalId).classList.remove("active");
            }

            function renderInventory() {
                const tbody = document.getElementById("inventoryTable");
                if (!state.inventory.length) {
                    tbody.innerHTML = `<tr><td colspan="7" class="empty-state">No inventory records available.</td></tr>`;
                    return;
                }
                tbody.innerHTML = state.inventory
                    .map((product) => {
                        const isArchived = product.status === "inactive";
                        const status = isArchived
                            ? "archived"
                            : product.stock > product.reorder_level
                              ? "active"
                              : product.stock > 0
                                ? "low"
                                : "expired";
                        const statusText = isArchived
                            ? "Archived"
                            : product.stock > product.reorder_level
                              ? "In Stock"
                              : product.stock > 0
                                ? "Low Stock"
                                : "Out/Critical";
                        const actions = isArchived
                            ? `<button class="restore-btn small-btn" onclick="restoreProduct(${product.id})"><i class="fas fa-undo"></i> Restore</button>`
                            : `<button class="small-btn" onclick="editProduct(${product.id})">Edit</button><button class="danger-btn small-btn" onclick="deleteProduct(${product.id})">Archive</button>`;
                        return `
                    <tr class="${isArchived ? 'archived-row' : ''}">
                        <td><strong>${escapeHtml(product.name)}</strong></td>
                        <td>${titleCase(product.category)}</td>
                        <td>${product.stock} units</td>
                        <td>${formatCurrency(product.price)}</td>
                        <td>${formatDate(product.expiry_date)}</td>
                        <td><span class="status-badge status-${status}">${statusText}</span></td>
                        <td><div class="inline-actions">${actions}</div></td>
                    </tr>
                `;
                    })
                    .join("");
            }

            function openProductCreateForm() {
                resetProductForm();
                const title = document.getElementById("productFormTitle");
                if (title) title.textContent = "Add New Product";
                const modal = document.getElementById("productModal");
                if (modal) modal.classList.add("active");
            }
            function resetProductForm() {
                const form = document.getElementById("productForm");
                if (form) form.reset();
                const idInput = document.getElementById("productId");
                if (idInput) idInput.value = "";
                const statusSelect = document.getElementById("productStatus");
                if (statusSelect) statusSelect.value = "active";
                const iconInput = document.getElementById("productIcon");
                if (iconInput) iconInput.value = "💊";
                const rxCheck = document.getElementById("productRx");
                if (rxCheck) rxCheck.checked = false;
            }
            function editProduct(id) {
                const product = state.inventory.find(
                    (entry) => entry.id === id,
                );
                if (!product) return;
                document.getElementById("productId").value = product.id;
                document.getElementById("productName").value = product.name || "";
                document.getElementById("productCategory").value = product.category || "otc";
                document.getElementById("productPrice").value = product.price;
                document.getElementById("productStock").value = product.stock;
                document.getElementById("productExpiryDate").value = (product.expiry_date || "").slice(0, 10);
                document.getElementById("productIcon").value = product.icon || "💊";
                document.getElementById("productStatus").value = product.status || "active";
                document.getElementById("productRx").checked = !!product.rx;
                const title = document.getElementById("productFormTitle");
                if (title) title.textContent = "Edit Product Profile";
                const modal = document.getElementById("productModal");
                if (modal) modal.classList.add("active");
            }

            async function submitProductForm(event) {
                event.preventDefault();
                const productId = document.getElementById("productId").value;
                const payload = {
                    name: document.getElementById("productName").value.trim(),
                    category: document.getElementById("productCategory").value,
                    price: Number(document.getElementById("productPrice").value || 0),
                    stock: Number(document.getElementById("productStock").value || 0),
                    expiry_date: document.getElementById("productExpiryDate").value,
                    icon: document.getElementById("productIcon").value.trim() || "💊",
                    status: document.getElementById("productStatus").value,
                    rx: document.getElementById("productRx").checked,
                };
                try {
                    if (productId) {
                        await apiFetch(`/api/products/${productId}`, {
                            method: "PUT",
                            body: JSON.stringify(payload),
                        });
                        showToast("Updated", "Product updated successfully.", "success");
                    } else {
                        await apiFetch("/api/admin/products", {
                            method: "POST",
                            body: JSON.stringify(payload),
                        });
                        showToast("Created", "Product created successfully.", "success");
                    }
                    closeModal("productModal");
                    resetProductForm();
                    await loadProducts();
                    await loadInventory();
                    renderProducts(
                        document.getElementById("searchInput")?.value.trim().toLowerCase() || "",
                    );
                    renderInventory();
                } catch (error) {
                    showToast("Product Error", error.message || "Unable to save product.", "error");
                }
            }

            async function deleteProduct(id) {
                if (!confirm("Archive this product? It will be hidden from the POS but remain in inventory.")) return;
                try {
                    await apiFetch(`/api/products/${id}`, { method: "DELETE" });
                    await loadProducts();
                    await loadInventory();
                    renderProducts(
                        document
                            .getElementById("searchInput")
                            .value.trim()
                            .toLowerCase(),
                    );
                    renderInventory();
                    showToast(
                        "Archived",
                        "Product archived. It is now hidden from POS sales.",
                        "success",
                    );
                } catch (error) {
                    showToast(
                        "Archive Failed",
                        error.message || "Unable to archive product.",
                        "error",
                    );
                }
            }

            async function restoreProduct(id) {
                if (!confirm("Restore this product? It will become active and visible in the POS again.")) return;
                try {
                    await apiFetch(`/api/products/${id}`, {
                        method: "PUT",
                        body: JSON.stringify({ status: "active" }),
                    });
                    await loadProducts();
                    await loadInventory();
                    renderProducts(
                        document
                            .getElementById("searchInput")
                            .value.trim()
                            .toLowerCase(),
                    );
                    renderInventory();
                    showToast(
                        "Restored",
                        "Product restored and is now active in the POS.",
                        "success",
                    );
                } catch (error) {
                    showToast(
                        "Restore Failed",
                        error.message || "Unable to restore product.",
                        "error",
                    );
                }
            }

            function renderDashboard() {
                const data = state.dashboard || {};
                document.getElementById("dashboardTodaySales").textContent =
                    formatCurrency(data.today_sales || 0);
                document.getElementById("dashboardTransactions").textContent =
                    data.transaction_count ?? 0;
                document.getElementById("dashboardPrescriptions").textContent =
                    data.prescription_count ?? 0;
                document.getElementById("dashboardLowStock").textContent =
                    data.low_stock_count ?? 0;
                const salesChange = document.getElementById(
                    "dashboardSalesChange",
                );
                salesChange.textContent = `${Number(data.sales_change || 0).toFixed(1)}% vs yesterday`;
                salesChange.className = `stat-change ${(data.sales_change || 0) >= 0 ? "positive" : "negative"}`;
                const txChange = document.getElementById("dashboardTxChange");
                txChange.textContent = `${data.tx_change || 0} vs yesterday`;
                txChange.className = `stat-change ${(data.tx_change || 0) >= 0 ? "positive" : "negative"}`;
                const rxChange = document.getElementById("dashboardRxChange");
                rxChange.textContent = `${data.rx_change || 0} vs yesterday`;
                rxChange.className = `stat-change ${(data.rx_change || 0) >= 0 ? "positive" : "negative"}`;
                renderBarChart(
                    "salesChart",
                    (state.analytics?.daily_sales || []).map((item) => ({
                        label: item.day?.slice(5) || item.day,
                        value: item.revenue || 0,
                        display: formatCurrency(item.revenue || 0),
                    })),
                );
                renderBarChart(
                    "topProductsChart",
                    (state.analytics?.category_breakdown || []).map((item) => ({
                        label: titleCase(item.category),
                        value: item.revenue || 0,
                        display: formatCurrency(item.revenue || 0),
                    })),
                );
            }

            function renderReports() {
                const summary = state.reports?.summary;
                const container = document.getElementById("reportsSummary");
                const topProducts =
                    document.getElementById("reportsTopProducts");
                if (!summary) {
                    container.innerHTML = `<div class="panel empty-state" style="grid-column:1/-1;">No report data available.</div>`;
                    topProducts.innerHTML = `<div class="empty-state">No top products available.</div>`;
                    return;
                }
                const cards = [
                    ["Revenue", formatCurrency(summary.revenue || 0)],
                    ["Transactions", summary.transactions || 0],
                    ["Units Sold", summary.units_sold || 0],
                    ["Prescriptions", summary.prescriptions || 0],
                    ["Low Stock Items", summary.low_stock_items || 0],
                    [
                        "Inventory Value",
                        formatCurrency(summary.inventory_value || 0),
                    ],
                ];
                container.innerHTML = cards
                    .map(
                        ([label, value]) =>
                            `<div class="kpi-card"><div class="stat-label">${label}</div><div class="stat-value" style="margin-top:8px;">${value}</div></div>`,
                    )
                    .join("");
                const items = state.reports?.top_products || [];
                topProducts.innerHTML = items.length
                    ? items
                          .map(
                              (item) =>
                                  `<div class="list-card"><strong>${escapeHtml(item.name)}</strong><div class="muted">Units sold: ${item.units} · Revenue: ${formatCurrency(item.revenue || 0)}</div></div>`,
                          )
                          .join("")
                    : `<div class="empty-state">No top products available.</div>`;
            }

            function renderAnalytics() {
                renderBarChart(
                    "analyticsDailyChart",
                    (state.analytics?.daily_sales || []).map((item) => ({
                        label: item.day?.slice(5) || item.day,
                        value: item.revenue || 0,
                        display: formatCurrency(item.revenue || 0),
                    })),
                );
                const container = document.getElementById(
                    "analyticsCategories",
                );
                const rows = state.analytics?.category_breakdown || [];
                container.innerHTML = rows.length
                    ? rows
                          .map(
                              (item) =>
                                  `<div class="list-card"><strong>${titleCase(item.category)}</strong><div class="muted">Units: ${item.units} · Revenue: ${formatCurrency(item.revenue || 0)}</div></div>`,
                          )
                          .join("")
                    : `<div class="empty-state">No analytics data available.</div>`;
            }

            function renderDatabaseOverview() {
                const container = document.getElementById(
                    "databaseOverviewCards",
                );
                const overview = state.databaseOverview?.overview || [];
                container.innerHTML = overview.length
                    ? overview
                          .map(
                              (item) =>
                                  `<div class="kpi-card"><div class="stat-label">${escapeHtml(item.label)}</div><div class="stat-value" style="margin-top:8px;">${typeof item.value === "number" && item.label.toLowerCase().includes("value") ? formatCurrency(item.value) : item.value}</div><div class="muted" style="margin-top:8px; font-size:13px;">${escapeHtml(item.description || "")}</div></div>`,
                          )
                          .join("")
                    : `<div class="panel empty-state" style="grid-column:1/-1;">No overview data available.</div>`;
                
                renderDatabaseTables();
            }

            function renderDatabaseTables() {
                const tbody = document.getElementById("databaseTablesBody");
                if (!tbody) return;
                const tables = state.databaseOverview?.tables || [];
                if (!tables.length) {
                    tbody.innerHTML = `<tr><td colspan="6" class="empty-state" style="text-align: center; padding: 20px;">No tables found.</td></tr>`;
                    return;
                }
                tbody.innerHTML = tables
                    .map(
                        (t) => `
                    <tr>
                        <td><strong>${escapeHtml(t.table_name)}</strong></td>
                        <td>${escapeHtml(t.description || "No description provided.")}</td>
                        <td><code>${t.row_count}</code></td>
                        <td><code>${t.column_count}</code></td>
                        <td><code>${escapeHtml(t.size_pretty || "0 B")}</code></td>
                        <td>
                            <button class="small-btn" onclick="showTableSchema('${escapeHtml(t.table_name)}')">
                                <i class="fas fa-eye"></i> View Schema
                            </button>
                        </td>
                    </tr>
                `
                    )
                    .join("");
            }

            function showTableSchema(tableName) {
                const tables = state.databaseOverview?.tables || [];
                const table = tables.find((t) => t.table_name === tableName);
                if (!table) return;

                document.getElementById("dbSchemaModalTitle").textContent = `Table: ${escapeHtml(table.table_name)}`;
                document.getElementById("dbSchemaModalDescription").textContent = escapeHtml(table.description || "No description provided.");

                const tbody = document.getElementById("dbSchemaModalBody");
                if (tbody) {
                    const columns = table.columns || [];
                    tbody.innerHTML = columns.length
                        ? columns
                              .map(
                                  (col) => `
                              <tr>
                                  <td><strong>${escapeHtml(col.name)}</strong></td>
                                  <td><code>${escapeHtml(col.type)}</code></td>
                                  <td><span class="status-badge ${col.nullable ? "status-expired" : "status-active"}" style="font-size: 11px; padding: 2px 6px;">${col.nullable ? "Yes" : "No"}</span></td>
                              </tr>
                          `
                              )
                              .join("")
                        : `<tr><td colspan="3" class="empty-state" style="text-align: center; padding: 10px;">No columns found.</td></tr>`;
                }

                document.getElementById("dbSchemaModal").classList.add("active");
            }

            async function unlockDatabaseOverview(event) {
                event.preventDefault();
                const password = document.getElementById("dbUnlockPassword").value;
                try {
                    const response = await apiFetch("/api/database-overview/access", {
                        method: "POST",
                        body: JSON.stringify({ password }),
                    });
                    state.databaseAccessToken = response.token;
                    localStorage.setItem("medplus_db_access_token", response.token);
                    document.getElementById("dbUnlockPassword").value = "";
                    showToast("Access Granted", "Database overview unlocked.", "success");
                    await loadDatabaseOverview();
                } catch (error) {
                    showToast("Access Denied", error.message || "Invalid password / PIN.", "error");
                }
            }

            function renderUsers() {
                const tbody = document.getElementById("usersTable");
                if (!tbody) return;
                if (!state.users.length) {
                    tbody.innerHTML = `<tr><td colspan="5" class="empty-state">Admin access unavailable or no users found.</td></tr>`;
                    return;
                }
                tbody.innerHTML = state.users
                    .map(
                        (user) => `
                <tr>
                    <td><strong>${escapeHtml(user.full_name)}</strong><br><span style="font-size:12px; color:#94a3b8;">@${escapeHtml(user.username)}</span></td>
                    <td><span class="badge" style="background:rgba(59,130,246,0.15); color:#3b82f6;">${escapeHtml(user.role)}</span></td>
                    <td><span class="badge ${user.status === "active" ? "success" : "danger"}">${escapeHtml(user.status)}</span></td>
                    <td>${formatDate(user.created_at)}</td>
                    <td>
                        <div class="inline-actions" style="gap: 6px;">
                            <button class="small-btn" onclick="editUser(${user.id})"><i class="fas fa-edit"></i> Edit</button>
                            ${user.status === "active"
                                ? `<button class="secondary-btn small-btn" onclick="toggleUserStatus(${user.id}, 'inactive')"><i class="fas fa-ban"></i> Deactivate</button>`
                                : `<button class="secondary-btn small-btn" style="color:#10b981; border-color:rgba(16,185,129,0.3);" onclick="toggleUserStatus(${user.id}, 'active')"><i class="fas fa-check"></i> Activate</button>`
                            }
                            <button class="danger-btn small-btn" onclick="deleteUser(${user.id})"><i class="fas fa-trash-alt"></i> Delete</button>
                        </div>
                    </td>
                </tr>
            `,
                    )
                    .join("");
            }

            function openUserCreateForm() {
                resetUserForm();
                showView("admin");
            }
            function resetUserForm() {
                document.getElementById("userForm").reset();
                document.getElementById("userId").value = "";
                const roleInput = document.getElementById("userRoleInput");
                roleInput.innerHTML = '<option value="Pharmacist">Pharmacist</option>';
                roleInput.value = "Pharmacist";
                document.getElementById("userStatusInput").value = "active";
            }
            function editUser(id) {
                const user = state.users.find((entry) => entry.id === id);
                if (!user) return;
                document.getElementById("userId").value = user.id;
                document.getElementById("userFullName").value =
                    user.full_name || "";
                document.getElementById("userUsername").value =
                    user.username || "";
                document.getElementById("userPassword").value = "";
                
                const roleInput = document.getElementById("userRoleInput");
                roleInput.innerHTML = "";
                if (user.role === "Admin") {
                    roleInput.innerHTML = '<option value="Admin">Admin</option>';
                } else {
                    roleInput.innerHTML = '<option value="Pharmacist">Pharmacist</option>';
                }
                
                document.getElementById("userRoleInput").value =
                    user.role || "Pharmacist";
                document.getElementById("userStatusInput").value =
                    user.status || "active";
                showView("admin");
            }

            async function submitUserForm(event) {
                event.preventDefault();
                const userId = document.getElementById("userId").value;
                const status = document.getElementById("userStatusInput").value || "active";
                const payload = {
                    full_name: document
                        .getElementById("userFullName")
                        .value.trim(),
                    username: document
                        .getElementById("userUsername")
                        .value.trim(),
                    password: document.getElementById("userPassword").value,
                    role: document.getElementById("userRoleInput").value,
                    status: status,
                };
                try {
                    if (userId) {
                        await apiFetch(
                            `/api/users/${userId}?status=${encodeURIComponent(status)}`,
                            { method: "PUT", body: JSON.stringify(payload) },
                        );
                        showToast(
                            "Updated",
                            "User updated successfully.",
                            "success",
                        );
                    } else {
                        if (!payload.password)
                            return showToast(
                                "Validation",
                                "Password is required for new users.",
                                "warning",
                            );
                        await apiFetch("/api/users", {
                            method: "POST",
                            body: JSON.stringify(payload),
                        });
                        showToast(
                            "Created",
                            "User created successfully.",
                            "success",
                        );
                    }
                    resetUserForm();
                    await loadUsersSafe();
                } catch (error) {
                    showToast(
                        "User Error",
                        error.message || "Unable to save user.",
                        "error",
                    );
                }
            }

            async function toggleUserStatus(id, newStatus) {
                const actionText = newStatus === "active" ? "activate" : "deactivate";
                if (!confirm(`Are you sure you want to ${actionText} this user?`)) return;
                try {
                    const user = state.users.find((u) => u.id === id);
                    const payload = {
                        full_name: user ? user.full_name : "",
                        username: user ? user.username : "",
                        role: user ? user.role : "Pharmacist",
                        status: newStatus,
                    };
                    await apiFetch(`/api/users/${id}`, {
                        method: "PUT",
                        body: JSON.stringify(payload),
                    });
                    await loadUsersSafe();
                    showToast("Status Updated", `User ${actionText}d successfully.`, "success");
                } catch (error) {
                    showToast("User Error", error.message || `Unable to ${actionText} user.`, "error");
                }
            }

            async function deleteUser(id) {
                const user = state.users.find(u => u.id === id);
                const userName = user ? user.username : `User #${id}`;
                if (!confirm(`Are you sure you want to PERMANENTLY DELETE user '${userName}'? This action cannot be undone.`)) return;
                try {
                    await apiFetch(`/api/users/${id}`, { method: "DELETE" });
                    await loadUsersSafe();
                    showToast("User Removed", "User deleted successfully.", "success");
                } catch (error) {
                    showToast("User Error", error.message || "Unable to delete user.", "error");
                }
            }

            function renderSupplierOptions() {
                const select = document.getElementById("poSupplierId");
                const options = state.suppliers.length
                    ? state.suppliers
                          .map(
                              (s) =>
                                  `<option value="${s.id}">${escapeHtml(s.name)}</option>`,
                          )
                          .join("")
                    : '<option value="">No suppliers available</option>';
                select.innerHTML = options;
            }

            function resetSupplierForm() {
                document.getElementById("supplierForm").reset();
                document.getElementById("supplierId").value = "";
            }
            function editSupplier(id) {
                const supplier = state.suppliers.find(
                    (entry) => entry.id === id,
                );
                if (!supplier) return;
                document.getElementById("supplierId").value = supplier.id;
                document.getElementById("supplierName").value =
                    supplier.name || "";
                document.getElementById("supplierContact").value =
                    supplier.contact_person || "";
                document.getElementById("supplierPhone").value =
                    supplier.phone || "";
                document.getElementById("supplierEmail").value =
                    supplier.email || "";
                document.getElementById("supplierAddress").value =
                    supplier.address || "";
                showView("supply");
            }

            async function submitSupplierForm(event) {
                event.preventDefault();
                const supplierId = document.getElementById("supplierId").value;
                const payload = {
                    name: document.getElementById("supplierName").value.trim(),
                    contact_person: document
                        .getElementById("supplierContact")
                        .value.trim(),
                    phone: document
                        .getElementById("supplierPhone")
                        .value.trim(),
                    email: document
                        .getElementById("supplierEmail")
                        .value.trim(),
                    address: document
                        .getElementById("supplierAddress")
                        .value.trim(),
                    status: "active",
                };
                try {
                    if (supplierId) {
                        await apiFetch(`/api/suppliers/${supplierId}`, {
                            method: "PUT",
                            body: JSON.stringify(payload),
                        });
                        showToast(
                            "Updated",
                            "Supplier updated successfully.",
                            "success",
                        );
                    } else {
                        await apiFetch("/api/suppliers", {
                            method: "POST",
                            body: JSON.stringify(payload),
                        });
                        showToast(
                            "Created",
                            "Supplier created successfully.",
                            "success",
                        );
                    }
                    resetSupplierForm();
                    await loadSuppliers();
                } catch (error) {
                    showToast(
                        "Supplier Error",
                        error.message || "Unable to save supplier.",
                        "error",
                    );
                }
            }

            async function archiveSupplier(id) {
                if (!confirm("Archive this supplier?")) return;
                try {
                    await apiFetch(`/api/suppliers/${id}`, {
                        method: "DELETE",
                    });
                    await loadSuppliers();
                    showToast(
                        "Archived",
                        "Supplier archived successfully.",
                        "success",
                    );
                } catch (error) {
                    showToast(
                        "Supplier Error",
                        error.message || "Unable to archive supplier.",
                        "error",
                    );
                }
            }

            function renderSuppliers() {
                const container = document.getElementById("suppliersList");
                container.innerHTML = state.suppliers.length
                    ? state.suppliers
                          .map(
                              (supplier) => `
                <div class="list-card">
                    <strong>${escapeHtml(supplier.name)}</strong>
                    <div class="muted">${escapeHtml(supplier.contact_person || "No contact")} · ${escapeHtml(supplier.phone || "No phone")}</div>
                    <div class="muted">${escapeHtml(supplier.email || "")}</div>
                    <div class="inline-actions" style="margin-top:10px;"><button class="small-btn" onclick="editSupplier(${supplier.id})">Edit</button><button class="danger-btn small-btn" onclick="archiveSupplier(${supplier.id})">Archive</button></div>
                </div>
            `,
                          )
                          .join("")
                    : `<div class="empty-state">No suppliers available.</div>`;
            }

            function addPoItemRow(productId = "", quantity = 10, unitCost = "") {
                const container = document.getElementById("poItemsContainer");
                if (!container) return;

                const optionsHtml = state.products
                    .map(
                        (p) =>
                            `<option value="${p.id}" ${String(p.id) === String(productId) ? "selected" : ""}>${escapeHtml(p.name)} (Stock: ${p.stock}) - Ksh ${p.price}</option>`,
                    )
                    .join("");

                const rowId = "po_row_" + Date.now() + "_" + Math.random().toString(36).substr(2, 4);
                const row = document.createElement("div");
                row.className = "po-item-row";
                row.id = rowId;
                row.style.cssText = "display: flex; gap: 8px; align-items: center;";

                row.innerHTML = `
                    <select class="form-select po-product-select" style="flex: 2; font-size: 13px;">
                        <option value="">-- Select Medicine --</option>
                        ${optionsHtml}
                    </select>
                    <input type="number" class="form-input po-qty-input" placeholder="Qty" min="1" value="${quantity}" style="width: 85px; font-size: 13px;" />
                    <input type="number" class="form-input po-cost-input" placeholder="Unit Cost" step="0.01" value="${unitCost}" style="width: 105px; font-size: 13px;" />
                    <button type="button" class="secondary-btn" style="padding: 6px 10px; color: #ef4444; border-color: rgba(239,68,68,0.3);" onclick="document.getElementById('${rowId}').remove()"><i class="fas fa-trash-alt"></i></button>
                `;

                container.appendChild(row);

                const selectEl = row.querySelector(".po-product-select");
                const costEl = row.querySelector(".po-cost-input");
                selectEl.addEventListener("change", function () {
                    const selId = Number(this.value);
                    const prod = state.products.find((p) => p.id === selId);
                    if (prod && !costEl.value) {
                        costEl.value = Math.round(prod.price * 0.75 * 100) / 100;
                    }
                });

                if (productId && !unitCost) {
                    const prod = state.products.find((p) => String(p.id) === String(productId));
                    if (prod) {
                        costEl.value = Math.round(prod.price * 0.75 * 100) / 100;
                    }
                }
            }

            function collectPoItems() {
                const container = document.getElementById("poItemsContainer");
                if (!container) return [];
                const rows = container.querySelectorAll(".po-item-row");
                const items = [];
                rows.forEach((row) => {
                    const productId = Number(row.querySelector(".po-product-select").value);
                    const quantity = Number(row.querySelector(".po-qty-input").value);
                    const unitCost = Number(row.querySelector(".po-cost-input").value);
                    if (productId && quantity > 0) {
                        items.push({
                            product_id: productId,
                            quantity: quantity,
                            unit_cost: unitCost || 0,
                        });
                    }
                });
                return items;
            }

            async function submitPurchaseOrderForm(event) {
                event.preventDefault();
                try {
                    const supplierId = Number(document.getElementById("poSupplierId").value);
                    if (!supplierId) {
                        return showToast("Select Supplier", "Please select a supplier.", "warning");
                    }
                    const items = collectPoItems();
                    if (!items.length) {
                        return showToast("Select Products", "Please add at least one medicine item to the purchase order.", "warning");
                    }
                    const payload = {
                        supplier_id: supplierId,
                        notes: document.getElementById("poNotes").value.trim(),
                        items,
                    };
                    await apiFetch("/api/purchase-orders", {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    document.getElementById("purchaseOrderForm").reset();
                    document.getElementById("poItemsContainer").innerHTML = "";
                    await loadPurchaseOrders();
                    showToast("Order Created", "Purchase order created successfully.", "success");
                } catch (error) {
                    showToast("PO Error", error.message || "Unable to create purchase order.", "error");
                }
            }

            function renderPurchaseOrders() {
                const container = document.getElementById("purchaseOrdersList");
                container.innerHTML = state.purchaseOrders.length
                    ? state.purchaseOrders
                          .map(
                              (po) => `
                <div class="list-card">
                    <strong>${escapeHtml(po.po_number)}</strong>
                    <div class="muted">Supplier: ${escapeHtml(po.supplier_name || "")}</div>
                    <div class="muted">Status: ${escapeHtml(po.status)} · Ordered: ${formatDate(po.ordered_at)}</div>
                    <div class="inline-actions" style="margin-top:10px;">
                        <button class="small-btn" onclick="viewPurchaseOrder(${po.id})"><i class="fas fa-eye"></i> View Details</button>
                        ${po.status !== "received" ? `<button class="primary-btn small-btn" onclick="receivePurchaseOrder(${po.id})"><i class="fas fa-box"></i> Receive Stock</button>` : `<span class="status-badge" style="background:rgba(16,185,129,0.1); color:#10b981; padding:4px 8px; font-size:12px;">Received</span>`}
                    </div>
                </div>
            `,
                          )
                          .join("")
                    : `<div class="empty-state">No purchase orders available.</div>`;
            }

            async function viewPurchaseOrder(id) {
                try {
                    const po = await apiFetch(`/api/purchase-orders/${id}`);
                    document.getElementById("poDetailNumber").textContent = `Purchase Order ${po.po_number}`;

                    const statusClass = po.status === "received" ? "success" : po.status === "cancelled" ? "danger" : "warning";
                    const statusText = titleCase(po.status);
                    document.getElementById("poDetailStatusBadge").innerHTML = `<span class="badge ${statusClass}">${statusText}</span>`;

                    document.getElementById("poDetailSupplier").innerHTML = `
                        <div><strong>${escapeHtml(po.supplier_name || "Supplier #" + po.supplier_id)}</strong></div>
                        ${po.supplier_contact ? `<div>Contact: ${escapeHtml(po.supplier_contact)}</div>` : ""}
                        ${po.supplier_phone ? `<div>Phone: ${escapeHtml(po.supplier_phone)}</div>` : ""}
                    `;

                    document.getElementById("poDetailMeta").innerHTML = `
                        <div>Date Created: ${formatDate(po.ordered_at)}</div>
                        ${po.received_at ? `<div>Date Received: ${formatDate(po.received_at)}</div>` : ""}
                        <div>Notes: ${escapeHtml(po.notes || "None")}</div>
                    `;

                    let totalValue = 0;
                    const itemsHtml = (po.items || []).map((item, index) => {
                        const prod = state.products.find(p => p.id === item.product_id);
                        const prodName = item.product_name || (prod ? prod.name : "Product #" + item.product_id);
                        const unitCost = Number(item.unit_cost || 0);
                        const qty = Number(item.quantity || 0);
                        const subtotal = qty * unitCost;
                        totalValue += subtotal;
                        return `
                            <tr>
                                <td>${index + 1}</td>
                                <td><strong>${escapeHtml(prodName)}</strong></td>
                                <td>${qty}</td>
                                <td>${formatCurrency(unitCost)}</td>
                                <td><strong>${formatCurrency(subtotal)}</strong></td>
                            </tr>
                        `;
                    }).join("");

                    document.getElementById("poDetailItemsBody").innerHTML = itemsHtml || `<tr><td colspan="5" style="text-align:center;">No line items found.</td></tr>`;
                    document.getElementById("poDetailTotal").textContent = formatCurrency(totalValue);

                    const actionsContainer = document.getElementById("poDetailActions");
                    if (po.status !== "received") {
                        actionsContainer.innerHTML = `
                            <button class="modal-btn secondary" onclick="closeModal('poDetailsModal')">Close</button>
                            <button class="modal-btn primary" onclick="closeModal('poDetailsModal'); receivePurchaseOrder(${po.id});"><i class="fas fa-box"></i> Receive Stock</button>
                        `;
                    } else {
                        actionsContainer.innerHTML = `
                            <button class="modal-btn secondary" onclick="closeModal('poDetailsModal')">Close</button>
                        `;
                    }

                    document.getElementById("poDetailsModal").classList.add("active");
                } catch (error) {
                    showToast("PO Error", error.message || "Unable to load purchase order details.", "error");
                }
            }

            async function receivePurchaseOrder(id) {
                try {
                    await apiFetch(`/api/purchase-orders/${id}/receive`, {
                        method: "POST",
                    });
                    await loadPurchaseOrders();
                    await loadProducts();
                    await loadInventory();
                    renderProducts(
                        document
                            .getElementById("searchInput")
                            .value.trim()
                            .toLowerCase(),
                    );
                    renderInventory();
                    showToast(
                        "Stock Received",
                        "Purchase order received and inventory updated.",
                        "success",
                    );
                } catch (error) {
                    showToast(
                        "Receive Error",
                        error.message || "Unable to receive stock.",
                        "error",
                    );
                }
            }

            async function createPurchaseOrderFromLowStock() {
                if (!state.suppliers.length)
                    return showToast("Supplier Required", "Create a supplier first.", "warning");
                const lowStockItems = state.inventory
                    .filter((product) => product.stock <= product.reorder_level)
                    .slice(0, 10);
                if (!lowStockItems.length)
                    return showToast("No Action Needed", "No low stock items currently require restocking.", "success");

                document.getElementById("poSupplierId").value = state.suppliers[0].id;
                document.getElementById("poNotes").value = "Auto-generated restock request for low stock items";

                const container = document.getElementById("poItemsContainer");
                container.innerHTML = "";
                lowStockItems.forEach(item => {
                    const restockQty = Math.max(item.reorder_level * 2 - item.stock, 10);
                    const unitCost = Math.round(item.price * 0.75 * 100) / 100;
                    addPoItemRow(item.id, restockQty, unitCost);
                });

                showToast("Draft Prepared", "Low-stock purchase order draft prepared with " + lowStockItems.length + " items. Review and submit.", "success");
                showView("supply");
            }

            function renderBarChart(elementId, rows) {
                const el = document.getElementById(elementId);
                if (!el) return;
                if (!rows.length) {
                    el.innerHTML = `<div class="empty-state" style="width:100%;">No chart data available.</div>`;
                    return;
                }
                const max = Math.max(
                    ...rows.map((row) => Number(row.value) || 0),
                    1,
                );
                el.innerHTML = rows
                    .map(
                        (row) => `
                <div class="bar-wrapper">
                    <div class="bar-value">${escapeHtml(String(row.display ?? row.value))}</div>
                    <div class="bar" style="height:${Math.max(((Number(row.value) || 0) / max) * 160, 8)}px;"></div>
                    <div class="bar-label">${escapeHtml(String(row.label))}</div>
                </div>
            `,
                    )
                    .join("");
            }

            function defaultApiBaseUrl() {
                if (window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1") {
                    return "";
                }
                return localStorage.getItem("medplus_api_base_url") || "";
            }

            function canAccessView(view) {
                const role = state.user?.role || "";
                const adminViews = ["admin", "database"];
                const pharmacistViews = [
                    "inventory",
                    "reports",
                    "analytics",
                    "supply",
                ];
                if (adminViews.includes(view)) return role === "Admin";
                if (pharmacistViews.includes(view))
                    return role === "Admin" || role === "Pharmacist";
                return true;
            }

            function applyRolePermissions() {
                document
                    .querySelectorAll(".nav-item[data-view]")
                    .forEach((item) => {
                        const view = item.getAttribute("data-view");
                        item.classList.toggle("hidden", !canAccessView(view));
                    });
            }

            function formatCurrency(value) {
                return "Ksh " + Math.round(Number(value || 0)).toLocaleString("en-US", {
                    minimumFractionDigits: 0,
                    maximumFractionDigits: 0,
                });
            }
            function formatDate(value) {
                if (!value) return "—";
                const date = new Date(value);
                return Number.isNaN(date.getTime())
                    ? String(value)
                    : date.toLocaleDateString("en-US", {
                          month: "short",
                          day: "numeric",
                          year: "numeric",
                      });
            }
            function titleCase(value) {
                return String(value || "")
                    .replace(/_/g, " ")
                    .replace(/\b\w/g, (c) => c.toUpperCase());
            }
            function escapeHtml(value) {
                return String(value ?? "").replace(
                    /[&<>"']/g,
                    (char) =>
                        ({
                            "&": "&amp;",
                            "<": "&lt;",
                            ">": "&gt;",
                            '"': "&quot;",
                            "'": "&#39;",
                        })[char],
                );
            }

            function showToast(title, message, type = "success") {
                const container = document.getElementById("toastContainer");
                if (container) {
                    container.innerHTML = "";
                }
                const toast = document.createElement("div");
                toast.className = `toast ${type}`;
                const icon =
                    type === "success"
                        ? "fa-check-circle"
                        : type === "warning"
                          ? "fa-exclamation-triangle"
                          : "fa-times-circle";
                toast.innerHTML = `<i class="fas ${icon} toast-icon"></i><div class="toast-content"><div class="toast-title">${escapeHtml(title)}</div><div class="toast-message">${escapeHtml(message)}</div></div>`;
                if (container) {
                    container.appendChild(toast);
                }
                setTimeout(() => {
                    toast.style.opacity = "0";
                    toast.style.transform = "translateX(100%)";
                    setTimeout(() => toast.remove(), 300);
                }, 6000);
            }
