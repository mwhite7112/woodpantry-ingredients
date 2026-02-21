//go:build integration

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationResolve_AutoCreate(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)

	result, err := svc.Resolve(context.Background(), "Flour")
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "flour", result.Ingredient.Name)
	assert.Equal(t, 1.0, result.Confidence)
}

func TestIntegrationResolve_ExactMatch(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)

	_, err := svc.Resolve(context.Background(), "flour")
	require.NoError(t, err)

	result, err := svc.Resolve(context.Background(), "flour")
	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, 1.0, result.Confidence)
	assert.Equal(t, "flour", result.Ingredient.Name)
}

func TestIntegrationResolve_FuzzyMatch(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.7)

	_, err := svc.Resolve(context.Background(), "chicken breast")
	require.NoError(t, err)

	result, err := svc.Resolve(context.Background(), "chicken breasts")
	require.NoError(t, err)
	assert.False(t, result.Created)
	assert.Equal(t, "chicken breast", result.Ingredient.Name)
	assert.Greater(t, result.Confidence, 0.7)
}

func TestIntegrationResolve_BelowThreshold(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)

	_, err := svc.Resolve(context.Background(), "flour")
	require.NoError(t, err)

	result, err := svc.Resolve(context.Background(), "garlic")
	require.NoError(t, err)
	assert.True(t, result.Created)
	assert.Equal(t, "garlic", result.Ingredient.Name)
}

func TestIntegrationResolve_ConcurrentRace(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)

	var wg sync.WaitGroup
	results := make([]ResolveResult, 10)
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.Resolve(context.Background(), "concurrent ingredient")
		}(i)
	}
	wg.Wait()

	var firstID string
	for i, err := range errs {
		require.NoError(t, err, "goroutine %d", i)
		if i == 0 {
			firstID = results[i].Ingredient.ID.String()
		}
		assert.Equal(t, firstID, results[i].Ingredient.ID.String(), "goroutine %d got different ID", i)
	}
}
