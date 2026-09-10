# Ticket #11 — Audit des tests du projet

## Objectif

Le ticket #11 ne consiste pas seulement à conserver les tests créés au moment du MVP.

Le MVP #39 a été validé à une étape précise du projet. Le code a ensuite évolué, notamment avec la refonte front-end de la PR #42. Les tests du MVP ont alors été supprimés avec l'ancienne organisation du code.

L'objectif de #11 est donc de tester l'état actuel du projet avec trois niveaux complémentaires :

- tests unitaires ;
- tests d'intégration ;
- tests de non-régression.

Les tests automatisés ne dépendent pas de l'API Groupie Tracker distante.

## 1. Tests unitaires

Fichier principal : `server/api_test.go`.

Ils vérifient séparément des fonctions du package `server`, notamment :

- `fetchJSON` avec une réponse JSON valide ;
- `fetchJSON` avec une erreur HTTP ;
- `fetchJSON` avec un JSON invalide ;
- `sortDates` ;
- la construction des `ArtistFull` ;
- le tri des membres, lieux et dates ;
- `GetArtistFull` avec un identifiant existant ;
- `GetArtistFull` avec un identifiant inconnu.

Le client HTTP utilise `httptest.NewServer`. Les tests restent donc déterministes et ne dépendent pas du réseau externe.

## 2. Tests d'intégration

Deux niveaux sont utilisés.

### Routes et handlers avec données contrôlées

Fichier : `server/handler_test.go`.

Il vérifie ensemble le routeur, les handlers et les données contrôlées :

- `/` ;
- `/artists` ;
- `/api/artists` ;
- `/api/locations` ;
- `/api/dates` ;
- `/api/relations` ;
- `/api/search` ;
- erreurs HTTP 400, 404 et 405 ;
- recherche partielle ;
- priorité d'une correspondance exacte.

Des templates minimaux sont utilisés ici afin d'isoler la logique HTTP.

### Intégration avec les vrais fichiers de l'application

Fichier : `application_integration_test.go`.

Ce test charge les vrais templates présents dans `templates/` et passe par le vrai routeur `server.Routes`.

Il vérifie notamment :

- que la page d'accueil réelle peut être rendue ;
- que la recherche atteint réellement `templates/pages/artists.html` ;
- que la page 404 réelle est rendue ;
- que les ressources statiques principales sont servies : `home.css`, `home.js` et `artists.js`.

Cette couche est importante : un handler peut fonctionner avec un template de test alors que le vrai template du projet est devenu incompatible.

## 3. Tests de non-régression

Les tests de non-régression protègent des comportements déjà validés et susceptibles d'être cassés lors d'une modification ultérieure.

Ils couvrent notamment :

- le port par défaut et la variable `PORT` dans `main_test.go` ;
- la réponse 404 sur une route inconnue ;
- les méthodes HTTP refusées ;
- les recherches partielles ;
- la priorité d'une recherche exacte ;
- le rendu des vrais templates ;
- le service des ressources statiques ;
- la compatibilité entre les structures Go et les champs utilisés dans les templates.

## 4. Régression réelle détectée pendant #11

Le nouveau test d'intégration avec les vrais templates a détecté une incompatibilité qui n'était pas visible avec les templates minimaux de `server/handler_test.go`.

### Côté Go

Dans `server/api.go`, lignes 58 à 61 à l'état du code contrôlé, la structure est :

```go
type LastConcert struct {
    City  string
    Dates []string
}
```

Le champ disponible est donc `Dates` et non `LastDate`.

### Côté template

Dans `templates/pages/artists.html`, ligne 55 sur `main` avant la correction de la PR #43, le template utilisait :

```html
<span class="relations-dates-tag">{{.LastDate}}</span>
```

Lors du rendu réel, Go a renvoyé l'erreur :

```text
template: artists.html:55:64: executing "artists" at <.LastDate>: can't evaluate field LastDate in type server.LastConcert
```

Conséquence : le handler pouvait être correct, les tests HTTP avec templates factices pouvaient passer, mais le vrai rendu de la page artistes échouait.

### Correction

La PR #43 remplace l'utilisation de `.LastDate` par une boucle sur le champ réel `.Dates` :

```html
{{- range .Dates -}}
<span class="relations-dates-tag">{{.}}</span>
{{- end -}}
```

Cette correction remet le template en cohérence avec `server.LastConcert`.

## 5. Exemple d'un test incorrect, et non d'un bug du programme

Pendant la création de #11, un premier test de recherche partielle utilisait :

```text
/artists?q=queen
```

Le test attendait `Queen` et `Queens of the Stone Age`.

Cependant, le serveur donne volontairement la priorité à une correspondance exacte. Avec `queen`, `Queen` est donc la seule réponse attendue.

Le problème venait du scénario de test, pas du programme.

Le test partiel a été corrigé avec :

```text
/artists?q=que
```

Cette valeur n'est pas une correspondance exacte et permet de vérifier correctement le comportement de recherche partielle.

Le code produit n'a pas été modifié pour faire passer un test erroné.

## 6. Vérifications automatiques

La CI du projet exécute :

```text
gofmt -l .
go vet ./...
go test -race ./...
go build ./...
```

Elle exécute également les tests du script de synchronisation GitHub/Jira.

Le `-race` permet de détecter les accès concurrents dangereux pendant l'exécution des tests. Les tests de charge, de saturation, de concurrence importante, de timeout et de freeze restent volontairement dans le ticket #31.

## 7. Ordre de validation

Au moment de cet audit :

1. la PR #43 contient la correction du template et doit être relue puis fusionnée ;
2. la PR #44 du ticket #11 doit ensuite être vérifiée contre le `main` corrigé ;
3. la CI de #44 doit être entièrement verte ;
4. #44 doit être relue et fusionnée ;
5. le ticket #11 sera alors terminé ;
6. le ticket #31 pourra commencer.

## Conclusion

Les tests du MVP ont servi de point de départ, mais ils ne suffisaient plus pour garantir l'état final du projet.

Le ticket #11 remet en place une couverture adaptée au code actuel et ajoute une vérification avec les vrais templates et ressources du projet. Cette vérification a déjà détecté une régression réelle entre `server/api.go` et `templates/pages/artists.html`, ce qui confirme son utilité.