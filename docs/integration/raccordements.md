# Feuille des raccordements

## Objectif

Cette documentation décrit les raccordements entre les briques du projet Groupie Tracker.

Un ticket peut fonctionner isolément sans être réellement intégré à l'application. Un checkpoint d'intégration doit donc vérifier le parcours complet jusqu'au résultat visible par l'utilisateur.

## Raccordement validé par le ticket #39

| Brique | Rôle | Raccordement attendu |
|---|---|---|
| Serveur Go | Expose l'application HTTP | Utilise le routeur principal |
| Client API | Charge les données Groupie Tracker | Alimente `AppData` |
| `AppData` | Regroupe les données | Est transmis aux handlers |
| Handler `/` | Gère l'accueil | Transmet `AppData` au template |
| `home.html` | Génère l'accueil | Parcourt `.Artists` |
| `/static/` | Sert CSS et JS | Rend les ressources accessibles |
| Navigateur | Affiche le résultat | Reçoit les artistes réels |

## Chaîne fonctionnelle

```text
API Groupie Tracker
        |
        v
    LoadData()
        |
        v
     AppData
        |
        v
    routes(data)
        |
        v
  homeHandler(data)
        |
        v
    home.html
        |
        v
  liste des artistes
        |
        v
     navigateur
```

## Problème détecté

Avant le checkpoint, les routes API répondaient correctement et contenaient les donrées réelles, mais `/` affichait encore la page temporaire du serveur.

Le client API et le frontend existaient donc, mais leur raccordement n'était pas terminé :

```text
API -> AppData -> routes API       OK
API -> AppData -X-> page d'accueil
```

## Correction minimale

Le ticket #39 a uniquement ajouté ce qui était nécessaire au MVP :

- transmission de `AppData` au handler d'accueil ;
- exécution des templates existants ;
- transmission des données au template ;
- affichage minimal de `.Artists` avec image et nom ;
- exposition des ressources existantes via `/static/`.

Aucun redesign ou refactor général n'a été ajouté.

## Validation

Les contrôles suivants passent :

```text
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

La validation manuelle a confirmé le démarrage sur le port 8080, l'affichage réel des artistes sur la page d'accueil, le chargement des ressources statiques et le fonctionnement du rafraîchissement navigateur.

## Règle pour les prochains tickets

Deux briques qui fonctionnent séparément ne sont pas nécessairement intégrées.

```text
brique A fonctionne
+
brique B fonctionne
```

doit être complété par :

```text
brique A -> brique B -> résultat utilisateur
```

Les futurs checkpoints doivent tester cette chaîne complète.
