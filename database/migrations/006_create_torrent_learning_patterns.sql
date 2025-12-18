-- Migration pour créer la table torrent_learning_patterns
CREATE TABLE IF NOT EXISTS torrent_learning_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES torrent_configs(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    pattern_type VARCHAR(20) DEFAULT 'keyword',
    action VARCHAR(20) NOT NULL,
    score FLOAT DEFAULT 1.0,
    usage_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    last_used TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Index pour améliorer les performances
CREATE INDEX IF NOT EXISTS idx_torrent_learning_patterns_config_id ON torrent_learning_patterns(config_id);
CREATE INDEX IF NOT EXISTS idx_torrent_learning_patterns_pattern_type ON torrent_learning_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_torrent_learning_patterns_action ON torrent_learning_patterns(action);

-- Commentaires pour documentation
COMMENT ON TABLE torrent_learning_patterns IS 'Patterns d''apprentissage pour améliorer la sélection automatique de torrents';
COMMENT ON COLUMN torrent_learning_patterns.pattern IS 'Pattern à rechercher (mot-clé, regex, qualité, langue, etc.)';
COMMENT ON COLUMN torrent_learning_patterns.pattern_type IS 'Type de pattern: keyword, regex, quality, language, size, seeders';
COMMENT ON COLUMN torrent_learning_patterns.action IS 'Action à effectuer: approve, reject, prefer';
COMMENT ON COLUMN torrent_learning_patterns.score IS 'Score de confiance (0-1), augmente avec les succès';

