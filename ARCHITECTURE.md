# Architecture StreamMake

## Vue d'ensemble

StreamMake est une plateforme de streaming vidéo avec trois composants principaux :

1. **Backend API** (Go + Gin)
2. **Frontend Web** (React + Vite)
3. **Application Mobile** (À développer)

## Architecture Actuelle

```
┌─────────────────────────────────────────────────────────────┐
│                      NGINX (Port 3300)                      │
│                    Reverse Proxy                            │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                    ┌───────▼────────┐
│  Frontend Web  │                    │  Backend API   │
│  (React/Vite)  │                    │  (Go/Gin)      │
│  Port 3000     │                    │  Port 8080      │
└────────────────┘                    └────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
            ┌───────▼──────┐          ┌───────▼──────┐         ┌───────▼──────┐
            │  PostgreSQL  │          │    Redis     │         │   Storage    │
            │  Port 3302  │          │  Port 3303  │         │   (Volumes)  │
            └─────────────┘          └─────────────┘         └──────────────┘
```

## Composants

### 1. Backend API (Go + Gin)

**Localisation**: `backend/`

**Technologies**:
- Go 1.24
- Gin Framework
- GORM (ORM)
- JWT pour l'authentification
- FFmpeg pour le transcodage vidéo

**Endpoints principaux**:
- `/api/auth/*` - Authentification
- `/api/videos/*` - Gestion des vidéos
- `/api/admin/*` - Administration
- `/api/playlists/*` - Listes personnelles
- `/api/history/*` - Historique de visionnage

**Port**: 8080 (interne), 3301 (exposé)

### 2. Frontend Web (React + Vite)

**Localisation**: `frontend/`

**Technologies**:
- React 18
- Vite
- React Router
- Axios
- Context API pour l'authentification

**Port**: 3000 (interne), 3300 via Nginx (exposé)

**Pages principales**:
- `/` - Page d'accueil
- `/login` - Connexion
- `/register` - Inscription
- `/video/:id` - Lecteur vidéo
- `/upload` - Upload vidéo (admin)
- `/admin/*` - Interface admin

### 3. Application Mobile (À développer)

**Technologies recommandées**:
- **React Native** (partage de code avec React)
- **Flutter** (alternative)
- **Ionic** (alternative)

**Structure proposée**:
```
mobile/
├── src/
│   ├── screens/        # Écrans de l'application
│   ├── components/     # Composants réutilisables
│   ├── services/       # Services API
│   ├── navigation/     # Navigation
│   └── context/        # Context API
├── android/
├── ios/
└── package.json
```

**Fonctionnalités**:
- Authentification
- Navigation vidéos
- Lecteur vidéo
- Upload vidéo (admin)
- Recherche
- Historique
- Listes personnelles

## Base de données

**PostgreSQL** - Port 3302

**Tables principales**:
- `users` - Utilisateurs
- `videos` - Vidéos
- `categories` - Catégories
- `playlists` - Listes personnelles
- `watch_history` - Historique de visionnage
- `torrents` - Torrents
- `torrent_configs` - Configurations torrent

## Services externes

### Clients Torrent
- **Transmission** - Port 3305
- **qBittorrent** - Port 3306
- **Deluge** - Port 3307
- **rTorrent** - Port 3308

## Communication API

### Format des réponses

**Succès**:
```json
{
  "data": {...},
  "message": "Success"
}
```

**Erreur**:
```json
{
  "error": "Message d'erreur"
}
```

### Authentification

Toutes les routes protégées nécessitent un header :
```
Authorization: Bearer <token>
```

## Déploiement

### Développement
```bash
make up          # Démarrer tous les services
make dev-backend # Backend en mode dev
make dev-frontend # Frontend en mode dev
```

### Production
- Docker Compose pour tous les services
- Nginx comme reverse proxy
- Volumes Docker pour le stockage

## Tests

### Tests Backend
```bash
make test        # Tests Go
make test-api    # Tests API
make test-all    # Tests complets
```

### Tests Frontend
```bash
cd frontend
npm test
```

## Prochaines étapes

1. ✅ Corriger les vidéos null dans le frontend
2. ✅ Tester l'upload de vidéos
3. ⏳ Créer l'application mobile (React Native)
4. ⏳ Ajouter les tests E2E
5. ⏳ Optimiser les performances

