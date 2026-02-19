-- name: ListUnitConversionsByIngredient :many
SELECT * FROM unit_conversions WHERE ingredient_id = $1;

-- name: CreateUnitConversion :one
INSERT INTO unit_conversions (ingredient_id, from_unit, to_unit, factor)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ReplaceUnitConversionIngredient :exec
UPDATE unit_conversions SET ingredient_id = $1 WHERE ingredient_id = $2;
