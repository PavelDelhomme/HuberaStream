# Hubera Stream sur Portainer / VPS

Produit live : catalogue légal **Pluto TV** + **Arte.tv** via `docker-compose.portainer.yml` (nginx, réseau `shared-network-copy`).

## Live

- Stack : `hubera-stream`
- Compose : `docker-compose.portainer.yml`
- URL : `https://stream.hubera.cloud/app/`
- `/admin/torrents` : landing légale (pas un client torrent)

Ne jamais : `docker compose down -v`, rename volumes, fusionner les Git users.

Le compose local `docker-compose.yml` (Postgres / backend Go / Next.js) n’embarque **pas** de client torrent ni de suite *arr.
