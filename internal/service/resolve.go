package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/agnivade/levenshtein"
	"github.com/mwhite7112/woodpantry-ingredients/internal/db"
)

// ResolveResult is returned by Resolve.
type ResolveResult struct {
	Ingredient db.Ingredient
	Confidence float64
	Created    bool
}

// similarity returns a 0.0–1.0 confidence score between two strings using
// Levenshtein distance: 1.0 - distance/max(len(a), len(b)).
func similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	maxLen := len([]rune(a))
	if lb := len([]rune(b)); lb > maxLen {
		maxLen = lb
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshtein.ComputeDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// Resolve finds the best-matching ingredient for a raw name string. If the best
// match is above the configured threshold, it is returned directly. Otherwise a
// new ingredient is auto-created (write-through). Concurrent callers are safe:
// the upsert uses ON CONFLICT DO NOTHING and falls back to a SELECT on conflict.
func (s *Service) Resolve(ctx context.Context, rawName string) (ResolveResult, error) {
	normalized := Normalize(rawName)

	all, err := s.q.ListIngredients(ctx)
	if err != nil {
		return ResolveResult{}, err
	}

	var bestIngredient db.Ingredient
	bestScore := -1.0

	for _, ing := range all {
		// Check exact name match first.
		if ing.Name == normalized {
			return ResolveResult{Ingredient: ing, Confidence: 1.0, Created: false}, nil
		}
		score := similarity(normalized, ing.Name)

		// Check aliases — exact alias match is an immediate hit.
		for _, alias := range ing.Aliases {
			if alias == normalized {
				return ResolveResult{Ingredient: ing, Confidence: 1.0, Created: false}, nil
			}
			if s := similarity(normalized, alias); s > score {
				score = s
			}
		}

		if score > bestScore {
			bestScore = score
			bestIngredient = ing
		}
	}

	if bestScore >= s.threshold {
		return ResolveResult{Ingredient: bestIngredient, Confidence: bestScore, Created: false}, nil
	}

	// No match above threshold — auto-create.
	ing, err := s.q.UpsertIngredient(ctx, db.UpsertIngredientParams{
		Name:        normalized,
		Aliases:     []string{},
		Category:    sql.NullString{},
		DefaultUnit: sql.NullString{},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Concurrent insert won the race; fetch the existing row.
			ing, err = s.q.GetIngredientByName(ctx, normalized)
			if err != nil {
				return ResolveResult{}, err
			}
			return ResolveResult{Ingredient: ing, Confidence: 1.0, Created: false}, nil
		}
		return ResolveResult{}, err
	}

	return ResolveResult{Ingredient: ing, Confidence: 1.0, Created: true}, nil
}
