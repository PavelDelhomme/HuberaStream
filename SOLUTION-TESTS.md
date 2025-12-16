# Solution pour les Tests StreamMake

## Problème

Les commandes `make test-api`, `make test-metadata`, et `make test-all` échouent avec l'erreur :
```
compose build requires buildx 0.17 or later
```

Cela se produit car :
1. Les images Docker n'existent pas encore
2. Docker essaie de les construire mais buildx n'est pas à jour ou Docker est trop ancien

## Solution Rapide

### Étape 1 : Créer les images Docker
```bash
make setup-images
```

Cette commande :
- Compile le backend localement
- Crée une image Docker minimale sans utiliser buildx
- Prépare tout pour démarrer les services

### Étape 2 : Démarrer les services
```bash
docker compose up -d --no-build
```

Ou simplement :
```bash
make up
```

### Étape 3 : Exécuter les tests
```bash
make test-api        # Tests API
make test-metadata   # Test métadonnées
make test-all        # Tests complets
```

## Alternative : Mise à jour de Docker

Si vous préférez mettre à jour Docker :
```bash
# Sur Manjaro/Arch
sudo pacman -Syu docker docker-compose

# Puis redémarrer Docker
sudo systemctl restart docker

# Ensuite
make up
make test-api
```

## Vérification

Pour vérifier que tout fonctionne :
```bash
# Vérifier les images
docker images | grep streammake

# Vérifier les services
make status

# Vérifier l'API
curl http://localhost:3300/api/categories
```

## Commandes Utiles

```bash
make setup-images      # Créer les images Docker
make up                # Démarrer les services
make down              # Arrêter les services
make status            # Statut des services
make logs              # Voir les logs
make test-api          # Tests API
make test-metadata     # Test métadonnées
make test-all          # Tests complets
```
