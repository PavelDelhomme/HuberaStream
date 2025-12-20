# Status du Projet StreamMake

**Dernière mise à jour**: 20 décembre 2025

## 🔴 PRIORITÉS À VALIDER / À FAIRE (En haut de la liste)

### ⚠️ À VALIDER EN PRIORITÉ

1. **Test des nouvelles sources de torrents** 🔴
   - [ ] Tester The Pirate Bay avec recherche "alice" ou "inception"
   - [ ] Tester TorrentDownload avec recherche "alice" ou "inception"
   - [ ] Tester 1337x.proxyninja.net (via base_url personnalisée)
   - [ ] Vérifier que les résultats sont correctement parsés
   - [ ] Vérifier que les candidats sont créés correctement

2. **Amélioration du filtrage audio** 🔴
   - [x] Filtrage des fichiers audio (MP3, FLAC, etc.)
   - [x] Filtrage des versions instrumentales
   - [ ] Tester avec plusieurs recherches pour valider le filtrage
   - [ ] Vérifier qu'aucun fichier audio n'apparaît dans les candidats

3. **Gestion des torrents en téléchargement** 🟡
   - [x] Affichage de la progression avec barre de chargement
   - [x] Indicateur "Récupération des métadonnées..."
   - [x] Affichage de la vitesse de téléchargement
   - [ ] Tester le rafraîchissement automatique toutes les 5 secondes
   - [ ] Vérifier que la progression se met à jour correctement
   - [ ] Tester pause/resume des torrents

4. **Système de candidats torrent** 🟡
   - [x] Création automatique de candidats lors de la recherche
   - [x] Rafraîchissement automatique du modal après recherche
   - [x] Affichage des candidats même s'ils ne respectent pas les filtres (marqués comme rejetés)
   - [ ] Tester l'approbation/désapprobation de candidats
   - [ ] Tester le téléchargement depuis les candidats
   - [ ] Vérifier que les candidats rejetés ne sont pas téléchargés automatiquement

5. **Post-traitement automatique** 🔴
   - [ ] Détection automatique de fin de téléchargement
   - [ ] Extraction des fichiers vidéo depuis les torrents terminés
   - [ ] Création automatique d'entrées vidéo dans la base de données
   - [ ] Association avec catégories et métadonnées
   - [ ] Transcodage automatique après téléchargement

## 🎯 État Actuel du Projet

### ✅ Problèmes Résolus Récemment

#### Problème 403 Forbidden - **RÉSOLU** ✅
- **Date de résolution**: 8 décembre 2025
- **Problème**: Les requêtes POST avec header `Origin` via Nginx retournaient 403 Forbidden
- **Solution**: Configuration `SetTrustedProxies` dans Gin avec les IPs correctes des réseaux Docker
- **Fichiers modifiés**:
  - `backend/cmd/server/main.go` - Configuration SetTrustedProxies
  - `backend/Dockerfile` - Mise à jour Go 1.24
  - `backend/go.mod` - Version Go 1.24
  - `frontend/src/services/api.js` - Réactivation Nginx
  - `docker-compose.yml` - Variables d'environnement
- **Documentation**: Voir `docs/SOLUTION-403-RESOLU.md`
- **Tests**: ✅ Tous les tests passent (200 OK)

#### Problème Upload Vidéo et Accès Vidéos en Traitement - **RÉSOLU** ✅
- **Date de résolution**: 10 décembre 2025
- **Problème**: 
  - Vidéos en traitement retournaient 404 même si elles existaient
  - Vidéos soft-deleted n'étaient pas accessibles même en traitement
  - Miniatures non servies correctement
  - Polling sans retry pour rechargement vidéo complète
- **Solution**: 
  - Gestion des vidéos soft-deleted avec accès autorisé si en traitement
  - `OptionalAuthMiddleware` pour routes publiques avec authentification optionnelle
  - Handler dédié pour servir les thumbnails (`/api/thumbnails/:filename`)
  - Polling avec retry (3 tentatives) pour rechargement vidéo complète
  - Exclusion vidéo actuelle des vidéos similaires
- **Fichiers modifiés**:
  - `backend/internal/api/video_handler.go` - Gestion soft-deleted et permissions
  - `backend/internal/api/stream_handler.go` - Handler `ServeThumbnail`
  - `backend/internal/api/routes.go` - Route thumbnails et `OptionalAuthMiddleware`
  - `backend/internal/middleware/auth.go` - Nouveau `OptionalAuthMiddleware`
  - `frontend/src/pages/video/VideoPlayer.jsx` - Polling avec retry
- **Tests**: ✅ Vidéos en traitement accessibles, thumbnails servis, polling fonctionnel

## ✅ Fonctionnalités Complétées

### Infrastructure
- [x] Architecture Docker complète avec docker-compose
- [x] Backend Go avec Gin framework (v1.11.0)
- [x] Frontend React avec Vite
- [x] Base de données PostgreSQL avec migrations
- [x] Redis pour le cache
- [x] Nginx comme reverse proxy (configuré et fonctionnel)
- [x] Makefile pour la gestion du projet
- [x] Configuration .env
- [x] Configuration SetTrustedProxies pour les proxies Docker

### Authentification & Utilisateurs
- [x] Système d'authentification JWT
- [x] Système de rôles (admin/user)
- [x] Utilisateur admin par défaut (dev@delhomme.ovh)
- [x] Inscription et connexion
- [x] Middleware d'authentification et d'administration

### Gestion des Vidéos
- [x] CRUD complet pour les vidéos
- [x] Upload de vidéos personnelles (admin seulement)
- [x] Upload de thumbnails
- [x] Système de catégories et sous-catégories
- [x] Association vidéos/catégories
- [x] Recherche de vidéos
- [x] Historique de visionnage
- [x] Listes personnelles (anciennement Playlists)
- [x] Section "Continuer à regarder" sur la page d'accueil
- [x] Extraction DVD (admin seulement)
- [x] Système de statut de traitement (pending, processing, ready, error, cancelled)
- [x] Polling en temps réel pour statut de traitement avec progression
- [x] Génération de thumbnails automatique (multiples positions, fallback textuel)
- [x] Vérification d'intégrité avec checksums SHA256
- [x] Gestion des vidéos invalides (détection et nettoyage)
- [x] Annulation de traitement vidéo
- [x] Accès aux vidéos en traitement même si soft-deleted
- [x] Exclusion vidéo actuelle des vidéos similaires

### Streaming
- [x] Support multi-formats (HLS, DASH, WebRTC, Progressive)
- [x] Lecteur vidéo amélioré
- [x] Sélection de qualité
- [x] Transcodage vidéo avec FFmpeg
- [x] Génération de thumbnails
- [x] Transcodage multi-qualités (360p, 480p, 720p, 1080p)
- [x] Génération HLS avec plusieurs qualités
- [x] Génération DASH avec plusieurs qualités
- [x] Affichage progression de traitement en temps réel
- [x] Écran de traitement avec barre de progression
- [x] Gestion des erreurs de transcodage

### Interface d'Administration
- [x] Tableau de bord admin avec statistiques
- [x] Gestion des utilisateurs
- [x] Gestion des catégories et sous-catégories
- [x] Interface de gestion des torrents
- [x] Configuration des téléchargements automatiques
- [x] Gestion des clients torrent (Transmission, qBittorrent, Deluge, rTorrent)
- [x] Paramètres multimédias (qualités, formats, thumbnails)
- [x] Menu admin amélioré avec accès rapide
- [x] Menu profil admin avec historique et configuration

### Système de Torrents (Structure)
- [x] Modèles de données pour torrents
- [x] Modèles pour configurations de torrents
- [x] API REST pour gestion des torrents
- [x] Support de multiples sources (1337x, RARBG, YTS, Nyaa, Custom)
- [x] **NOUVEAU**: Support The Pirate Bay (piratebay)
- [x] **NOUVEAU**: Support TorrentDownload (torrentdownload)
- [x] **NOUVEAU**: Support 1337x.proxyninja.net (via base_url)
- [x] **NOUVEAU**: Système de candidats torrent (pending, approved, rejected, downloaded)
- [x] **NOUVEAU**: Filtrage automatique des fichiers audio et versions instrumentales
- [x] **NOUVEAU**: Rafraîchissement automatique des candidats après recherche
- [x] **NOUVEAU**: Affichage de la progression de téléchargement avec barre
- [x] **NOUVEAU**: Indicateur de récupération des métadonnées
- [x] **NOUVEAU**: Affichage de la vitesse de téléchargement
- [x] **NOUVEAU**: Système d'index de torrents configurable

## 🚧 En Cours / À Implémenter

### Clients Torrent
- [x] Intégration Transmission (API RPC) - Port 3305
- [x] Intégration qBittorrent (API v2) - Port 3306
- [x] Intégration Deluge (JSON-RPC) - Port 3307
- [x] Intégration rTorrent (ruTorrent Web UI) - Port 3308
- [x] Service unifié de gestion torrent
- [x] Interface admin pour gérer les clients torrent
- [x] Configuration des clients via interface web
- [x] Test de connexion aux clients
- [x] Tous les ports exposés dans la plage 33XX
- [ ] **À FAIRE**: Tests d'intégration avec clients réels
- [ ] **À FAIRE**: Gestion des erreurs de connexion aux clients

### Scrapers de Sources Torrent
- [x] Scraper 1337x (parser HTML amélioré avec goquery)
- [x] Scraper RARBG (API officielle)
- [x] Scraper YTS (API officielle)
- [x] Scraper Nyaa (parser HTML)
- [x] **NOUVEAU**: Scraper The Pirate Bay (parser HTML avec goquery)
- [x] **NOUVEAU**: Scraper TorrentDownload (parser HTML avec goquery)
- [x] Service de recherche unifié
- [x] Support des URLs personnalisées (base_url, search_url)
- [x] Gestion de la compression Brotli/Gzip
- [x] Filtrage automatique des fichiers audio et versions instrumentales
- [ ] **À FAIRE**: Gestion des erreurs et retry améliorée
- [ ] **À FAIRE**: Cache des résultats de recherche

### Téléchargement Automatique
- [x] Service de monitoring des configurations
- [x] Recherche automatique selon mots-clés
- [x] Filtrage par seeders, taille, qualité
- [x] **NOUVEAU**: Système de candidats (création, approbation, rejet)
- [x] **NOUVEAU**: Filtrage automatique des fichiers audio/instrumentaux
- [x] **NOUVEAU**: Création de candidats même s'ils ne respectent pas les filtres (marqués comme rejetés)
- [x] **NOUVEAU**: Rafraîchissement automatique du modal après recherche
- [x] Ajout automatique au client torrent (pour candidats approuvés)
- [x] Suivi de progression avec mise à jour automatique
- [x] Affichage de la vitesse de téléchargement
- [x] Gestion des statuts (pending, downloading, seeding, completed, error, paused)
- [ ] **À FAIRE**: Post-traitement (transcodage, organisation)
- [ ] **À FAIRE**: Détection automatique de fin de téléchargement
- [ ] **À FAIRE**: Création automatique de vidéo depuis torrent
- [ ] **À FAIRE**: Notification de fin de téléchargement

### Amélioration du Transcodage
- [x] Transcodage multi-qualités automatique (1080p, 720p, 480p, 360p)
- [x] Génération HLS avec plusieurs qualités
- [x] Génération DASH avec plusieurs qualités
- [ ] **À FAIRE**: Queue de transcodage avec priorité
- [ ] **À FAIRE**: Notification de fin de transcodage
- [ ] **À FAIRE**: Transcodage automatique après upload/téléchargement
- [ ] **À FAIRE**: Gestion des erreurs de transcodage

### Interface Utilisateur
- [ ] **À FAIRE**: Amélioration du lecteur média
- [ ] **À FAIRE**: Filtrage par catégories sur la page d'accueil
- [ ] **À FAIRE**: Page de détails vidéo améliorée
- [ ] **À FAIRE**: Gestion des favoris
- [ ] **À FAIRE**: Notifications en temps réel
- [ ] **À FAIRE**: Mode sombre/clair
- [ ] **À FAIRE**: Responsive design amélioré

### Tests & Qualité
- [ ] **À FAIRE**: Tests unitaires backend
- [ ] **À FAIRE**: Tests d'intégration API
- [ ] **À FAIRE**: Tests end-to-end frontend
- [ ] **À FAIRE**: Tests de charge
- [ ] **À FAIRE**: Couverture de code

## 🐛 Problèmes Connus

### Extraction DVD - **EN COURS** 🟡
- **Date de détection**: 20 décembre 2025
- **Problème 1**: L'extraction DVD échoue avec l'erreur `Unrecognized option 'dvd_device'`
  - **Cause**: Le backend semble utiliser une version obsolète du code compilé qui ajoute encore `-dvd_device` aux arguments `ffmpeg`, alors que cette option n'est plus supportée
  - **Solution en cours**: 
    - Ajout de logs de debug détaillés pour tracer la construction des arguments `ffmpeg`
    - Reconstruction du backend avec `--no-cache` pour forcer une recompilation complète
    - Vérification que `-dvd_device` n'est pas ajouté aux args et utilisation de la variable d'environnement `DVD_DEVICE` à la place
- **Problème 2**: Erreur de parsing JSON lors de la lecture des informations DVD : `invalid character '[' looking for beginning of object key string`
  - **Cause**: Les messages d'erreur AC3 (`[ac3 @ ...] error decoding the audio block "programs"`) polluent la sortie JSON de `ffprobe`, et la regex trouve `"programs"` dans le message d'erreur au lieu de la vraie clé JSON
  - **Solution appliquée**: 
    - Amélioration de la logique de nettoyage JSON pour trouver toutes les occurrences de clés JSON
    - Vérification que chaque occurrence n'est pas dans un message d'erreur (détection de patterns comme `[ac3 @`, `error decoding`, `expacc`, `out-of-range`)
    - Utilisation de la première occurrence valide qui n'est pas dans un message d'erreur
- **Fichiers modifiés**:
  - `backend/internal/video/dvd_extractor.go` - Logs de debug, vérification `-dvd_device`, amélioration nettoyage JSON
- **Prochaines étapes**: 
  - Tester la lecture des informations DVD avec la nouvelle logique de nettoyage JSON
  - Tester l'extraction DVD après reconstruction du backend
  - Vérifier les logs de debug pour identifier où `-dvd_device` est ajouté si le problème persiste

### Clients Torrent
- [x] **CONFIGURÉ** - Tous les clients torrent sont configurés dans docker-compose.yml
  - Transmission: http://localhost:3305 (port 3305)
  - qBittorrent: http://localhost:3306 (port 3306)
  - Deluge: http://localhost:3307 (port 3307)
  - ruTorrent: http://localhost:3308 (port 3308)
- [x] **FAIT**: Interface admin pour gérer les clients torrent (`/admin/torrent-clients`)
- [x] **FAIT**: Test de connexion pour chaque client
- [x] **FAIT**: Configuration via interface web
- [x] **FAIT**: Commande `make install-torrents` pour installer/démarrer

### Base de Données
- [x] **RÉSOLU** - Migrations automatiques avec colonnes role et is_active

### CORS
- [x] **RÉSOLU** - Configuration CORS mise à jour pour inclure tous les ports
- [x] **RÉSOLU** - Configuration SetTrustedProxies pour les proxies Docker

### Upload et Traitement Vidéo
- [x] **RÉSOLU** - Vidéos en traitement accessibles même si soft-deleted
- [x] **RÉSOLU** - Miniatures servies correctement via handler dédié
- [x] **RÉSOLU** - Polling avec retry pour rechargement vidéo complète
- [x] **RÉSOLU** - Exclusion vidéo actuelle des vidéos similaires
- [x] **RÉSOLU** - Gestion vidéos invalides avec détection et nettoyage

## 📝 Notes Techniques

### Ports Utilisés
- 3300 : Nginx (point d'entrée principal) ✅
- 3301 : Backend API
- 3302 : PostgreSQL
- 3303 : Redis
- 3304 : Frontend direct (dev)
- 3305 : Transmission Web UI ✅
- 3306 : qBittorrent Web UI ✅
- 3307 : Deluge Web UI ✅
- 3308 : ruTorrent Web UI ✅
- 3309 : Deluge Daemon
- 3310 : rTorrent XMLRPC
- 3311 : rTorrent DHT
- 2345 : Delve debugger

### Identifiants Admin
- Email: dev@delhomme.ovh
- Password: AdminStreamMake2024!

### Technologies
- Backend: Go 1.24, Gin v1.11.0, GORM, PostgreSQL
- Frontend: React 18, Vite, React Router
- Streaming: FFmpeg, HLS, DASH
- Containerisation: Docker, Docker Compose

### Structure du Projet
```
StreamMake/
├── backend/          # Code backend Go
├── frontend/         # Code frontend React
├── nginx/            # Configuration Nginx
├── database/         # Migrations SQL
├── docs/             # Documentation (investigations, solutions)
├── scripts/          # Scripts utilitaires (diagnostic, tests)
├── README.md         # Documentation principale
└── STATUS.md         # Ce fichier
```

## 🔄 Prochaines Étapes Prioritaires

> **Note**: Une fois les problèmes d'upload de vidéos complètement résolus, nous nous concentrerons sur les fonctionnalités torrent et les améliorations suivantes.

### Priorité Haute 🔴 (Après résolution upload)

#### Intégration Torrent Complète
1. **Post-traitement automatique** - Créer des vidéos depuis les torrents téléchargés
   - Détection automatique de fichiers vidéo dans les torrents terminés
   - Extraction et organisation des fichiers
   - Création automatique d'entrées vidéo dans la base de données
   - Association avec catégories et métadonnées

2. **Queue de transcodage** - Gérer les transcodages en arrière-plan avec priorité
   - Système de queue avec priorités (haute, normale, basse)
   - Gestion de la charge CPU/mémoire
   - Limite de transcodages simultanés
   - Reprise après erreur

3. **Détection automatique de fin de téléchargement**
   - Monitoring des torrents actifs
   - Détection changement de statut (downloading → seeding/completed)
   - Déclenchement automatique du post-traitement
   - Notification utilisateur/admin

4. **Amélioration des parsers HTML**
   - Parser 1337x complet et robuste
   - Parser Nyaa amélioré avec gestion erreurs
   - Gestion des changements de structure HTML
   - Retry automatique en cas d'échec

### Priorité Moyenne 🟡

5. **Gestion des erreurs améliorée**
   - Logging structuré et centralisé
   - Retry automatique avec backoff exponentiel
   - Notifications d'erreurs critiques
   - Dashboard d'erreurs pour admin

6. **Tests unitaires**
   - Tests backend (handlers, services, modèles)
   - Tests d'intégration API
   - Tests end-to-end frontend
   - Couverture de code > 80%

7. **Amélioration UI**
   - Lecteur vidéo amélioré (contrôles, raccourcis clavier)
   - Page de détails vidéo enrichie
   - Filtrage par catégories sur page d'accueil
   - Interface admin améliorée

### Priorité Basse 🟢

8. **Mode sombre/clair**
   - Toggle thème dans profil utilisateur
   - Persistance préférence (localStorage)
   - Thème système automatique
   - Transition fluide entre thèmes

9. **Notifications en temps réel**
   - WebSocket pour notifications push
   - Notifications fin de téléchargement
   - Notifications fin de transcodage
   - Notifications erreurs critiques
   - Centre de notifications dans UI

10. **Système de favoris**
    - Ajout/suppression favoris
    - Page dédiée "Mes Favoris"
    - Export/import favoris
    - Partage de favoris (optionnel)

11. **Tests de charge**
    - Tests avec JMeter/k6
    - Optimisation performances
    - Cache Redis pour requêtes fréquentes
    - CDN pour assets statiques (optionnel)

## 📚 Documentation Disponible

- `README.md` - Documentation principale du projet
- `docs/SOLUTION-403-RESOLU.md` - Solution au problème 403 Forbidden
- `docs/INVESTIGATION-403.md` - Investigation du problème 403
- `docs/WORKAROUND-403.md` - Workaround temporaire (historique)
- `docs/SOLUTION-FINALE.md` - Solutions testées (historique)
- `docs/diagnostic-403-report-*.txt` - Rapports de diagnostic

## 🛠️ Scripts Disponibles

- `scripts/test-api.sh` - Tests de l'API (CORS, login, etc.)
- `scripts/test-all.sh` - Tests complets de l'application (toutes les fonctionnalités)
- `scripts/diagnose-403.sh` - Diagnostic du problème 403 (historique)

## 🎛️ Commandes Make Disponibles

- `make install-torrents` - Installer et démarrer tous les clients torrent
- `make test-all` - Exécuter tous les tests complets de l'application
- `make test-api` - Test rapide de l'API
- `make status` - Statut des services Docker
- `make help` - Afficher toutes les commandes disponibles

## ✅ Checklist de Déploiement

Avant de déployer en production :

- [ ] Configurer les variables d'environnement de production
- [ ] Configurer les clients torrent en production
- [ ] Tester tous les flux (upload, téléchargement, streaming)
- [ ] Configurer les backups de la base de données
- [ ] Configurer les logs et monitoring
- [ ] Tester les performances sous charge
- [ ] Sécuriser les endpoints sensibles
- [ ] Configurer HTTPS/SSL
- [ ] Documenter la configuration de production
