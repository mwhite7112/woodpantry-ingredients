CREATE TABLE IF NOT EXISTS ingredients (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  aliases TEXT[],
  category TEXT,
  default_unit TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
