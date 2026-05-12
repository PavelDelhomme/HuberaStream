# StreamMake sur Portainer / VPS (Nginx Proxy Manager)

Ce document aligne StreamMake sur le même modèle que tes autres stacks (réseau `web` + conteneurs joignables par nom DNS).

## Rôles des composants

| Composant | Rôle |
|-----------|------|
| **nginx** (streammake) | Point d’entrée unique : API `/api/`, frontend Next.js, fichiers `/videos/`. C’est lui que NPM doit cibler. |
| **backend** | API Go (auth, vidéos, torrents, auto-download). Reste sur le réseau interne `streammake-network` sauf besoin particulier. |
| **Prowlarr / Radarr / Sonarr** | Leur **API et leur UI** sont propres à chaque outil. StreamMake **ne les remplace pas** : tu configures indexers + clients torrent dans Prowlarr, films dans Radarr, séries dans Sonarr. Le backend expose `GET /api/admin/media-stack/status` pour vérifier qu’ils répondent (monitoring / tests). |
| **Transmission / qBittorrent / …** | Clients de téléchargement ; Radarr/Sonarr y envoient les jobs une fois configurés. |

## Réseau `web` (Nginx Proxy Manager)

1. Crée un fichier d’environnement Portainer (ou `.env`) avec au minimum `JWT_SECRET`, `POSTGRES_PASSWORD`, `DB_PASSWORD` (même valeur que Postgres), et **sans** commiter ces valeurs.
2. Déploie avec l’overlay :

```bash
docker compose -f docker-compose.yml -f docker-compose.portainer.yml up -d
```

3. Dans **Nginx Proxy Manager**, ajoute un *Proxy Host* :

- **Domain** : `stream.delhomme.ovh` (exemple)
- **Forward hostname** : `streammake-nginx`
- **Forward port** : `80`
- **SSL** : Let’s Encrypt

Les services `cookingrecipes-api` et `streammake-nginx` doivent être sur le **même réseau Docker `web`** pour que le nom `streammake-nginx` se résolve.

## Suite *arr (optionnelle)

Ports **hôte** réservés pour éviter les collisions avec rTorrent (3310–3312) :

| Service   | Port hôte |
|-----------|-----------|
| Prowlarr  | 3326      |
| Radarr    | 3327      |
| Sonarr    | 3328      |

Démarrage :

```bash
docker compose --profile arr up -d
# ou depuis la racine du dépôt :
make up-arr
```

Dans `.env` (ou variables Portainer), une fois le profile `arr` actif :

```env
PROWLARR_URL=http://prowlarr:9696
RADARR_URL=http://radarr:7878
SONARR_URL=http://sonarr:8989
```

Volumes nommés : `prowlarr_config`, `radarr_config`, `sonarr_config` ; téléchargements partagés avec `torrent_storage` monté sur `/downloads` dans les trois services (à aligner avec le chemin configuré dans Radarr/Sonarr et le client torrent).

## Nettoyage côté serveur (recommandé)

- Tu as deux réseaux proches pour cooking-recipes (`cooking_recipes_network` et `cookingrecipes_network`) et des volumes « Unused » : après sauvegarde, supprime la stack morte et les volumes orphelins pour clarifier Portainer.
- **Ne colle jamais** de mots de passe en clair dans un compose versionné ; utilise les **secrets / variables** Portainer. Si un secret a été exposé (chat, capture d’écran), **fais-le tourner**.

## Tests

- `make test-api` — API derrière nginx.
- `make test-all` — inclut le statut media-stack (admin).
- `curl -s -H "Authorization: Bearer <token>" https://ton-domaine/api/admin/media-stack/status`

## Webhooks Radarr / Sonarr → StreamMake

1. Définis **`WEBHOOK_SECRET`** (long aléatoire) dans Portainer / `.env` pour le service **backend**, puis redémarre le backend.
2. Optionnel : **`PUBLIC_WEBHOOK_BASE_URL`** = URL publique sans `/api` (ex. `https://stream.delhomme.ovh`) pour que la page admin `/admin/webhooks` affiche les bonnes URL à copier.
3. Dans **Radarr** / **Sonarr** : *Settings* → *Connect* → *Webhook* — URL du type  
   `https://ton-domaine/api/webhooks/radarr?token=TON_SECRET` (idem `/webhooks/sonarr`). Méthode **POST**, corps **JSON**.  
   Si Radarr est sur le même réseau Docker et que tu ne passes pas par NPM, tu peux utiliser  
   `http://backend:8080/api/webhooks/radarr?token=...` (nom du service `backend` du compose StreamMake).
4. Historique : page admin **Webhooks *arr** ou `GET /api/admin/webhooks/events` (JWT admin).
5. **Import automatique** : `WEBHOOK_IMPORT_ENABLED=true` sur le backend. Les chemins `MovieFile.Path` / `EpisodeFile.Path` des webhooks doivent être **lisibles dans le conteneur backend** : le compose monte `torrent_storage:/downloads:ro`. Configure Radarr/Sonarr pour que les fichiers finaux résident sous un chemin cohérent (ex. sous `/downloads/...` dans chaque conteneur).

Les webhooks ne passent **pas** par l’API Radarr/Sonarr de StreamMake : ce sont **eux** qui appellent **ton** backend ; StreamMake enregistre l’événement (`media_webhook_events`) et peut **copier + transcoder** la vidéo si l’import auto est activé et le fichier est accessible.

