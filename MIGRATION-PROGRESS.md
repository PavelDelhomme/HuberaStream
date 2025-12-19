# Progression de la Migration Next.js

## ✅ Pages Migrées

### Pages Publiques
- [x] **Page d'accueil** (`/`) - ISR avec revalidation 60s
- [x] **Login** (`/login`) - Client Component avec authentification
- [x] **Register** (`/register`) - Client Component avec inscription
- [x] **Search** (`/search`) - ISR avec revalidation 60s

### Pages Utilisateur
- [x] **Profile** (`/user/profile`) - Client Component avec statistiques
- [x] **History** (`/user/history`) - Client Component avec historique de visionnage
- [x] **Favorites** (`/user/favorites`) - Client Component avec favoris
- [x] **Playlists** (`/user/playlists`) - Client Component avec gestion des listes

### Pages Vidéo
- [x] **VideoPlayer** (`/video/[id]`) - **SSR avec métadonnées dynamiques** ✅

### Composants
- [x] **Navigation** - Client Component avec gestion de l'authentification
- [x] **Services API** - Configuration axios avec intercepteurs
- [x] **Service Auth** - Gestion de l'authentification avec cookies

## 🚧 Pages à Migrer

### Pages Utilisateur
- [ ] **Profile** (`/user/profile`)
- [ ] **History** (`/user/history`)
- [ ] **Favorites** (`/user/favorites`)
- [ ] **Playlists** (`/user/playlists`)

### Pages Vidéo
- [x] **VideoPlayer** (`/video/[id]`) - **SSR avec métadonnées dynamiques** ✅
- [ ] **Upload** (`/video/upload`)
- [ ] **DVDExtract** (`/dvd-extract`)

### Pages Admin
- [x] **AdminLayout** (`/admin/layout`) - Layout avec sidebar et protection
- [x] **AdminDashboard** (`/admin/dashboard`) - Tableau de bord avec statistiques
- [x] **AdminUsers** (`/admin/users`) - Gestion des utilisateurs avec validation
- [x] **AdminVideos** (`/admin/videos`) - Gestion des vidéos avec nettoyage invalides
- [x] **AdminTorrents** (`/admin/torrents`) - Gestion des torrents avec auto-refresh
- [x] **AdminTorrentConfigs** (`/admin/torrent-configs`) - Configurations de recherche torrent
- [x] **AdminTorrentCandidates** (`/admin/torrent-candidates`) - Candidats torrent avec filtres
- [x] **AdminTorrentIndexes** (`/admin/torrent-indexes`) - Index torrent avec tests
- [x] **AdminTorrentClients** (`/admin/torrent-clients`) - Clients torrent avec tests
- [x] **AdminCategories** (`/admin/categories`) - Gestion des catégories
- [x] **AdminMediaSettings** (`/admin/media-settings`) - Paramètres média et transcodage
- [ ] **AdminVideos** (`/admin/videos`)
- [ ] **AdminTorrents** (`/admin/torrents`)
- [ ] **AdminTorrentConfigs** (`/admin/torrent-configs`)
- [ ] **AdminTorrentCandidates** (`/admin/torrent-candidates`)
- [ ] **AdminTorrentIndexes** (`/admin/torrent-indexes`)
- [ ] **AdminTorrentClients** (`/admin/torrent-clients`)
- [ ] **AdminCategories** (`/admin/categories`)
- [ ] **AdminMediaSettings** (`/admin/media-settings`)

### Composants à Migrer
- [ ] **VideoGrid** - Affichage des vidéos en grille
- [ ] **VideoPlayer** - Lecteur vidéo
- [ ] **LoadingSpinner** - Indicateur de chargement
- [ ] **CandidatesModal** - Modal pour les candidats torrent

## 📋 Configuration SSR/SSG

### Pages Statiques (ISR) ✅
- [x] Page d'accueil (`/`) - Revalidation toutes les 60 secondes
- [x] Page de recherche (`/search`) - Revalidation toutes les 60 secondes

### Pages Dynamiques (SSR) ✅
- [x] Page vidéo (`/video/[id]`) - SSR avec métadonnées dynamiques et génération de metadata SEO
- [x] Middleware d'authentification - Protection des routes admin et utilisateur

### API Routes (si nécessaire)
- [ ] Proxy API pour éviter CORS
- [ ] Routes d'authentification

## 🛠️ Commandes Makefile

Toutes les commandes sont disponibles via le Makefile :

```bash
# Installer les dépendances Next.js
make install-nextjs

# Démarrer en mode développement
make dev-nextjs

# Build pour production
make build-nextjs

# Démarrer avec Docker (profile nextjs)
make up-nextjs
```

## 📝 Notes Techniques

### Structure App Router
- Utilisation de l'App Router de Next.js 13+
- Client Components avec `'use client'` pour l'interactivité
- Server Components par défaut pour le SSR

### Authentification
- Utilisation de cookies HTTP-only (à implémenter côté serveur)
- Service `authService` pour la gestion côté client
- Middleware Next.js pour protéger les routes (à implémenter)

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

