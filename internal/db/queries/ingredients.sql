-- name: ListIngredients :many
SELECT * FROM ingredients ORDER BY name;

-- name: GetIngredient :one
SELECT * FROM ingredients WHERE id = $1;

-- name: GetIngredientByName :one
SELECT * FROM ingredients WHERE name = $1;

-- name: CreateIngredient :one
INSERT INTO ingredients (name, aliases, category, default_unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpsertIngredient :one
INSERT INTO ingredients (name, aliases, category, default_unit)
VALUES ($1, $2, $3, $4)
ON CONFLICT (name) DO NOTHING
RETURNING *;

-- name: UpdateIngredient :one
UPDATE ingredients
SET aliases = $2, category = $3, default_unit = $4
WHERE id = $1
RETURNING *;

-- name: DeleteIngredient :exec
DELETE FROM ingredients WHERE id = $1;
