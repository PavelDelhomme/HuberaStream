-- Migration pour créer la table torrent_indexes
CREATE TABLE IF NOT EXISTS torrent_indexes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    source VARCHAR(50) NOT NULL,
    base_url TEXT NOT NULL,
    search_url TEXT,
    enabled BOOLEAN DEFAULT TRUE NOT NULL,
    priority INTEGER DEFAULT 0 NOT NULL,
    is_default BOOLEAN DEFAULT FALSE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Index pour améliorer les performances
CREATE INDEX IF NOT EXISTS idx_torrent_indexes_source ON torrent_indexes(source);
CREATE INDEX IF NOT EXISTS idx_torrent_indexes_enabled ON torrent_indexes(enabled);
CREATE INDEX IF NOT EXISTS idx_torrent_indexes_priority ON torrent_indexes(priority DESC);

-- Contrainte unique : un seul index par défaut par source
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_default_index_per_source 
ON torrent_indexes(source) 
WHERE is_default = TRUE AND deleted_at IS NULL;

-- Commentaires pour documentation
COMMENT ON TABLE torrent_indexes IS 'Table pour stocker les index de torrents configurables (1337x, etc.)';
COMMENT ON COLUMN torrent_indexes.base_url IS 'URL de base de l''index (ex: https://1337x.quest)';
COMMENT ON COLUMN torrent_indexes.search_url IS 'URL de recherche avec placeholder {query} (ex: /lmsearch?q={query}&category=lmsearch)';
COMMENT ON COLUMN torrent_indexes.priority IS 'Priorité de l''index (plus élevé = utilisé en premier)';
COMMENT ON COLUMN torrent_indexes.is_default IS 'Si true, cet index est utilisé par défaut pour cette source';

-- Insérer l'index par défaut pour 1337x
INSERT INTO torrent_indexes (name, source, base_url, search_url, enabled, priority, is_default, description)
VALUES (
    '1337x.quest (Par défaut)',
    '1337x',
    'https://1337x.quest',
    '/lmsearch?q={query}&category=lmsearch',
    TRUE,
    100,
    TRUE,
    'Index 1337x.quest par défaut pour la recherche de torrents'
) ON CONFLICT DO NOTHING;

