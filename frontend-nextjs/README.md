# StreamMake Frontend Next.js

Migration du frontend React/Vite vers Next.js pour bénéficier du SSR/SSG.

## Installation

```bash
npm install
```

## Développement

```bash
npm run dev
```

Le serveur de développement sera accessible sur http://localhost:3000

## Build Production

```bash
npm run build
npm start
```

## Structure

- `app/` - App Router (Next.js 13+)
- `components/` - Composants réutilisables
- `lib/` - Utilitaires et services (API, auth)
- `public/` - Assets statiques
- `styles/` - Styles globaux

## Migration en cours

Cette version est en cours de migration depuis React/Vite. Les fonctionnalités seront migrées progressivement.

## Docker

Pour démarrer avec Docker :

```bash
docker-compose --profile nextjs up frontend-nextjs
```

Ou pour tout démarrer :

```bash
docker-compose --profile nextjs up
```

