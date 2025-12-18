-- Migration pour ajouter les champs BaseURL et SearchURL à torrent_configs
ALTER TABLE torrent_configs 
ADD COLUMN IF NOT EXISTS base_url TEXT,
ADD COLUMN IF NOT EXISTS search_url TEXT;

-- Commentaires pour documentation
COMMENT ON COLUMN torrent_configs.base_url IS 'URL de base personnalisée du site torrent (ex: https://1337x.quest)';
COMMENT ON COLUMN torrent_configs.search_url IS 'URL de recherche/liste personnalisée (ex: /watch-movie/)';

