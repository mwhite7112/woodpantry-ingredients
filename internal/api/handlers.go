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
func NewRouter(svc *service.Service, log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(logging.Middleware)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", handleHealth)

	r.Get("/ingredients", handleListIngredients(svc, log))
	r.Post("/ingredients", handleCreateIngredient(svc, log))
	r.Post("/ingredients/resolve", handleResolve(svc, log))
	r.Post("/ingredients/merge", handleMerge(svc, log))
	r.Get("/ingredients/{id}", handleGetIngredient(svc, log))
	r.Put("/ingredients/{id}", handleUpdateIngredient(svc, log))

	r.Get("/ingredients/{id}/substitutes", handleListSubstitutes(svc, log))
	r.Post("/ingredients/{id}/substitutes", handleCreateSubstitute(svc, log))
	r.Get("/ingredients/{id}/conversions", handleListConversions(svc, log))
	r.Post("/ingredients/{id}/conversions", handleCreateConversion(svc, log))

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok")) //nolint:errcheck
}

// --- list ---

func handleListIngredients(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.Queries().ListIngredients(r.Context())
		if err != nil {
			jsonError(log, w, "failed to list ingredients", http.StatusInternalServerError, err)
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

func handleCreateIngredient(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createIngredientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			jsonError(log, w, "name is required", http.StatusBadRequest)
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
			jsonError(log, w, "failed to create ingredient", http.StatusInternalServerError, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ing) //nolint:errcheck,musttag // sqlc-generated Ingredient lacks json tags
	}
}

// --- get ---

func handleGetIngredient(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		ing, err := svc.Queries().GetIngredient(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(log, w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(log, w, "failed to get ingredient", http.StatusInternalServerError, err)
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

func handleUpdateIngredient(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		var req updateIngredientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
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
				jsonError(log, w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(log, w, "failed to update ingredient", http.StatusInternalServerError, err)
			return
		}
		jsonOK(w, ing)
	}
}

// --- substitutes ---

type substituteResponse struct {
	ID           uuid.UUID `json:"id"`
	IngredientID uuid.UUID `json:"ingredient_id"`
	SubstituteID uuid.UUID `json:"substitute_id"`
	Ratio        float64   `json:"ratio"`
	Notes        string    `json:"notes,omitempty"`
}

func handleListSubstitutes(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		items, err := svc.Queries().ListSubstitutesByIngredient(r.Context(), id)
		if err != nil {
			jsonError(log, w, "failed to list substitutes", http.StatusInternalServerError, err)
			return
		}
		if items == nil {
			items = []db.IngredientSubstitute{}
		}

		resp := make([]substituteResponse, len(items))
		for i, item := range items {
			resp[i] = substituteResponse{
				ID:           item.ID,
				IngredientID: item.IngredientID,
				SubstituteID: item.SubstituteID,
				Ratio:        item.Ratio,
				Notes:        item.Notes.String,
			}
		}
		jsonOK(w, resp)
	}
}

type createSubstituteRequest struct {
	SubstituteID string  `json:"substitute_id"`
	Ratio        float64 `json:"ratio"`
	Notes        string  `json:"notes"`
}

func handleCreateSubstitute(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		var req createSubstituteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
			return
		}
		subID, err := uuid.Parse(req.SubstituteID)
		if err != nil {
			jsonError(log, w, "invalid substitute_id", http.StatusBadRequest)
			return
		}

		ratio := req.Ratio
		if ratio == 0 {
			ratio = 1.0
		}

		item, err := svc.Queries().CreateSubstitute(r.Context(), db.CreateSubstituteParams{
			IngredientID: id,
			SubstituteID: subID,
			Ratio:        ratio,
			Notes:        nullString(req.Notes),
		})
		if err != nil {
			jsonError(log, w, "failed to create substitute", http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(substituteResponse{ //nolint:errcheck
			ID:           item.ID,
			IngredientID: item.IngredientID,
			SubstituteID: item.SubstituteID,
			Ratio:        item.Ratio,
			Notes:        item.Notes.String,
		})
	}
}

// --- conversions ---

type conversionResponse struct {
	ID           uuid.UUID `json:"id"`
	IngredientID uuid.UUID `json:"ingredient_id"`
	FromUnit     string    `json:"from_unit"`
	ToUnit       string    `json:"to_unit"`
	Factor       float64   `json:"factor"`
}

func handleListConversions(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		items, err := svc.Queries().ListUnitConversionsByIngredient(r.Context(), id)
		if err != nil {
			jsonError(log, w, "failed to list conversions", http.StatusInternalServerError, err)
			return
		}
		if items == nil {
			items = []db.UnitConversion{}
		}

		resp := make([]conversionResponse, len(items))
		for i, item := range items {
			resp[i] = conversionResponse{
				ID:           item.ID,
				IngredientID: item.IngredientID,
				FromUnit:     item.FromUnit,
				ToUnit:       item.ToUnit,
				Factor:       item.Factor,
			}
		}
		jsonOK(w, resp)
	}
}

type createConversionRequest struct {
	FromUnit string  `json:"from_unit"`
	ToUnit   string  `json:"to_unit"`
	Factor   float64 `json:"factor"`
}

func handleCreateConversion(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			jsonError(log, w, "invalid id", http.StatusBadRequest)
			return
		}
		var req createConversionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.FromUnit == "" || req.ToUnit == "" || req.Factor == 0 {
			jsonError(log, w, "from_unit, to_unit and factor are required", http.StatusBadRequest)
			return
		}

		item, err := svc.Queries().CreateUnitConversion(r.Context(), db.CreateUnitConversionParams{
			IngredientID: id,
			FromUnit:     req.FromUnit,
			ToUnit:       req.ToUnit,
			Factor:       req.Factor,
		})
		if err != nil {
			jsonError(log, w, "failed to create conversion", http.StatusInternalServerError, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(conversionResponse{ //nolint:errcheck
			ID:           item.ID,
			IngredientID: item.IngredientID,
			FromUnit:     item.FromUnit,
			ToUnit:       item.ToUnit,
			Factor:       item.Factor,
		})
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

func handleResolve(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req resolveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			jsonError(log, w, "name is required", http.StatusBadRequest)
			return
		}
		result, err := svc.Resolve(r.Context(), req.Name)
		if err != nil {
			jsonError(log, w, "resolve failed", http.StatusInternalServerError, err)
			return
		}
		status := http.StatusOK
		if result.Created {
			status = http.StatusCreated
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resolveResponse{ //nolint:errcheck,musttag // sqlc Ingredient
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

func handleMerge(svc *service.Service, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req mergeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(log, w, "invalid request body", http.StatusBadRequest)
			return
		}
		winnerID, err := uuid.Parse(req.WinnerID)
		if err != nil {
			jsonError(log, w, "invalid winner_id", http.StatusBadRequest)
			return
		}
		loserID, err := uuid.Parse(req.LoserID)
		if err != nil {
			jsonError(log, w, "invalid loser_id", http.StatusBadRequest)
			return
		}
		winner, err := svc.Merge(r.Context(), winnerID, loserID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(log, w, "ingredient not found", http.StatusNotFound)
				return
			}
			jsonError(log, w, "merge failed", http.StatusInternalServerError, err)
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

func jsonError(log *slog.Logger, w http.ResponseWriter, msg string, status int, errs ...error) {
	if status >= http.StatusInternalServerError && len(errs) > 0 {
		log.Error(msg, "status", status, "error", errs[0])
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
