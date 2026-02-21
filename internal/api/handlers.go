package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/logging"
	"github.com/mwhite7112/woodpantry-ingredients/internal/service"
)

// NewRouter wires up all routes with the provided Service.
func NewRouter(svc *service.Service) http.Handler {
	r := chi.NewRouter()
	r.Use(logging.Middleware)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handleHealth)

	r.Get("/ingredients", handleListIngredients(svc))
	r.Post("/ingredients", handleCreateIngredient(svc))
	r.Post("/ingredients/resolve", handleResolve(svc))
	r.Post("/ingredients/merge", handleMerge(svc))
	r.Get("/ingredients/{id}", handleGetIngredient(svc))
	r.Put("/ingredients/{id}", handleUpdateIngredient(svc))

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok")) //nolint:errcheck
}

// --- list ---

func handleListIngredients(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.Queries().ListIngredients(r.Context())
		if err != nil {
			jsonError(w, "failed to list ingredients", http.StatusInternalServerError, err)
			return
		}
		if items == nil {
			items = []db.Ingredient{}
		}
		jsonOK(w, items)
	}
}

// --- create ---

type createIngredientRequest struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	Category    string   `json:"category"`
	DefaultUnit string   `json:"default_unit"`
}

func handleCreateIngredient(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createIngredientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}
		aliases := req.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		ing, err := svc.Queries().CreateIngredient(r.Context(), db.CreateIngredientParams{
			Name:        service.Normalize(req.Name),
			Aliases:     aliases,
			Category:    nullString(req.Category),
			DefaultUnit: nullString(req.DefaultUnit),
		})
		if err != nil {
			jsonError(w, "failed to create ingredient", http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ing) //nolint:errcheck
	}
}

// --- get ---

func handleGetIngredient(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(w, "invalid id", http.StatusBadRequest)
			return
		}
		ing, err := svc.Queries().GetIngredient(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(w, "failed to get ingredient", http.StatusInternalServerError, err)
			return
		}
		jsonOK(w, ing)
	}
}

// --- update ---

type updateIngredientRequest struct {
	Aliases     []string `json:"aliases"`
	Category    string   `json:"category"`
	DefaultUnit string   `json:"default_unit"`
}

func handleUpdateIngredient(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(w, "invalid id", http.StatusBadRequest)
			return
		}
		var req updateIngredientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		aliases := req.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		ing, err := svc.Queries().UpdateIngredient(r.Context(), db.UpdateIngredientParams{
			ID:          id,
			Aliases:     aliases,
			Category:    nullString(req.Category),
			DefaultUnit: nullString(req.DefaultUnit),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(w, "failed to update ingredient", http.StatusInternalServerError, err)
			return
		}
		jsonOK(w, ing)
	}
}

// --- resolve ---

type resolveRequest struct {
	Name string `json:"name"`
}

type resolveResponse struct {
	Ingredient db.Ingredient `json:"ingredient"`
	Confidence float64       `json:"confidence"`
	Created    bool          `json:"created"`
}

func handleResolve(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resolveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			jsonError(w, "name is required", http.StatusBadRequest)
			return
		}
		result, err := svc.Resolve(r.Context(), req.Name)
		if err != nil {
			jsonError(w, "resolve failed", http.StatusInternalServerError, err)
			return
		}
		status := http.StatusOK
		if result.Created {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resolveResponse{ //nolint:errcheck
			Ingredient: result.Ingredient,
			Confidence: result.Confidence,
			Created:    result.Created,
		})
	}
}

// --- merge ---

type mergeRequest struct {
	WinnerID string `json:"winner_id"`
	LoserID  string `json:"loser_id"`
}

func handleMerge(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mergeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		winnerID, err := uuid.Parse(req.WinnerID)
		if err != nil {
			jsonError(w, "invalid winner_id", http.StatusBadRequest)
			return
		}
		loserID, err := uuid.Parse(req.LoserID)
		if err != nil {
			jsonError(w, "invalid loser_id", http.StatusBadRequest)
			return
		}
		winner, err := svc.Merge(r.Context(), winnerID, loserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(w, "merge failed", http.StatusInternalServerError, err)
			return
		}
		jsonOK(w, winner)
	}
}

// --- helpers ---

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, msg string, status int, errs ...error) {
	if status >= 500 && len(errs) > 0 {
		slog.Error(msg, "status", status, "error", errs[0])
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
