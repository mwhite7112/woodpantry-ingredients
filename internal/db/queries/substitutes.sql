-- name: ListSubstitutesByIngredient :many
SELECT * FROM ingredient_substitutes WHERE ingredient_id = $1;

-- name: CreateSubstitute :one
INSERT INTO ingredient_substitutes (ingredient_id, substitute_id, ratio, notes)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ReplaceSubstituteIngredient :exec
UPDATE ingredient_substitutes SET ingredient_id = $1 WHERE ingredient_id = $2;

-- name: ReplaceSubstituteSubId :exec
UPDATE ingredient_substitutes SET substitute_id = $1 WHERE substitute_id = $2;

-- name: DeleteSubstitutesByIngredient :exec
DELETE FROM ingredient_substitutes WHERE ingredient_id = $1 OR substitute_id = $1;
