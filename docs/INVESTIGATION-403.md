# Investigation du Problème 403 Forbidden

## Résumé du Diagnostic

### Observations Clés

1. **Backend direct (port 3301)** : ✅ FONCTIONNE
2. **Test interne Nginx → Backend** : ✅ FONCTIONNE (RemoteAddr = 172.19.0.6)
3. **GET via Nginx (port 3300)** : ✅ FONCTIONNE
4. **OPTIONS (preflight CORS)** : ✅ FONCTIONNE
5. **POST via Nginx depuis extérieur** : ❌ 403 Forbidden (RemoteAddr = 172.19.0.1)

### Détails Techniques

- Le 403 est retourné en **quelques microsecondes** (8-14µs), ce qui suggère un blocage **AVANT** les middlewares
- **Aucun log EARLY-DEBUG n'apparaît**, confirmant que la requête est bloquée avant d'atteindre les middlewares
- Une requête POST a réussi avec RemoteAddr = 172.19.0.1 (ligne 81 du rapport), suggérant un problème intermittent ou conditionnel

### Tentatives de Résolution

1. ✅ `SetTrustedProxies(nil)` - Échec
2. ✅ `SetTrustedProxies([]string{})` - Échec
3. ✅ `SetTrustedProxies([...réseaux Docker...])` - Échec
4. ✅ Configuration `TrustedPlatform = "X-Forwarded-For"` - Échec
5. ✅ Configuration `TrustedPlatform = ""` - Échec

## Hypothèses

### Hypothèse 1: Problème avec le body de la requête POST
Gin pourrait vérifier quelque chose de spécifique pour les requêtes POST avec body qui n'est pas vérifié pour les GET.

### Hypothèse 2: Bug connu de Gin v1.9.1
Il pourrait y avoir un bug spécifique dans Gin v1.9.1 avec `SetTrustedProxies` et les requêtes POST.

### Hypothèse 3: Nginx ne transmet pas correctement le body
Nginx pourrait avoir un problème avec la transmission du body pour les requêtes POST, mais cela semble peu probable car le test interne fonctionne.

## Prochaines Étapes Recommandées

### Option 1: Mise à jour de Gin
```bash
cd backend
go get -u github.com/gin-gonic/gin@latest
go mod tidy
```

### Option 2: Workaround temporaire - Frontend direct vers Backend
Modifier le frontend pour utiliser directement le backend (port 3301) au lieu de passer par Nginx (port 3300).

### Option 3: Investigation plus poussée
- Vérifier si le problème vient de la façon dont Nginx transmet les headers
- Tester avec un autre reverse proxy (Caddy, Traefik)
- Examiner le code source de Gin pour comprendre pourquoi le 403 est retourné si rapidement

### Option 4: Patch personnalisé
Créer un middleware personnalisé qui contourne la vérification des proxies de Gin.

## Fichiers de Diagnostic

- `diagnostic-403-report-*.txt` : Rapport complet du diagnostic
- `diagnose-403.sh` : Script de diagnostic réutilisable

## Commandes Utiles

```bash
# Relancer le diagnostic
./diagnose-403.sh

# Tester directement le backend
curl -X POST http://localhost:3301/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}'

# Tester via Nginx
curl -X POST http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}'

# Voir les logs backend
docker-compose logs backend | grep -E "POST.*login|403|200"

# Voir les logs Nginx
docker-compose logs nginx | grep "POST.*login"
```

