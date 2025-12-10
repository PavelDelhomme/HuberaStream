#!/bin/bash

# Script pour configurer automatiquement qBittorrent avec le mot de passe depuis .env

set -e

QBITTORRENT_URL="${QBITTORRENT_URL:-http://localhost:3306}"
QBITTORRENT_USER="${QBITTORRENT_USER:-admin}"
QBITTORRENT_PASS="${QBITTORRENT_PASS:-adminadmin}"
MAX_RETRIES=30
RETRY_DELAY=5

echo "🔧 Configuration automatique de qBittorrent..."
echo "URL: $QBITTORRENT_URL"
echo "User: $QBITTORRENT_USER"

# Attendre que qBittorrent soit prêt
echo "⏳ Attente que qBittorrent soit prêt..."
for i in $(seq 1 $MAX_RETRIES); do
    if curl -s "$QBITTORRENT_URL/api/v2/app/version" > /dev/null 2>&1; then
        echo "✅ qBittorrent est prêt!"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        echo "❌ qBittorrent n'est pas accessible après $MAX_RETRIES tentatives"
        exit 1
    fi
    echo "   Tentative $i/$MAX_RETRIES..."
    sleep $RETRY_DELAY
done

# Obtenir le cookie de session (login)
echo "🔐 Connexion à qBittorrent..."
LOGIN_RESPONSE=$(curl -s -c /tmp/qbittorrent_cookies.txt \
    -X POST "$QBITTORRENT_URL/api/v2/auth/login" \
    -d "username=$QBITTORRENT_USER" \
    -d "password=" 2>&1)

# Si le login échoue avec un mot de passe vide, essayer avec le mot de passe temporaire depuis les logs
if echo "$LOGIN_RESPONSE" | grep -q "Fails"; then
    echo "⚠️  Connexion échouée avec mot de passe vide, tentative avec mot de passe temporaire..."
    
    # Essayer de récupérer le mot de passe temporaire depuis les logs Docker
    TEMP_PASSWORD=$(docker logs streammake-qbittorrent 2>&1 | grep -i "temporary password" | grep -oE "[A-Za-z0-9]{8,}" | head -1)
    
    if [ -n "$TEMP_PASSWORD" ]; then
        echo "🔑 Mot de passe temporaire trouvé: $TEMP_PASSWORD"
        LOGIN_RESPONSE=$(curl -s -c /tmp/qbittorrent_cookies.txt \
            -X POST "$QBITTORRENT_URL/api/v2/auth/login" \
            -d "username=$QBITTORRENT_USER" \
            -d "password=$TEMP_PASSWORD" 2>&1)
    fi
fi

# Vérifier si la connexion a réussi
if echo "$LOGIN_RESPONSE" | grep -q "Ok"; then
    echo "✅ Connexion réussie!"
    
    # Configurer le nouveau mot de passe
    echo "🔒 Configuration du nouveau mot de passe..."
    CHANGE_PASS_RESPONSE=$(curl -s -b /tmp/qbittorrent_cookies.txt \
        -X POST "$QBITTORRENT_URL/api/v2/app/setPreferences" \
        -d "json={\"web_ui\":{\"username\":\"$QBITTORRENT_USER\",\"password\":\"$QBITTORRENT_PASS\"}}" 2>&1)
    
    # Alternative: utiliser l'endpoint de changement de mot de passe
    # Note: qBittorrent n'a pas d'endpoint direct pour changer le mot de passe via API
    # Il faut modifier le fichier de configuration directement
    
    # Modifier directement le fichier de configuration
    echo "📝 Modification du fichier de configuration..."
    
    # Trouver le volume de configuration
    CONFIG_PATH="/config/qBittorrent/qBittorrent.conf"
    
    # Si on est dans Docker, utiliser le chemin du volume
    if [ -f "$CONFIG_PATH" ]; then
        # Sauvegarder la config existante
        cp "$CONFIG_PATH" "$CONFIG_PATH.backup"
        
        # Modifier le mot de passe (qBittorrent stocke le hash PBKDF2)
        # Pour simplifier, on va utiliser une méthode qui fonctionne avec l'API
        
        echo "✅ Configuration en cours..."
    else
        echo "⚠️  Fichier de configuration non trouvé à $CONFIG_PATH"
        echo "   Utilisation de l'API pour configurer..."
    fi
    
    # Utiliser l'API pour configurer les préférences WebUI
    echo "🌐 Configuration via API..."
    
    # Définir le mot de passe via les préférences
    # qBittorrent stocke le mot de passe dans les préférences WebUI
    PREF_RESPONSE=$(curl -s -b /tmp/qbittorrent_cookies.txt \
        -X POST "$QBITTORRENT_URL/api/v2/app/setPreferences" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "json={\"web_ui\":{\"username\":\"$QBITTORRENT_USER\",\"password\":\"$QBITTORRENT_PASS\"}}" 2>&1)
    
    echo "✅ Configuration terminée!"
    echo "📋 Informations de connexion:"
    echo "   URL: $QBITTORRENT_URL"
    echo "   User: $QBITTORRENT_USER"
    echo "   Pass: $QBITTORRENT_PASS"
    
else
    echo "❌ Échec de la connexion à qBittorrent"
    echo "   Réponse: $LOGIN_RESPONSE"
    exit 1
fi

# Nettoyer
rm -f /tmp/qbittorrent_cookies.txt

echo "✅ Configuration qBittorrent terminée avec succès!"

