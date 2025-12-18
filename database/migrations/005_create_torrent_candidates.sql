-- Migration pour créer la table torrent_candidates
CREATE TABLE IF NOT EXISTS torrent_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES torrent_configs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    magnet_url TEXT,
    torrent_url TEXT,
    torrent_file_url TEXT,
    infohash VARCHAR(40),
    source VARCHAR(50) NOT NULL,
    size BIGINT,
    seeders INTEGER DEFAULT 0,
    leechers INTEGER DEFAULT 0,
    quality VARCHAR(20),
    language VARCHAR(10),
    category VARCHAR(100),
    upload_date VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending',
    approved_for_download BOOLEAN DEFAULT FALSE,
    rejected BOOLEAN DEFAULT FALSE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    sub_category_id UUID REFERENCES sub_categories(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index pour améliorer les performances
CREATE INDEX IF NOT EXISTS idx_torrent_candidates_config_id ON torrent_candidates(config_id);
CREATE INDEX IF NOT EXISTS idx_torrent_candidates_infohash ON torrent_candidates(infohash);
CREATE INDEX IF NOT EXISTS idx_torrent_candidates_status ON torrent_candidates(status);
CREATE INDEX IF NOT EXISTS idx_torrent_candidates_approved ON torrent_candidates(approved_for_download);

-- Commentaires pour documentation
COMMENT ON TABLE torrent_candidates IS 'Torrents trouvés par le système de recherche automatique, en attente de validation';
COMMENT ON COLUMN torrent_candidates.status IS 'pending, approved, rejected, downloaded';
COMMENT ON COLUMN torrent_candidates.approved_for_download IS 'Si true, ce candidat sera téléchargé automatiquement lors de la prochaine vérification';

