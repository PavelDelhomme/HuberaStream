#!/bin/bash

# Script de diagnostic pour le problème 403 Forbidden
# Génère un rapport détaillé pour investigation

REPORT_FILE="diagnostic-403-report-$(date +%Y%m%d-%H%M%S).txt"

echo "=========================================" > "$REPORT_FILE"
echo "RAPPORT DE DIAGNOSTIC - Problème 403 Forbidden" >> "$REPORT_FILE"
echo "Date: $(date)" >> "$REPORT_FILE"
echo "=========================================" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

echo "🔍 Démarrage du diagnostic..."
echo "📝 Rapport: $REPORT_FILE"
echo ""

# 1. Vérification des services Docker
echo "1. Vérification des services Docker..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose ps >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 2. Test direct au backend (port 3301)
echo "2. Test direct au backend (port 3301)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
echo "Commande: curl -X POST http://localhost:3301/api/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"dev@delhomme.ovh\",\"password\":\"AdminStreamMake2024!\"}'" >> "$REPORT_FILE"
RESPONSE_DIRECT=$(curl -s -w "\nHTTP_CODE:%{http_code}\nTIME:%{time_total}\n" -X POST http://localhost:3301/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}' 2>&1)
echo "$RESPONSE_DIRECT" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 3. Test via Nginx (port 3300)
echo "3. Test via Nginx (port 3300)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
echo "Commande: curl -X POST http://localhost:3300/api/auth/login -H 'Origin: http://localhost:3300' -H 'Content-Type: application/json' -d '{\"email\":\"dev@delhomme.ovh\",\"password\":\"AdminStreamMake2024!\"}'" >> "$REPORT_FILE"
RESPONSE_NGINX=$(curl -s -w "\nHTTP_CODE:%{http_code}\nTIME:%{time_total}\n" -X POST http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}' 2>&1)
echo "$RESPONSE_NGINX" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 4. Test OPTIONS (preflight CORS)
echo "4. Test OPTIONS (preflight CORS)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
echo "Commande: curl -X OPTIONS http://localhost:3300/api/auth/login -H 'Origin: http://localhost:3300' -v" >> "$REPORT_FILE"
curl -s -X OPTIONS http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v 2>&1 | grep -E "< HTTP|Access-Control" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 5. Headers détaillés de la requête via Nginx
echo "5. Headers détaillés de la requête via Nginx..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
curl -v -X POST http://localhost:3300/api/auth/login \
  -H "Origin: http://localhost:3300" \
  -H "Content-Type: application/json" \
  -d '{"email":"dev@delhomme.ovh","password":"AdminStreamMake2024!"}' 2>&1 | \
  grep -E "> |< |HTTP_CODE" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 6. Test depuis Nginx vers Backend
echo "6. Test depuis Nginx vers Backend (interne)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose exec -T nginx sh -c "curl -s -X POST http://backend:8080/api/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"dev@delhomme.ovh\",\"password\":\"AdminStreamMake2024!\"}'" >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 7. Logs backend récents
echo "7. Logs backend récents (20 dernières lignes)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose logs --tail=20 backend >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 8. Logs Nginx récents
echo "8. Logs Nginx récents (20 dernières lignes)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose logs --tail=20 nginx >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 9. Configuration Nginx
echo "9. Configuration Nginx (location /api/)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose exec -T nginx cat /etc/nginx/nginx.conf | grep -A 30 "location /api/" >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 10. Variables d'environnement backend
echo "10. Variables d'environnement backend..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose exec -T backend env | grep -E "DB_|REDIS_|JWT_|PORT|ENV" >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 11. Test GET pour comparaison
echo "11. Test GET /api/videos (pour comparaison)..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
echo "GET direct (port 3301):" >> "$REPORT_FILE"
curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:3301/api/videos >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"
echo "GET via Nginx (port 3300):" >> "$REPORT_FILE"
curl -s -w "\nHTTP_CODE:%{http_code}\n" http://localhost:3300/api/videos >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 12. Vérification de la connexion réseau Docker
echo "12. Vérification réseau Docker..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
echo "IP du backend:" >> "$REPORT_FILE"
docker-compose exec -T backend hostname -i >> "$REPORT_FILE" 2>&1
echo "IP de Nginx:" >> "$REPORT_FILE"
docker-compose exec -T nginx hostname -i >> "$REPORT_FILE" 2>&1
echo "Test de connexion Nginx -> Backend:" >> "$REPORT_FILE"
docker-compose exec -T nginx ping -c 2 backend >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 13. Vérification des headers envoyés par Nginx
echo "13. Headers envoyés par Nginx au backend..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
docker-compose exec -T nginx sh -c "curl -v -X POST http://backend:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -H 'X-Forwarded-For: 172.19.0.1' \
  -H 'X-Real-IP: 172.19.0.1' \
  -H 'Host: localhost:3300' \
  -d '{\"email\":\"dev@delhomme.ovh\",\"password\":\"AdminStreamMake2024!\"}'" 2>&1 | \
  grep -E "> |< HTTP" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 14. Version de Gin
echo "14. Version de Gin utilisée..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
grep "gin-gonic/gin" backend/go.mod >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 15. Configuration SetTrustedProxies dans le code
echo "15. Configuration SetTrustedProxies dans main.go..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
grep -A 20 "SetTrustedProxies" backend/cmd/server/main.go >> "$REPORT_FILE" 2>&1
echo "" >> "$REPORT_FILE"

# 16. Résumé des résultats
echo "16. RÉSUMÉ DES RÉSULTATS..." | tee -a "$REPORT_FILE"
echo "----------------------------------------" >> "$REPORT_FILE"
if echo "$RESPONSE_DIRECT" | grep -q "HTTP_CODE:200"; then
    echo "✅ Backend direct (3301): FONCTIONNE" >> "$REPORT_FILE"
else
    echo "❌ Backend direct (3301): ÉCHEC" >> "$REPORT_FILE"
fi

if echo "$RESPONSE_NGINX" | grep -q "HTTP_CODE:200"; then
    echo "✅ Via Nginx (3300): FONCTIONNE" >> "$REPORT_FILE"
else
    echo "❌ Via Nginx (3300): ÉCHEC (403 Forbidden)" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"
echo "=========================================" >> "$REPORT_FILE"
echo "Fin du rapport" >> "$REPORT_FILE"
echo "=========================================" >> "$REPORT_FILE"

echo ""
echo "✅ Diagnostic terminé!"
echo "📄 Rapport complet: $REPORT_FILE"
echo ""
echo "Affichage du résumé:"
tail -10 "$REPORT_FILE"

