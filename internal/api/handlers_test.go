package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mwhite7112/woodpantry-ingredients/internal/api"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/mocks"
	"github.com/mwhite7112/woodpantry-ingredients/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	svc := service.New(mockQ, nil, 0.8)
	router := api.NewRouter(svc)
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
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&items))
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
	assert.Equal(t, 1.0, resp["confidence"])
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
