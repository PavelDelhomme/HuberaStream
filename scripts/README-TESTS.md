# Guide des Tests StreamMake

## Commandes de Test Disponibles

### 1. `make check` - Vérification Complète
Vérifie tous les éléments du projet avant le démarrage :
- ✅ Prérequis (Docker, Go, Node.js, FFmpeg)
- ✅ Structure des fichiers
- ✅ Compilation backend/frontend
- ✅ Variables d'environnement
- ✅ Services Docker
- ✅ Base de données
- ✅ Routes API
- ✅ Erreurs de code

**Usage :**
```bash
make check
```

### 2. `make test` - Tests Unitaires Backend
Exécute les tests unitaires Go du backend.

**Usage :**
```bash
make test
```

### 3. `make test-api` - Tests API
Teste les endpoints de l'API (authentification, CORS, routes).

**Note :** Les services Docker seront démarrés automatiquement si nécessaire.

**Usage :**
```bash
make test-api
```

**Si les services ne démarrent pas automatiquement :**
```bash
make up    # Démarrer les services manuellement
make test-api
```

### 4. `make test-metadata` - Test Recherche Métadonnées
Teste la recherche de métadonnées pour le fichier `RED_ONE.webm`.

**Note :** Les services Docker seront démarrés automatiquement si nécessaire.

**Usage :**
```bash
make test-metadata
```

**Si les services ne démarrent pas automatiquement :**
```bash
make up    # Démarrer les services manuellement
make test-metadata
```

### 5. `make test-all` - Tests Complets
Exécute tous les tests complets de l'application (API + fonctionnalités).

**Note :** Les services Docker seront démarrés automatiquement si nécessaire.

**Usage :**
```bash
make test-all
```

**Si les services ne démarrent pas automatiquement :**
```bash
make up    # Démarrer les services manuellement
make test-all
```

## Résolution des Problèmes

### Erreur : "compose build requires buildx 0.17 or later"

Cette erreur survient lorsque Docker essaie de builder les images mais que buildx n'est pas à jour ou que la version de Docker est trop ancienne.

**Solution 1 : Créer les images manuellement (recommandé)**
```bash
# Créer les images Docker sans buildx
make setup-images

# Démarrer les services
docker compose up -d --no-build
# ou
make up

# Exécuter les tests
make test-api
```

**Solution 2 : Mettre à jour Docker**
```bash
# Mettre à jour Docker vers la dernière version
# Puis :
make up
make test-api
```

**Solution 3 : Utiliser la compilation locale uniquement**
```bash
# Compiler le backend localement
make rebuild-backend-local

# Créer l'image Docker minimale manuellement
cd .backend-bin
cat > Dockerfile << 'EOF'
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata ffmpeg
COPY main /app/main
RUN chmod +x /app/main
WORKDIR /app
RUN mkdir -p /app/storage/videos /app/storage/thumbnails
CMD ["/app/main"]
EOF
DOCKER_BUILDKIT=0 docker build -t streammake-backend:latest .
cd ..

# Démarrer les services
docker compose up -d --no-build
```

### Erreur : "Les services Docker ne sont pas démarrés"

Les scripts de test tentent de démarrer les services automatiquement, mais si cela échoue :

**Solution :**
```bash
# Démarrer les services manuellement
make up

# Attendre que les services soient prêts (environ 20 secondes)
sleep 20

# Exécuter les tests
make test-api
```

### Erreur : "L'API n'est pas accessible"

Si l'API ne répond pas après le démarrage des services :

**Solution :**
1. Vérifier que les services sont bien démarrés :
   ```bash
   make status
   ```

2. Vérifier les logs pour identifier le problème :
   ```bash
   make logs-backend
   ```

3. Redémarrer les services :
   ```bash
   make restart
   ```

## Ordre Recommandé pour les Tests

1. **Vérification initiale :**
   ```bash
   make check
   ```

2. **Démarrer les services :**
   ```bash
   make up
   ```

3. **Attendre que les services soient prêts :**
   ```bash
   sleep 20
   ```

4. **Exécuter les tests :**
   ```bash
   make test-api        # Tests API de base
   make test-metadata   # Test recherche métadonnées
   make test-all        # Tests complets
   ```

## Notes Importantes

- Les tests nécessitent que les services Docker soient démarrés
- Les scripts tentent de démarrer les services automatiquement, mais cela peut échouer si les images Docker n'existent pas
- En cas d'échec du démarrage automatique, utilisez `make up` manuellement
- Les tests peuvent prendre plusieurs minutes selon la vitesse de votre machine

