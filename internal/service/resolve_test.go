package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// similarity() unit tests
// ---------------------------------------------------------------------------

func TestSimilarity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		a, b    string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "exact match returns 1.0",
			a:       "garlic",
			b:       "garlic",
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "close match garlic/garlc",
			a:       "garlic",
			b:       "garlc",
			wantMin: 0.8,
			wantMax: 1.0,
		},
		{
			name:    "distant match garlic/butter",
			a:       "garlic",
			b:       "butter",
			wantMin: 0.0,
			wantMax: 0.4,
		},
		{
			name:    "both empty strings returns 1.0",
			a:       "",
			b:       "",
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "one empty string returns 0.0",
			a:       "garlic",
			b:       "",
			wantMin: 0.0,
			wantMax: 0.01,
		},
		{
			name:    "unicode strings",
			a:       "jalapeno",
			b:       "jalapeño",
			wantMin: 0.7,
			wantMax: 1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			score := similarity(tc.a, tc.b)
			assert.GreaterOrEqual(t, score, tc.wantMin, "score %f below expected min %f", score, tc.wantMin)
			assert.LessOrEqual(t, score, tc.wantMax, "score %f above expected max %f", score, tc.wantMax)
		})
	}
}

// ---------------------------------------------------------------------------
// Resolve() unit tests
// ---------------------------------------------------------------------------

func newIngredient(name string, aliases []string) db.Ingredient {
	return db.Ingredient{
		ID:          uuid.New(),
		Name:        name,
		Aliases:     aliases,
		Category:    sql.NullString{},
		DefaultUnit: sql.NullString{},
		CreatedAt:   time.Now(),
	}
}

func TestResolve_ExactNameMatch(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	garlic := newIngredient("garlic", []string{})
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic}, nil)

	result, err := svc.Resolve(context.Background(), "garlic")
	require.NoError(t, err)
	assert.Equal(t, garlic.ID, result.Ingredient.ID)
	assert.Equal(t, 1.0, result.Confidence)
	assert.False(t, result.Created)
}

func TestResolve_ExactAliasMatch(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	garlic := newIngredient("garlic", []string{"garlic clove"})
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic}, nil)

	result, err := svc.Resolve(context.Background(), "garlic clove")
	require.NoError(t, err)
	assert.Equal(t, garlic.ID, result.Ingredient.ID)
	assert.Equal(t, 1.0, result.Confidence)
	assert.False(t, result.Created)
}

func TestResolve_FuzzyAboveThreshold(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	garlic := newIngredient("garlic", []string{})
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic}, nil)

	// "garlc" is 1 edit away from "garlic" (6 chars) => similarity ~0.833
	result, err := svc.Resolve(context.Background(), "garlc")
	require.NoError(t, err)
	assert.Equal(t, garlic.ID, result.Ingredient.ID)
	assert.GreaterOrEqual(t, result.Confidence, 0.8)
	assert.False(t, result.Created)
}

func TestResolve_BelowThreshold_AutoCreate(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	garlic := newIngredient("garlic", []string{})
	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{garlic}, nil)

	created := newIngredient("butter", []string{})
	mockQ.EXPECT().UpsertIngredient(mock.Anything, mock.MatchedBy(func(p db.UpsertIngredientParams) bool {
		return p.Name == "butter"
	})).Return(created, nil)

	result, err := svc.Resolve(context.Background(), "Butter")
	require.NoError(t, err)
	assert.Equal(t, created.ID, result.Ingredient.ID)
	assert.Equal(t, 1.0, result.Confidence)
	assert.True(t, result.Created)
}

func TestResolve_ConcurrentConflictFallback(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{}, nil)

	// Upsert returns sql.ErrNoRows (ON CONFLICT DO NOTHING returned no row).
	mockQ.EXPECT().UpsertIngredient(mock.Anything, mock.Anything).
		Return(db.Ingredient{}, sql.ErrNoRows)

	// Fallback: GetIngredientByName succeeds.
	existing := newIngredient("butter", []string{})
	mockQ.EXPECT().GetIngredientByName(mock.Anything, "butter").
		Return(existing, nil)

	result, err := svc.Resolve(context.Background(), "Butter")
	require.NoError(t, err)
	assert.Equal(t, existing.ID, result.Ingredient.ID)
	assert.Equal(t, 1.0, result.Confidence)
	assert.False(t, result.Created)
}

func TestResolve_EmptyDB_AutoCreate(t *testing.T) {
	t.Parallel()

	mockQ := mocks.NewMockQuerier(t)
	svc := New(mockQ, nil, 0.8)

	mockQ.EXPECT().ListIngredients(mock.Anything).Return([]db.Ingredient{}, nil)

	created := newIngredient("salt", []string{})
	mockQ.EXPECT().UpsertIngredient(mock.Anything, mock.MatchedBy(func(p db.UpsertIngredientParams) bool {
		return p.Name == "salt"
	})).Return(created, nil)

	result, err := svc.Resolve(context.Background(), "Salt")
	require.NoError(t, err)
	assert.Equal(t, created.ID, result.Ingredient.ID)
	assert.True(t, result.Created)
}
