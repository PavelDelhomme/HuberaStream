-- Migration: Ajout des métadonnées enrichies aux vidéos
-- Date: 2025-12-11

-- Ajouter les colonnes pour les métadonnées enrichies
ALTER TABLE videos 
ADD COLUMN IF NOT EXISTS director VARCHAR(255),
ADD COLUMN IF NOT EXISTS actors TEXT,
ADD COLUMN IF NOT EXISTS release_date VARCHAR(50),
ADD COLUMN IF NOT EXISTS production_year INTEGER,
ADD COLUMN IF NOT EXISTS synopsis TEXT,
ADD COLUMN IF NOT EXISTS genre VARCHAR(255),
ADD COLUMN IF NOT EXISTS rating DOUBLE PRECISION,
ADD COLUMN IF NOT EXISTS imdb_id VARCHAR(20),
ADD COLUMN IF NOT EXISTS tmdb_id INTEGER,
ADD COLUMN IF NOT EXISTS poster_url TEXT,
ADD COLUMN IF NOT EXISTS backdrop_url TEXT,
ADD COLUMN IF NOT EXISTS trailer_url TEXT,
ADD COLUMN IF NOT EXISTS runtime INTEGER,
ADD COLUMN IF NOT EXISTS writer VARCHAR(255),
ADD COLUMN IF NOT EXISTS platform VARCHAR(255);

-- Créer un index sur tmdb_id pour les recherches rapides
CREATE INDEX IF NOT EXISTS idx_videos_tmdb_id ON videos(tmdb_id);

-- Créer un index sur imdb_id pour les recherches rapides
CREATE INDEX IF NOT EXISTS idx_videos_imdb_id ON videos(imdb_id);

