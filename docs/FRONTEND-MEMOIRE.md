# Mémoire et alternatives — frontend StreamMake (Next.js)

Ce document résume **pourquoi le frontend peut consommer beaucoup de RAM** en Docker et **ce que changent** les autres technologies (sans les implémenter dans le dépôt).

## 1. Pourquoi `make up` ne démarrait pas correctement le frontend

Causes fréquentes (corrigées dans le projet) :

| Cause | Explication |
|--------|-------------|
| **`docker compose up -d --no-build`** | Si l’image Next n’existait pas ou était invalide, Compose ne reconstruisait pas. Le `Makefile` utilise désormais **`up -d`** (build si nécessaire) et **échoue** si le build Next échoue. |
| **Volume anonyme `/app/node_modules`** | Au premier run, ce volume est **vide** : il écrase les `node_modules` de l’image. **`npm run dev`** plantait faute de paquets. **Correctif** : commande de démarrage qui lance **`npm ci`** si `node_modules/.bin/next` est absent. |
| **Healthcheck `wget`** | L’image `node:alpine` n’inclut pas toujours `wget`. **Correctif** : `apk add wget` dans `Dockerfile.dev` + `start_period` allongé (premier `npm ci` est long). |

## 2. Ordre de grandeur mémoire (Next.js en dev dans Docker)

| Composant | RAM typique (ordre de grandeur) |
|-----------|----------------------------------|
| **Node.js + V8** | ~80–200 Mo de base pour un process. |
| **Next.js `next dev`** | Très gourmand : **500 Mo–1,5 Go+** (compilation incrémentale, cache SWC, HMR, source maps). |
| **`node_modules` sur disque** | ~400–800 Mo ; la RAM monte encore lors du premier `npm ci` / transpilation. |
| **`.next/cache`** | Centaines de Mo possibles sur une grosse app. |

En **production** (`next build` + `next start`), la RAM est en général **nettement plus basse** qu’en `dev` (souvent ~150–400 Mo selon trafic et pages).

## 3. Leviers déjà adaptés à Next (sans changer de framework)

- **`output: 'standalone'`** (déjà dans `next.config.js`) : image Docker prod plus petite.
- **Dynamic import** des gros modules (`react-player`, éditeurs, charts) : réduit le JS initial et un peu la pression mémoire côté client.
- **Ne pas monter** tout le repo en prod : image multi-stage avec seulement `.next/standalone` + `node_modules` prod.
- **Désactiver le source map** en prod, limiter les workers de build en CI.

## 4. Alternatives technologiques (mémoire / perf)

| Stack | Mémoire côté serveur / build | Quand ça vaut le coup |
|-------|----------------------------|------------------------|
| **Vite + React (SPA)** | Dev : souvent **moins** que Next dev ; pas de SSR par défaut. | Admin 100 % client, pas besoin de SSR ni de routes serveur. |
| **Astro** | Très bas si pages statiques ; peu adapté à un **dashboard** lourd. | Site vitrine + peu d’interactivité. |
| **Remix / Qwik** | Variables ; intérêt surtout sur **TTI** ou données au bord. | Refonte majeure, équipe prête à migrer. |
| **Svelte / SvelteKit** | Bundle souvent plus léger que React. | Nouvelle app ou équipe Svelte. |

Pour **StreamMake**, l’admin est déjà en **Next client** : passer à Vite ferait surtout **gagner sur la DX dev et un peu sur le footprint**, au prix d’**une deuxième stack** (auth, routing, déploiement) à maintenir. Le ratio effort / gain est faible tant que le goulot est **`next dev` en Docker**, pas le framework en soi.

## 5. Recommandation pratique

- **Développement local** : `npm run dev` **sur la machine** ou frontend Docker avec **ressources suffisantes** (≥ 2 Go pour le service Next).  
- **Production** : image **Next `standalone`** ou build statique derrière nginx, **sans** bind-mount du code source.

Voir aussi **`docs/FRONTEND-STRATEGIE-COMPARATIVE.md`** : tableau Next / Vite / Remix / Svelte / Astro / Nuxt / Qwik, gains attendus et recommandation pour StreamMake (multimédia). Script de mesure : `bash scripts/benchmark-frontends.sh` (npm local) ou **`USE_DOCKER=1 bash scripts/benchmark-frontends.sh`** / `make benchmark-frontends-docker` pour tout exécuter dans Docker.