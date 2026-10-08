# Plan de Migration vers Next.js

## ✅ Migration Complétée - Next.js est maintenant le Frontend Principal

**Date**: 19 décembre 2025

### Changements Effectués

1. **Nginx Configuration**
   - Nginx route maintenant vers `frontend-nextjs` au lieu de `frontend`
   - Next.js accessible sur **http://localhost:3300** (port principal)
   - Ancien frontend React désactivé par défaut

2. **Docker Compose**
   - `frontend-nextjs` : Service principal (actif par défaut)
   - `frontend` : Service legacy (profile `legacy`, désactivé par défaut)

3. **Makefile**
   - `make install-nextjs` : Installer dépendances Next.js
   - `make build-nextjs` : Build Next.js pour production
   - `make dev-nextjs` : Développement Next.js hors Docker
   - `make up-legacy` : Démarrer l'ancien frontend React si nécessaire

## 🚀 Utilisation

### Démarrage Normal (Next.js)
```bash
make up
# Accès: http://localhost:3300
```

### Développement Next.js
```bash
make dev-nextjs
# Accès: http://localhost:3000
```

### Build Production
```bash
make build-nextjs
make up
```

### Utiliser l'Ancien Frontend (si nécessaire)
```bash
make up-legacy
# Accès: http://localhost:3304
```

## 📋 Pages Migrées

### ✅ Pages Publiques
- [x] Page d'accueil (`/`)
- [x] Login (`/login`)
- [x] Register (`/register`)
- [x] Search (`/search`)

### 🚧 Pages à Migrer
- [ ] VideoPlayer (`/video/[id]`) - **PRIORITÉ HAUTE**
- [ ] Pages utilisateur (Profile, History, Favorites, Playlists)
- [ ] Pages admin (Dashboard, Users, Videos, etc.)
- [ ] Upload et DVD Extract

## 🔧 Configuration

### Variables d'Environnement
- `NEXT_PUBLIC_API_URL`: URL de l'API (http://localhost:3300/api)

### Ports
- **3300** : Nginx (point d'entrée principal) → Next.js
- **3301** : Backend API direct
- **3304** : Frontend React legacy (si activé)

## 📝 Notes Techniques

### Structure App Router
- Utilisation de l'App Router de Next.js 13+
- Client Components avec `'use client'` pour l'interactivité
- Server Components par défaut pour le SSR

### Authentification
- Utilisation de cookies via `js-cookie`
- Service `authService` pour la gestion côté client
- Middleware Next.js à implémenter pour protéger les routes

### API
- Service `api.ts` avec intercepteurs axios
- Configuration automatique du token depuis les cookies
- Gestion des erreurs 401/403 avec redirection

## 🎯 Prochaines Étapes

1. **Migrer VideoPlayer** - Page la plus importante avec SSR
2. **Créer middleware d'authentification** - Protection des routes admin
3. **Migrer les pages admin** - Progressivement
4. **Configurer SSR/SSG** - Optimiser les performances
5. **Tests** - Vérifier toutes les fonctionnalités

## 🔄 Retour à l'Ancien Système

Si vous devez temporairement revenir à l'ancien frontend React :

1. Modifier `nginx/nginx.conf` :
   ```nginx
   upstream frontend {
       server frontend:3000;  # Au lieu de frontend-nextjs:3000
   }
   ```

2. Redémarrer Nginx :
   ```bash
   docker-compose restart nginx
   ```

3. Démarrer l'ancien frontend :
   ```bash
   make up-legacy
   ```
