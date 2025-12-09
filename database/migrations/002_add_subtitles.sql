-- Migration pour ajouter le support des sous-titres
-- Date: 2025-12-08

-- Table des sous-titres
CREATE TABLE IF NOT EXISTS subtitles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL,
    language_name VARCHAR(50),
    file_path TEXT NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index pour améliorer les performances
CREATE INDEX IF NOT EXISTS idx_subtitles_video_id ON subtitles(video_id);
CREATE INDEX IF NOT EXISTS idx_subtitles_language ON subtitles(language);

-- Commentaires
COMMENT ON TABLE subtitles IS 'Sous-titres disponibles pour les vidéos';
COMMENT ON COLUMN subtitles.language IS 'Code langue ISO 639-1 (fr, en, es, etc.)';
COMMENT ON COLUMN subtitles.language_name IS 'Nom de la langue (Français, English, etc.)';
COMMENT ON COLUMN subtitles.file_path IS 'Chemin vers le fichier .vtt ou .srt';
COMMENT ON COLUMN subtitles.is_default IS 'Sous-titre par défaut pour cette vidéo';

