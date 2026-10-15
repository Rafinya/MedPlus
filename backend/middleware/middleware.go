package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	JWTSecret      []byte
	allowedOrigins = []string{"*"}
	appEnv         = "development"
	rateLimitMu    sync.Mutex
	rateLimiters   = map[string]*rateWindow{}
)

type rateWindow struct {
	Count      int
	WindowEnds time.Time
}

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// UserClaims defines JWT token payload structure
type UserClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func Configure(jwtSecret, origins, env string) error {
	appEnv = strings.ToLower(strings.TrimSpace(env))
	if appEnv == "" {
		appEnv = "development"
	}

	if strings.TrimSpace(jwtSecret) == "" {
		if appEnv == "production" {
			return fmt.Errorf("JWT_SECRET is required in production")
		}
		jwtSecret = "dev-only-secret-change-me"
	}
	JWTSecret = []byte(jwtSecret)

	if strings.TrimSpace(origins) == "" {
		if appEnv == "production" {
			return fmt.Errorf("ALLOWED_ORIGINS is required in production")
		}
		allowedOrigins = []string{"*"}
		return nil
	}

	parsed := []string{}
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			parsed = append(parsed, origin)
		}
	}
	if len(parsed) == 0 {
		return fmt.Errorf("ALLOWED_ORIGINS must contain at least one origin")
	}
	allowedOrigins = parsed
	return nil
}

// GenerateToken generates a signed JWT token for a user
func GenerateToken(userID int, username, role string) (string, error) {
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(startedAt))
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered while serving %s %s: %v", r.Method, r.URL.Path, recovered)
				writeJSONError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RateLimit(name string, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := name + ":" + clientIP(r)
			now := time.Now()

			rateLimitMu.Lock()
			for existingKey, existingWindow := range rateLimiters {
				if now.After(existingWindow.WindowEnds) {
					delete(rateLimiters, existingKey)
				}
			}

			entry, exists := rateLimiters[key]
			if !exists || now.After(entry.WindowEnds) {
				rateLimiters[key] = &rateWindow{Count: 1, WindowEnds: now.Add(window)}
				rateLimitMu.Unlock()
				next.ServeHTTP(w, r)
				return
			}
			if entry.Count >= maxRequests {
				retryAfter := fmt.Sprintf("%.0f", time.Until(entry.WindowEnds).Seconds())
				rateLimitMu.Unlock()
				w.Header().Set("Retry-After", retryAfter)
				writeJSONError(w, http.StatusTooManyRequests, "Too many requests, please try again later")
				return
			}
			entry.Count++
			rateLimitMu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	forwardedFor := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if forwardedFor != "" {
		return forwardedFor
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if allowOrigin(origin) {
			if origin == "" || allowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Database-Access-Token")
		w.Header().Set("Access-Control-Allow-Private-Network", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func allowOrigin(origin string) bool {
	if appEnv == "development" {
		return true
	}
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
		return true
	}
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// RequireAuth checks if request contains a valid JWT token
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeJSONError(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		tokenStr := parts[1]
		claims := &UserClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			writeJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRoles checks if user claims role matches allowed roles
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*UserClaims)
			if !ok || claims == nil {
				writeJSONError(w, http.StatusUnauthorized, "Access denied: unauthorized")
				return
			}

			roleAllowed := false
			for _, role := range allowedRoles {
				if strings.EqualFold(claims.Role, role) || strings.EqualFold(claims.Role, "Pharmacist") || strings.EqualFold(claims.Role, "Admin") {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				writeJSONError(w, http.StatusForbidden, "Access denied: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":%q}`, message)))
}
