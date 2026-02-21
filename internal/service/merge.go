package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
)

// Merge combines loser into winner. The loser's name and aliases are appended
// to winner's aliases (deduplicated). All foreign key references in
// ingredient_substitutes and unit_conversions are re-pointed to winner, then
// the loser row is deleted (cascading any remaining FKs).
func (s *Service) Merge(ctx context.Context, winnerID, loserID uuid.UUID) (db.Ingredient, error) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return db.Ingredient{}, err
	}
	defer tx.Rollback() //nolint:errcheck

	qtx := db.New(tx)

	winner, err := qtx.GetIngredient(ctx, winnerID)
	if err != nil {
		return db.Ingredient{}, err
	}
	loser, err := qtx.GetIngredient(ctx, loserID)
	if err != nil {
		return db.Ingredient{}, err
	}

	// Merge loser name + aliases into winner aliases, deduplicated.
	merged := mergeAliases(winner.Aliases, loser.Name, loser.Aliases, winner.Name)

	winner, err = qtx.UpdateIngredient(ctx, db.UpdateIngredientParams{
		ID:          winnerID,
		Aliases:     merged,
		Category:    winner.Category,
		DefaultUnit: winner.DefaultUnit,
	})
	if err != nil {
		return db.Ingredient{}, err
	}

	// Re-point substitute references from loser to winner.
	if err := qtx.ReplaceSubstituteIngredient(ctx, db.ReplaceSubstituteIngredientParams{
		IngredientID:   winnerID,
		IngredientID_2: loserID,
	}); err != nil {
		return db.Ingredient{}, err
	}
	if err := qtx.ReplaceSubstituteSubId(ctx, db.ReplaceSubstituteSubIdParams{
		SubstituteID:   winnerID,
		SubstituteID_2: loserID,
	}); err != nil {
		return db.Ingredient{}, err
	}

	// Re-point unit conversion references.
	if err := qtx.ReplaceUnitConversionIngredient(ctx, db.ReplaceUnitConversionIngredientParams{
		IngredientID:   winnerID,
		IngredientID_2: loserID,
	}); err != nil {
		return db.Ingredient{}, err
	}

	// Delete loser — cascades any remaining substitutes/conversions.
	if err := qtx.DeleteIngredient(ctx, loserID); err != nil {
		return db.Ingredient{}, err
	}

	if err := tx.Commit(); err != nil {
		return db.Ingredient{}, err
	}

	return winner, nil
}

// mergeAliases combines existing winner aliases with the loser's name and
// aliases, excluding winnerName itself. Returns a deduplicated slice.
func mergeAliases(winnerAliases []string, loserName string, loserAliases []string, winnerName string) []string {
	seen := make(map[string]struct{}, len(winnerAliases)+1+len(loserAliases))
	result := make([]string, 0, len(winnerAliases)+1+len(loserAliases))

	add := func(s string) {
		if s == winnerName {
			return
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	for _, a := range winnerAliases {
		add(a)
	}
	add(loserName)
	for _, a := range loserAliases {
		add(a)
	}

	return result
}

