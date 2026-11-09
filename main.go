package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	db        *pgxpool.Pool
	jwtSecret []byte
}

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Name string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found; using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}

	app := &App{
		db:        db,
		jwtSecret: []byte(secret),
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logging)

	r.Get("/health", app.health)

	r.Post("/users", app.createUser)
	r.Get("/users", app.listUsers)
	r.Get("/users/{id}", app.getUser)
	r.Put("/users/{id}", app.updateUser)
	r.Delete("/users/{id}", app.deleteUser)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", app.login)
	})

	r.Group(func(r chi.Router) {
		r.Use(app.auth)
		r.Get("/me", app.me)
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting API on http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (a *App) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest

	if err := decodeJSON(w, r, &req); err != nil {
		log.Printf("decode create user request: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" ||
		!strings.Contains(req.Email, "@") ||
		len(req.Password) < 8 {
		writeError(
			w,
			http.StatusBadRequest,
			"name and valid email are required; password must be at least 8 characters",
		)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not secure password")
		return
	}

	var u User

	err = a.db.QueryRow(r.Context(), `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, created_at, updated_at`,
		req.Name,
		req.Email,
		string(hash),
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") ||
			strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}

		log.Printf("create user: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	writeJSON(w, http.StatusCreated, u)
}

func (a *App) listUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseBoundedInt(r.URL.Query().Get("limit"), 20, 1, 100)
	offset := parseBoundedInt(
		r.URL.Query().Get("offset"),
		0,
		0,
		1_000_000,
	)

	rows, err := a.db.Query(r.Context(), `
		SELECT id, name, email, created_at, updated_at
		FROM users
		ORDER BY id
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		log.Printf("list users: %v", err)
		writeError(w, http.StatusInternalServerError, "could not list users")
		return
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var u User

		if err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			log.Printf("scan user: %v", err)
			writeError(w, http.StatusInternalServerError, "could not read users")
			return
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		log.Printf("iterate users: %v", err)
		writeError(w, http.StatusInternalServerError, "could not read users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   users,
		"limit":  limit,
		"offset": offset,
	})
}

func (a *App) getUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	u, err := findUser(r.Context(), a.db, id)

	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err != nil {
		log.Printf("get user: %v", err)
		writeError(w, http.StatusInternalServerError, "could not get user")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (a *App) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req updateUserRequest

	if err := decodeJSON(w, r, &req); err != nil {
		log.Printf("decode update user request: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var u User

	err := a.db.QueryRow(r.Context(), `
		UPDATE users
		SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, email, created_at, updated_at`,
		req.Name,
		id,
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err != nil {
		log.Printf("update user: %v", err)
		writeError(w, http.StatusInternalServerError, "could not update user")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	tag, err := a.db.Exec(
		r.Context(),
		"DELETE FROM users WHERE id = $1",
		id,
	)
	if err != nil {
		log.Printf("delete user: %v", err)
		writeError(w, http.StatusInternalServerError, "could not delete user")
		return
	}

	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := decodeJSON(w, r, &req); err != nil {
		log.Printf("decode login request: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var id int64
	var hash string

	err := a.db.QueryRow(
		r.Context(),
		"SELECT id, password_hash FROM users WHERE email = $1",
		email,
	).Scan(&id, &hash)

	if err != nil ||
		bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})

	signed, err := token.SignedString(a.jwtSecret)
	if err != nil {
		log.Printf("create token: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token":      signed,
		"token_type": "Bearer",
	})
}

func (a *App) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		parts := strings.SplitN(header, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		parsed, err := jwt.ParseWithClaims(
			parts[1],
			&claims{},
			func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("unexpected signing method")
				}
				return a.jwtSecret, nil
			},
		)

		if err != nil || !parsed.Valid {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c, ok := parsed.Claims.(*claims)
		if !ok || c.UserID < 1 {
			writeError(w, http.StatusUnauthorized, "invalid token claims")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, c.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type userIDKey struct{}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value(userIDKey{}).(int64)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	u, err := findUser(r.Context(), a.db, id)

	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if err != nil {
		log.Printf("get profile: %v", err)
		writeError(w, http.StatusInternalServerError, "could not get profile")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findUser(ctx context.Context, db queryer, id int64) (User, error) {
	var u User

	err := db.QueryRow(ctx, `
		SELECT id, name, email, created_at, updated_at
		FROM users
		WHERE id = $1`,
		id,
	).Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	return u, err
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}

	return id, true
}

func parseBoundedInt(raw string, fallback, min, max int) int {
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n < min {
		return fallback
	}

	if n > max {
		return max
	}

	return n
}

// decodeJSON reads exactly one JSON value and rejects unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB maximum

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the requested JSON object.
	if err := dec.Decode(dst); err != nil {
		return err
	}

	// Reject a second JSON value or any trailing non-whitespace data.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}

	// FIX: the JSON is valid and contains exactly one value.
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf(
			"%s %s %s",
			r.Method,
			r.URL.Path,
			time.Since(start),
		)
	})
}
