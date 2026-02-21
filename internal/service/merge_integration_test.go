//go:build integration

package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
	"github.com/mwhite7112/woodpantry-ingredients/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_Integration(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)
	ctx := context.Background()

	// Create two ingredients.
	winner, err := q.CreateIngredient(ctx, db.CreateIngredientParams{
		Name:    "garlic",
		Aliases: []string{"garlic clove"},
	})
	require.NoError(t, err)

	loser, err := q.CreateIngredient(ctx, db.CreateIngredientParams{
		Name:    "minced garlic",
		Aliases: []string{"garlic paste"},
	})
	require.NoError(t, err)

	// Merge loser into winner.
	merged, err := svc.Merge(ctx, winner.ID, loser.ID)
	require.NoError(t, err)
	assert.Equal(t, winner.ID, merged.ID)
	assert.Contains(t, merged.Aliases, "minced garlic")
	assert.Contains(t, merged.Aliases, "garlic paste")
	assert.Contains(t, merged.Aliases, "garlic clove")

	// Loser should be gone.
	_, err = q.GetIngredient(ctx, loser.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestMerge_MovesSubstitutes(t *testing.T) {
	sqlDB := testutil.SetupDB(t)
	q := db.New(sqlDB)
	svc := New(q, sqlDB, 0.8)
	ctx := context.Background()

	winner, err := q.CreateIngredient(ctx, db.CreateIngredientParams{
		Name:    "butter",
		Aliases: []string{},
	})
	require.NoError(t, err)

	loser, err := q.CreateIngredient(ctx, db.CreateIngredientParams{
		Name:    "unsalted butter",
		Aliases: []string{},
	})
	require.NoError(t, err)

	// Create a substitute referencing the loser.
	other, err := q.CreateIngredient(ctx, db.CreateIngredientParams{
		Name:    "margarine",
		Aliases: []string{},
	})
	require.NoError(t, err)

	_, err = q.CreateSubstitute(ctx, db.CreateSubstituteParams{
		IngredientID: loser.ID,
		SubstituteID: other.ID,
		Ratio:        1.0,
	})
	require.NoError(t, err)

	// Merge — substitute should now point to winner.
	_, err = svc.Merge(ctx, winner.ID, loser.ID)
	require.NoError(t, err)

	subs, err := q.ListSubstitutesByIngredient(ctx, winner.ID)
	require.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, other.ID, subs[0].SubstituteID)
}
