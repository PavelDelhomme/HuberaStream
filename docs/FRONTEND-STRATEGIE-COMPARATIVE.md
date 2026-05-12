# Stratégie frontend StreamMake — comparaison pour un usage multimédia (films, lecteur, admin)

Objectif : **maintenabilité**, **simplicité d’exploitation**, et **surtout moins gourmand** que le combo actuel **Next.js en mode dev dans Docker**.  
Ce document compare les options **dans l’optique du projet** (catalogue, lecture, admin, auth, appels API).

> Mesures **réelles partielles** (machine de dev, une exécution `du`) : `frontend-nextjs/` → `node_modules` **~484 Mo**, `.next` **~124 Mo** après build partiel. Pour des temps de build et tailles `dist` à jour, lance sur ta machine :  
> `bash scripts/benchmark-frontends.sh`

---

## 1. Où part la « gourmandise » aujourd’hui ?

| Source | Impact |
|--------|--------|
| **`next dev` dans Docker** | Très lourd (V8 + compilateur SWC + HMR + caches). Souvent **0,5–1,5 Go+** de RAM pour le seul process. |
| **Volume `node_modules` anonyme** | Premier démarrage = `npm ci` long et gros pic RAM disque + mémoire. |
| **`react-player` + dépendances** | Gros morceau du bundle client ; à charger sur les pages lecteur. |
| **Next en prod (`next start`)** | Beaucoup plus raisonnable que `dev` ; **`output: 'standalone'`** déjà prévu dans `next.config.js`. |

**Conclusion** : le plus gros levier n’est pas « remplacer React », c’est **ne pas servir l’app utilisateur en `next dev` en production** et **réduire le JS du lecteur** (lazy import, routes dédiées).

---

## 2. Tableau comparatif (orienté StreamMake)

Légende **RAM prod** : process serveur ou équivalent derrière nginx (ordre de grandeur, charge faible).  
**Effort** : migration depuis l’existant Next (pages, auth, admin).  
**Multimédia** : intégration lecteur HLS/DASH/progressif, gros fichiers, UX type « Netflix ».

| Option | Modèle | RAM **dev** | RAM **prod** (ordre de grandeur) | Bundle client | Effort migration | Multimédia | Maintenabilité |
|--------|--------|-------------|-----------------------------------|----------------|------------------|------------|----------------|
| **Next.js 14 (actuel)** | App Router + client | Très élevé | Moyen | Moyen–gros | — (référence) | Très bon (SSR optionnel, images) | Une stack, doc riche |
| **Next `start` + standalone** | Même code, autre runtime | — | **Bien plus bas** que `dev` | Idem | Faible | Idem | **Recommandé en priorité** |
| **Vite + React SPA** (`frontend/` legacy) | SPA classique | **Plus bas** que Next dev | **Bas** (fichiers statiques + nginx) | Souvent **plus léger** que Next (moins de runtime) | **Moyen–élevé** (réécrire routes, SSR si besoin) | Bon (Video.js déjà là) | Deux apps si on garde Next aussi |
| **Vite + React (nouvelle app)** | Remplacer Next | Plus bas | Bas | Maîtrisable | Élevé (réimplémenter tout le Next) | Bon | Une stack si Next supprimé |
| **Remix** | SSR data + forms | Moyen–élevé | Moyen | Moyen | Élevé | Très bon | Courbe Remix |
| **SvelteKit** | SSR + islands | Moyen | **Bas–moyen** | **Souvent plus léger** (moins de KB) | Très élevé (réécriture) | Bon | Moins d’écosystème React |
| **Astro** | Pages statiques + îlots React | Bas en statique | **Très bas** si peu d’îlots | Très bas sur landing | Élevé pour **app type dashboard** | Moyen (îlots pour lecteur) | Excellent site vitrine, **moins** pour app dense |
| **Nuxt 3** | Vue SSR | Moyen | Moyen | Moyen | Très élevé (changement de langage) | Bon | Si l’équipe préfère Vue |
| **Qwik** | Resumable / lazy extrême | Variable | Potentiellement bas TTI | Peut être très bas | Très élevé | À valider lecteur | Outils plus jeunes |

---

## 3. Gains **réels** attendus (sans magie)

| Changement | Gain mémoire / perf | Coût |
|------------|---------------------|------|
| **Prod : `next build` + `next start` + image standalone**, pas de bind-mount du repo | **Très fort** (le plus rentable) | Dockerfile prod + CI |
| **Dev local** : `npm run dev` sur l’hôte, pas dans Docker | **Fort** en RAM Docker | Discipline équipe |
| **`dynamic(() => import('react-player'))`** sur la page lecteur | **Moyen** (moins de JS sur accueil / catalogue) | Quelques lignes |
| **Remplacer Next par Vite SPA** (même React) | **Moyen** en prod, **fort** en dev Docker | **Semaines** de portage + tests |
| **SvelteKit / Astro / autre** | Variable ; parfois **-20–40 %** bundle vs React | **Mois** de refonte + risque régressions lecteur |

---

## 4. « Tester réellement tous les frontends » — ce qui est raisonnable

Mettre **7 stacks** en concurrence dans le repo **n’est pas maintenable**. La pratique utile :

1. **Garder une seule cible produit** (aujourd’hui Next).
2. **Mesurer** avec un script (temps de build, taille `dist` / `.next`, `du node_modules`) — voir `scripts/benchmark-frontends.sh`.
3. Si un jour tu veux challenger Vite : faire un **POC sur une branche** (login + liste vidéos + lecteur) pendant 2–3 jours, puis comparer **Lighthouse + RAM** sur la même machine.

Les chiffres « web officiels » des frameworks varient selon les apps : pour StreamMake, ce qui domine reste **react-player / vidéo / nombre de pages client**.

---

## 5. Recommandation pour StreamMake (web + multimédia)

1. **Court terme (maximum gain / effort)**  
   - Poursuivre **Next.js** pour l’app complète.  
   - **Production** : build `standalone`, **pas** de `next dev` derrière nginx.  
   - **Lazy-load** du lecteur et des pages admin lourdes.

2. **Moyen terme (si la RAM Docker dev reste insupportable)**  
   - Développeurs lancent Next **en local** ; Docker ne monte que **backend + nginx + API** (ou nginx sert une preview `next export` / build statique si applicable).  

3. **Long terme (seulement si décision produit)**  
   - **Unifier** sur **Vite + React** en repartant du dossier `frontend/` comme base technique (déjà Video.js), et **retirer** Next pour n’avoir **qu’une** stack — gros chantier, gain surtout sur **simplicité d’infra** et **footprint dev**, pas miracle sur la lecture vidéo.

4. **À ne pas faire « pour la perf » sans besoin clair**  
   - Deux frontends (Next + Vite) en parallèle sur le long terme : double CI, double auth, dette.

---

## 6. Synthèse une phrase

Pour une app **streaming + admin**, le **vrai saut** de consommation et de facilité d’exploitation vient du **passage dev → prod correct** et du **découpage JS du lecteur**, pas du remplacement de Next par la mode framework du moment.
