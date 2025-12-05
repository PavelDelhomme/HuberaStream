# StreamMake

Plateforme de streaming personnelle avec backend Go, frontend React, et support multi-formats (HLS, DASH, WebRTC, progressive).

## Architecture

- **Backend**: Go avec Gin framework, GORM, PostgreSQL
- **Frontend**: React avec Vite, React Router, React Player
- **Base de données**: PostgreSQL
- **Cache**: Redis
- **Reverse Proxy**: Nginx
- **Containerisation**: Docker Compose

## Prérequis

- Docker et Docker Compose installés
- FFmpeg installé (pour le transcodage vidéo)

## Installation

1. Cloner le projet
```bash
git clone git@github.com:PavelDelhomme/StreamMake.git
cd StreamMake
```

2. Basculer sur la branche de développement (optionnel)
```bash
git checkout dev
```

2. Démarrer les services avec Docker Compose
```bash
docker-compose up -d
```

3. Les services seront disponibles sur:
   - Frontend (via Nginx): http://localhost:3300
   - Frontend (direct, dev): http://localhost:3304
   - Backend API: http://localhost:3301
   - PostgreSQL: localhost:3302
   - Redis: localhost:3303
   - Debugger (Delve): localhost:2345

## Utilisation du Makefile

Le projet inclut un Makefile pour faciliter les opérations courantes :

```bash
make help          # Affiche toutes les commandes disponibles
make build         # Construit les images Docker
make up            # Démarre tous les services
make down          # Arrête tous les services
make restart       # Redémarre tous les services
make logs          # Affiche les logs de tous les services
make logs-backend  # Affiche les logs du backend uniquement
make logs-frontend # Affiche les logs du frontend uniquement
make clean         # Supprime les conteneurs, volumes et images
make dev-backend   # Lance le backend en mode développement (sans Docker)
make dev-frontend  # Lance le frontend en mode développement (sans Docker)
make debug-backend # Lance le backend avec Delve (debugger sur port 2345)
make test          # Lance les tests
make install-deps  # Installe les dépendances
make status        # Affiche le statut des services
make rebuild       # Reconstruit et redémarre les services
```

## Structure du projet

```
StreamMake/
├── backend/          # Application Go
├── frontend/         # Application React
├── database/         # Migrations SQL
├── nginx/            # Configuration Nginx
└── docker-compose.yml
```

## Fonctionnalités

- Gestion des utilisateurs (inscription, connexion)
- Upload et téléchargement de vidéos depuis des plateformes de streaming
- Transcodage automatique en multiples formats (HLS, DASH, progressive)
- Lecture vidéo multi-format
- Playlists personnelles
- Historique de visionnage
- Recherche de vidéos

## Développement

### Backend

```bash
cd backend
go mod tidy
go run cmd/server/main.go
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Configuration

Les variables d'environnement peuvent être configurées dans `docker-compose.yml` ou via des fichiers `.env`.

## Workflow Git

Le projet utilise un workflow avec deux branches principales :
- **`main`** : Branche de production, code stable et testé
- **`dev`** : Branche de développement, travail actif et nouvelles fonctionnalités

### Travailler sur le projet

```bash
# Basculer sur la branche dev
git checkout dev

# Créer une nouvelle branche de fonctionnalité
git checkout -b feature/ma-fonctionnalite

# Après développement, merger dans dev
git checkout dev
git merge feature/ma-fonctionnalite
git push origin dev

# Quand dev est stable, merger dans main pour la production
git checkout main
git merge dev
git push origin main
```

## Notes

- Le système de téléchargement vidéo est basique et nécessite des améliorations pour supporter différentes plateformes
- WebRTC n'est pas encore complètement implémenté
- Le transcodage vidéo nécessite FFmpeg installé dans le conteneur

