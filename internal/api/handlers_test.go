package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mwhite7112/woodpantry-ingredients/internal/api"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/mocks"
	"github.com/mwhite7112/woodpantry-ingredients/internal/service"
)

// helpers

func newTestIngredient(name string) db.Ingredient {
	return db.Ingredient{
		ID:          uuid.New(),
		Name:        name,
		Aliases:     []string{},
		Category:    sql.NullString{},
		DefaultUnit: sql.NullString{},
		CreatedAt:   time.Now(),
	}
}

func setupRouter(t *testing.T) (*mocks.MockQuerier, http.Handler) {
	t.Helper()
	mockQ := mocks.NewMockQuerier(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(mockQ, nil, 0.8, log)
	router := api.NewRouter(svc, log)
	return mockQ, router
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, json.NewEncoder(buf).Encode(v))
	return buf
}

// ---------------------------------------------------------------------------
// GET /healthz
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	t.Parallel()
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

// ---------------------------------------------------------------------------
// GET /ingredients
// ---------------------------------------------------------------------------

func TestListIngredients_Empty(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	mockQ.EXPECT().ListIngredients(mock.Anything).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/ingredients", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var items []db.Ingredient
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&items)) //nolint:musttag // sqlc Ingredient
	assert.Empty(t, items)
}

func TestListIngredients_WithItems(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	garlic := newTestIngredient("garlic")
	salt := newTestIngredient("salt")
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic, salt}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ingredients", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var items []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&items))
	assert.Len(t, items, 2)
}

// ---------------------------------------------------------------------------
// POST /ingredients
// ---------------------------------------------------------------------------

func TestCreateIngredient_Success(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	created := newTestIngredient("garlic")
	mockQ.EXPECT().CreateIngredient(mock.Anything, mock.MatchedBy(func(p db.CreateIngredientParams) bool {
		return p.Name == "garlic"
	})).Return(created, nil)

	body := jsonBody(t, map[string]any{"name": "Garlic"})
	req := httptest.NewRequest(http.MethodPost, "/ingredients", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, created.ID.String(), got["ID"])
}

func TestCreateIngredient_MissingName(t *testing.T) {
	t.Parallel()
	_, router := setupRouter(t)

	body := jsonBody(t, map[string]any{"name": ""})
	req := httptest.NewRequest(http.MethodPost, "/ingredients", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var got map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Contains(t, got["error"], "name is required")
}

// ---------------------------------------------------------------------------
// GET /ingredients/:id
// ---------------------------------------------------------------------------

func TestGetIngredient_Success(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	garlic := newTestIngredient("garlic")
	mockQ.EXPECT().GetIngredient(mock.Anything, garlic.ID).Return(garlic, nil)

	req := httptest.NewRequest(http.MethodGet, "/ingredients/"+garlic.ID.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, garlic.ID.String(), got["ID"])
}

func TestGetIngredient_NotFound(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	mockQ.EXPECT().GetIngredient(mock.Anything, id).Return(db.Ingredient{}, sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/ingredients/"+id.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetIngredient_InvalidID(t *testing.T) {
	t.Parallel()
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ingredients/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// PUT /ingredients/:id
// ---------------------------------------------------------------------------

func TestUpdateIngredient_Success(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	updated := db.Ingredient{
		ID:          id,
		Name:        "garlic",
		Aliases:     []string{"garlic clove"},
		Category:    sql.NullString{String: "produce", Valid: true},
		DefaultUnit: sql.NullString{},
		CreatedAt:   time.Now(),
	}
	mockQ.EXPECT().UpdateIngredient(mock.Anything, mock.MatchedBy(func(p db.UpdateIngredientParams) bool {
		return p.ID == id
	})).Return(updated, nil)

	body := jsonBody(t, map[string]any{
		"aliases":  []string{"garlic clove"},
		"category": "produce",
	})
	req := httptest.NewRequest(http.MethodPut, "/ingredients/"+id.String(), body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, id.String(), got["ID"])
}

func TestUpdateIngredient_NotFound(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	mockQ.EXPECT().UpdateIngredient(mock.Anything, mock.Anything).Return(db.Ingredient{}, sql.ErrNoRows)

	body := jsonBody(t, map[string]any{"aliases": []string{}})
	req := httptest.NewRequest(http.MethodPut, "/ingredients/"+id.String(), body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// POST /ingredients/resolve
// ---------------------------------------------------------------------------

func TestResolve_ExistingIngredient(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	garlic := newTestIngredient("garlic")
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic}, nil)

	body := jsonBody(t, map[string]string{"name": "garlic"})
	req := httptest.NewRequest(http.MethodPost, "/ingredients/resolve", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	ing := resp["ingredient"].(map[string]any)
	assert.Equal(t, garlic.ID.String(), ing["ID"])
	assert.InDelta(t, 1.0, resp["confidence"], 0)
	assert.Equal(t, false, resp["created"])
}

func TestResolve_CreatesNew(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{}, nil)

	created := newTestIngredient("butter")
	mockQ.EXPECT().UpsertIngredient(mock.Anything, mock.Anything).Return(created, nil)

	body := jsonBody(t, map[string]string{"name": "Butter"})
	req := httptest.NewRequest(http.MethodPost, "/ingredients/resolve", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, true, resp["created"])
}

// ---------------------------------------------------------------------------
// POST /ingredients/merge
// ---------------------------------------------------------------------------

func TestMerge_InvalidIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body map[string]string
	}{
		{
			name: "invalid winner_id",
			body: map[string]string{"winner_id": "bad", "loser_id": uuid.New().String()},
		},
		{
			name: "invalid loser_id",
			body: map[string]string{"winner_id": uuid.New().String(), "loser_id": "bad"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, router := setupRouter(t)

			body := jsonBody(t, tc.body)
			req := httptest.NewRequest(http.MethodPost, "/ingredients/merge", body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

// ---------------------------------------------------------------------------
// Substitutes
// ---------------------------------------------------------------------------

func TestListSubstitutes(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	subID := uuid.New()
	sub := db.IngredientSubstitute{
		ID:           uuid.New(),
		IngredientID: id,
		SubstituteID: subID,
		Ratio:        1.5,
		Notes:        sql.NullString{String: "use more", Valid: true},
	}
	mockQ.EXPECT().ListSubstitutesByIngredient(mock.Anything, id).Return([]db.IngredientSubstitute{sub}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ingredients/"+id.String()+"/substitutes", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got, 1)
	assert.Equal(t, id.String(), got[0]["ingredient_id"])
	assert.Equal(t, subID.String(), got[0]["substitute_id"])
	ratio, ok := got[0]["ratio"].(float64)
	require.True(t, ok)
	assert.InEpsilon(t, 1.5, ratio, 1e-9)
	assert.Equal(t, "use more", got[0]["notes"])
}

func TestCreateSubstitute(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	subID := uuid.New()
	sub := db.IngredientSubstitute{
		ID:           uuid.New(),
		IngredientID: id,
		SubstituteID: subID,
		Ratio:        1.5,
		Notes:        sql.NullString{String: "test notes", Valid: true},
	}
	mockQ.EXPECT().CreateSubstitute(mock.Anything, mock.MatchedBy(func(p db.CreateSubstituteParams) bool {
		return p.IngredientID == id && p.SubstituteID == subID && p.Ratio == 1.5
	})).Return(sub, nil)

	body := jsonBody(t, map[string]any{
		"substitute_id": subID.String(),
		"ratio":         1.5,
		"notes":         "test notes",
	})
	req := httptest.NewRequest(http.MethodPost, "/ingredients/"+id.String()+"/substitutes", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, id.String(), got["ingredient_id"])
}

// ---------------------------------------------------------------------------
// Unit Conversions
// ---------------------------------------------------------------------------

func TestListConversions(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	conv := db.UnitConversion{
		ID:           uuid.New(),
		IngredientID: id,
		FromUnit:     "tbsp",
		ToUnit:       "tsp",
		Factor:       3.0,
	}
	mockQ.EXPECT().ListUnitConversionsByIngredient(mock.Anything, id).Return([]db.UnitConversion{conv}, nil)

	req := httptest.NewRequest(http.MethodGet, "/ingredients/"+id.String()+"/conversions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got, 1)
	assert.Equal(t, "tbsp", got[0]["from_unit"])
	factor, ok := got[0]["factor"].(float64)
	require.True(t, ok)
	assert.InEpsilon(t, 3.0, factor, 1e-9)
}

func TestCreateConversion(t *testing.T) {
	t.Parallel()
	mockQ, router := setupRouter(t)

	id := uuid.New()
	conv := db.UnitConversion{
		ID:           uuid.New(),
		IngredientID: id,
		FromUnit:     "kg",
		ToUnit:       "g",
		Factor:       1000.0,
	}
	mockQ.EXPECT().CreateUnitConversion(mock.Anything, mock.MatchedBy(func(p db.CreateUnitConversionParams) bool {
		return p.IngredientID == id && p.FromUnit == "kg" && p.ToUnit == "g" && p.Factor == 1000.0
	})).Return(conv, nil)

	body := jsonBody(t, map[string]any{
		"from_unit": "kg",
		"to_unit":   "g",
		"factor":    1000.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/ingredients/"+id.String()+"/conversions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "kg", got["from_unit"])
}
