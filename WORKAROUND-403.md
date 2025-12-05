# Workaround Temporaire - Problème 403 Forbidden

## Solution Mise en Place

Le frontend communique maintenant **directement avec le backend** (port 3301) au lieu de passer par Nginx (port 3300).

### Modifications Effectuées

1. **`frontend/src/services/api.js`**
   - Changé `apiBaseURL` de `http://localhost:3300/api` vers `http://localhost:3301/api`

2. **`docker-compose.yml`**
   - Mis à jour `VITE_API_URL` et `REACT_APP_API_URL` pour pointer vers le backend direct

### Avantages

- ✅ Les requêtes POST fonctionnent maintenant
- ✅ Pas de problème 403 Forbidden
- ✅ Solution immédiate et fonctionnelle

### Inconvénients

- ⚠️ Le frontend ne passe plus par Nginx (pas de reverse proxy)
- ⚠️ CORS doit être géré par le backend directement
- ⚠️ Solution temporaire en attendant la résolution du problème avec Gin

### Pour Réactiver Nginx (Quand le Problème Sera Résolu)

1. Modifier `frontend/src/services/api.js` :
   ```javascript
   const apiBaseURL = import.meta.env.VITE_API_URL || 'http://localhost:3300/api'
   ```

2. Modifier `docker-compose.yml` :
   ```yaml
   environment:
     REACT_APP_API_URL: "http://localhost:3300/api"
     VITE_API_URL: "http://localhost:3300/api"
   ```

3. Redémarrer le frontend :
   ```bash
   docker-compose restart frontend
   ```

## État Actuel

- ✅ Backend direct (port 3301) : FONCTIONNE
- ✅ Frontend → Backend direct : FONCTIONNE
- ❌ Frontend → Nginx → Backend : ÉCHEC (403 Forbidden)

## Problème Sous-Jacent

Le problème persiste avec Gin v1.11.0 et `SetTrustedProxies`. Les requêtes POST via Nginx retournent toujours un 403 Forbidden en quelques microsecondes, avant même d'atteindre les middlewares.

## Prochaines Étapes

1. Investiguer plus en profondeur le code source de Gin
2. Tester avec un autre reverse proxy (Caddy, Traefik)
3. Créer un middleware personnalisé qui contourne la vérification des proxies
4. Ouvrir un issue sur le dépôt GitHub de Gin si le problème persiste

