CREATE TABLE IF NOT EXISTS unit_conversions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
  from_unit TEXT NOT NULL,
  to_unit TEXT NOT NULL,
  factor FLOAT8 NOT NULL
);

CREATE TABLE IF NOT EXISTS ingredient_substitutes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ingredient_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
  substitute_id UUID NOT NULL REFERENCES ingredients(id) ON DELETE CASCADE,
  ratio FLOAT8 NOT NULL DEFAULT 1.0,
  notes TEXT
);
