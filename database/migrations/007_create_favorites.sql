-- Migration pour créer la table favorites
CREATE TABLE IF NOT EXISTS favorites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, video_id)
);

-- Index pour améliorer les performances
CREATE INDEX IF NOT EXISTS idx_favorites_user_id ON favorites(user_id);
CREATE INDEX IF NOT EXISTS idx_favorites_video_id ON favorites(video_id);
CREATE INDEX IF NOT EXISTS idx_favorites_created_at ON favorites(created_at DESC);

-- Commentaires pour documentation
COMMENT ON TABLE favorites IS 'Table pour stocker les vidéos favorites des utilisateurs';
COMMENT ON COLUMN favorites.user_id IS 'ID de l''utilisateur qui a ajouté la vidéo en favori';
COMMENT ON COLUMN favorites.video_id IS 'ID de la vidéo ajoutée en favori';

