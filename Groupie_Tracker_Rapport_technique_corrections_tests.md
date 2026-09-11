<!--
Rapport technique Groupie Tracker — version Markdown issue du document DOCX.
Document distinct du README et de la documentation générale du projet.
-->

**GROUPIE TRACKER**

**Rapport technique de corrections, modifications et validation des tests**

*Document distinct du README et de la documentation fonctionnelle du projet*

Version de travail - état au 10 septembre 2026

Sources principales : historique GitHub, Pull Requests, diffs et résultats CI


# 1. Objet du rapport

Ce document explique les corrections, modifications et ajouts techniques réalisés sur Groupie Tracker au cours de la phase de stabilisation. Il ne remplace ni le README ni la documentation générale du projet. Son objectif est de rendre la démarche d'ingénierie vérifiable : pour chaque problème important, le rapport présente le symptôme observé, la cause identifiée, un extrait avant/après, la correction appliquée et les tests utilisés pour prouver la non-régression.

> Principe de preuve - Une correction n'est considérée comme validée que lorsqu'elle est couverte par un test pertinent et que la CI du dépôt passe sur l'état réellement proposé à la fusion.


# 2. Périmètre et état des travaux

| Element | Objectif | Statut | Preuve |
| --- | --- | --- | --- |
| PR #42 | Refonte front / architecture ayant notamment supprimé les anciens tests Go | MERGEE | Merge 7d4374e |
| PR #43 | Recherche serveur, résultats multiples, compatibilité template/données | MERGEE | Merge 2d5f364 |
| PR #44 / #11 | Restauration et extension des tests essentiels | MERGEE / FERME | Merge 80bd20f |
| PR #46 / #31 | Robustesse, concurrence, timeouts, rendu 500 | MERGEE / FERME | Merge ec69287 |
| PR #47 / #12 | Recette UI, accessibilité et navigation | OUVERTE - CI VERTE | En attente de validation humaine au moment de rédaction |


# 3. Méthode d'analyse

- Comparer les diffs des Pull Requests pour identifier les modifications effectives.

- Reproduire les comportements avec des données contrôlées et des serveurs httptest locaux.

- Séparer les bugs du produit des erreurs de scénario de test.

- Utiliser des tests unitaires pour les fonctions isolables, des tests d'intégration pour les routes et vrais templates, puis des tests de robustesse pour les erreurs, timeouts et accès concurrents.

- Faire exécuter automatiquement gofmt, go vet, go test -race ./... et go build ./... par la CI avant fusion.

# 4. Point de départ : disparition des tests lors de la refonte #42

La PR #42 a effectué une refonte importante du front-end et de l'organisation du code. Dans ce mouvement, le fichier main_test.go du checkpoint précédent a été supprimé. Le fait que le MVP ait été vert auparavant ne garantissait donc plus le comportement du code final : les tests historiques ne s'exécutaient plus contre la nouvelle architecture.

**Avant - tests présents**

```
// Avant la PR #42 : main_test.go contenait notamment
func TestHome(t *testing.T) { ... }
func TestNotFound(t *testing.T) { ... }
func TestServerPort(t *testing.T) { ... }
```


**Après la PR #42 - fichier supprimé**

```
// Diff de la PR #42
main_test.go
@@ -1,75 +0,0 @@
- package main
- ...
- func TestHome(...)
- func TestNotFound(...)
- func TestServerPort(...)
```


> Pourquoi c'est important - Une ancienne CI verte prouve uniquement l'état du code testé à ce moment-là. Après une refonte qui déplace les handlers, les structures et les templates, la couverture doit être restaurée sur l'architecture active.


# 5. PR #43 - Recherche serveur et compatibilité des résultats

Après la refonte du front, la recherche devait rester simple et pilotée par le serveur Go. La PR #43 a conservé une seule barre de recherche, supprimé l'ancien mécanisme de suggestions natif et permis l'affichage de plusieurs correspondances pour une saisie partielle, sans ajouter de JavaScript de recherche.

## 5.1 Formulaire de recherche

**Avant**

```
<input type="text" name="q" class="search" ...>
<div id="suggestions-box"></div>
<button type="submit" class="search-btn">Search</button>
```


**Après**

```
<input type="search" name="q" class="search"
       placeholder="Type an artist..." maxlength="30"
       autocomplete="off" value="{{.Query}}" required>
<button type="submit" class="search-btn"
        aria-label="Search artists">Search</button>
```


Le formulaire utilise GET /artists?q=... : la logique de recherche reste côté serveur. L'attribut required empêche une soumission vide depuis l'interface et le type search rend l'intention explicite.

## 5.2 Bug réel : .LastDate contre .Dates

Le test d'intégration utilisant les vrais templates a révélé une incompatibilité entre le modèle Go actif et le template artistes. La structure server.LastConcert expose un tableau Dates []string, alors que le template appelait encore un ancien champ .LastDate.

**Structure Go active**

```
type LastConcert struct {
    City  string
    Dates []string
}
```


**Avant - template incompatible**

```
{{range .LastConcerts}}
    <strong class="relations-city">{{.City}}:</strong>
    <span class="relations-dates-tag">{{.LastDate}}</span>
{{end}}
```


**Après - template aligné sur la structure Go**

```
{{range .LastConcerts}}
    <strong class="relations-city">{{.City}}:</strong>
    {{range .Dates}}
        <span class="relations-dates-tag">{{.}}</span>
    {{end}}
{{end}}
```


```
Pourquoi ça cassait - Le moteur html/template résout les champs au moment de l'exécution. .LastDate n'existant pas sur server.LastConcert, le rendu du template échouait. Le bug n'était donc ni visuel ni aléatoire : c'était une incompatibilité de type entre le template et les données réellement fournies.
```


**Preuve CI - run #72 du 10/09/2026**

```
template: artists.html:55:64: executing "artists" at <.LastDate>:
can't evaluate field LastDate in type server.LastConcert

--- FAIL: TestApplicationUsesRealTemplates
application_integration_test.go:86:
artists page does not contain "Queens of the Stone Age"
```


# 6. PR #44 / ticket #11 - Restauration et extension des tests essentiels

La PR #44 a recréé une base de tests adaptée au package server et à la nouvelle architecture. Elle ajoute quatre fichiers de tests pour couvrir plusieurs niveaux : fonctions isolées, handlers, routes, vrais templates et ressources statiques. Les tests réseau utilisent httptest.NewServer afin de rester déterministes et indépendants de l'API distante.

## 6.1 Tests unitaires

- fetchJSON : décodage JSON valide.

- fetchJSON : réponse HTTP non-200.

- fetchJSON : JSON invalide.

- sortDates : ordre attendu et absence de modification de l'entrée.

- GetFullArtists : assemblage et tri des données.

- GetArtistFull : artiste existant et identifiant absent.

- serverPort : port par défaut et variable d'environnement PORT.

**Exemple - erreur HTTP contrôlée**

```
func TestFetchJSONStatusError(t *testing.T) {
    testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "upstream unavailable", http.StatusBadGateway)
    }))
    defer testServer.Close()

    var target any
    err := fetchJSON(testServer.URL, &target)
    if err == nil {
        t.Fatal("fetchJSON should return an error for a non-200 response")
    }
}
```


## 6.2 Tests d'intégration

- Route d'accueil avec les vrais templates du dépôt.

- Recherche partielle et exacte via /artists?q=....

- Codes 400, 404 et 405 visibles dans les pages d'erreur.

- Routes API et endpoint /api/search.

- Chargement des ressources statiques principales.

- Compatibilité routeur -> handler -> structure Go -> template réel.

**Exemple - intégration réelle de la recherche partielle**

```
res := httptest.NewRecorder()
handler.ServeHTTP(res, httptest.NewRequest(
    http.MethodGet,
    "/artists?q=Que",
    nil,
))

for _, expected := range []string{
    "Queen",
    "Queens of the Stone Age",
    "/static/JS/artists.js",
} {
    if !strings.Contains(res.Body.String(), expected) {
        t.Fatalf("artists page does not contain %q", expected)
    }
}
```


## 6.3 Un test peut être faux : le cas "queen"

Une première attente de test utilisait une requête équivalente à "queen" tout en attendant Queen et Queens of the Stone Age. Or le comportement fonctionnel valide donne la priorité à une correspondance exacte : si Queen existe, la recherche exacte doit retourner Queen uniquement. Le test a donc été corrigé vers une saisie réellement partielle comme "que". Aucun code produit n'a été modifié pour satisfaire une attente incorrecte.

> Leçon - Un test qui échoue n'implique pas automatiquement un bug produit. Il faut d'abord vérifier que l'oracle du test (le résultat attendu) correspond bien à la règle fonctionnelle.


# 7. PR #46 / ticket #31 - Robustesse, concurrence et stabilité

Les tests essentiels validaient les fonctionnalités, mais ils ne couvraient pas encore les blocages réseau, les annulations, les erreurs répétées, le rendu 500 après une erreur de template ni les accès concurrents. Le ticket #31 a volontairement traité cette couche séparément.

## 7.1 Timeout du client API

**Avant - client HTTP sans timeout explicite**

```
func fetchJSON(url string, target any) error {
    response, err := http.Get(url)
    ...
}
```


**Après - timeout et contexte**

```
const apiRequestTimeout = 5 * time.Second

var apiHTTPClient = &http.Client{
    Timeout: apiRequestTimeout,
}

func fetchJSONContext(ctx context.Context, client *http.Client, url string, target any) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    ...
    response, err := client.Do(req)
    ...
}
```


Le contexte permet d'interrompre explicitement une requête. Le timeout borne la durée maximale du client même si le serveur distant répond très lentement ou reste bloqué.

## 7.2 Timeouts du serveur HTTP

**Avant**

```
http.ListenAndServe(":"+port, router)
```


**Après**

```
return &http.Server{
    Addr:              ":" + port,
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```


Ces limites réduisent le risque qu'une connexion lente ou incomplète monopolise indéfiniment les ressources du serveur.

## 7.3 Rendu 500 fiable

**Avant - écriture directe dans ResponseWriter**

```
w.Header().Set("Content-Type", "text/html; charset=utf-8")
err := templates.ExecuteTemplate(w, "artists", artistData)
if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
}
```


**Après - rendu en mémoire puis envoi**

```
func renderPage(w http.ResponseWriter, templates *template.Template, templateName string, data any) {
    var buffer bytes.Buffer
    if err := templates.ExecuteTemplate(&buffer, templateName, data); err != nil {
        renderErrors(w, http.StatusInternalServerError, templates)
        return
    }

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write(buffer.Bytes())
}
```


> Pourquoi le buffer est nécessaire - Une réponse HTTP peut être considérée comme commencée dès les premiers octets écrits. Si le template échoue après avoir déjà écrit une partie du HTML, il est trop tard pour garantir proprement un statut 500 et une page d'erreur cohérente. Le buffer évite de valider la réponse avant la réussite complète du rendu.


## 7.4 Tests de robustesse ajoutés

| Scénario | Mécanisme | Ce que le test prouve |
| --- | --- | --- |
| API lente | Client local avec timeout court | La requête se termine dans une borne mesurable |
| Annulation | context.WithCancel + serveur httptest bloqué | context.Canceled remonte sans blocage |
| Erreurs 503 répétées | 5 échecs puis succès | Le client récupère après la panne |
| Erreur template | Template volontairement cassé | HTTP 500 + page 500, sans contenu partiel |
| Concurrence | 24 workers x 20 requêtes | 480 requêtes contrôlées sans freeze |
| Goroutines | Mesure avant/après | Pas de croissance anormale évidente |
| Client annulé | Contexte déjà annulé | Le handler ne reste pas bloqué |
| Benchmark | testing.B + RunParallel | Charge légère reproductible avec stdlib |


# 8. PR #47 / ticket #12 - Recette UI et accessibilité

La recette finale a identifié des défauts qui n'étaient pas des bugs métier mais pouvaient casser la structure HTML ou la navigation : document HTML imbriqué dans le header, lien /artists sans requête menant à une 400, absence de lien d'évitement et contrôles du carrousel insuffisants pour le clavier et la réduction des animations.

> Statut au moment de rédaction - La PR #47 est ouverte, la CI est verte et la revue a été demandée à Dev789. Les changements ci-dessous sont donc validés automatiquement mais pas encore fusionnés au moment où cette version du rapport est générée.


## 8.1 Structure HTML du header

**Avant - composant contenant un second document HTML**

```
{{define "header"}}
<!DOCTYPE html>
<html lang="en">
<head>...</head>
<body>
    <header>
        <nav>...</nav>
    </header>
</div>
{{end}}
</body>
</html>
```


**Après - composant réutilisable valide**

```
{{define "header"}}
<header>
    <nav aria-label="Main navigation">
        <a href="/" class="title2">Home</a>
        <a href="/" class="title3" aria-label="MetaRock home">MetaRock</a>
        <a href="/#artist-search" class="login">Search</a>
    </nav>
</header>
{{end}}
```


Le header devient un composant HTML partiel au lieu d'insérer un deuxième DOCTYPE, html, head et body dans les pages home/artists. La navigation centrale ne pointe plus vers /artists sans paramètre, route qui renvoie volontairement 400 lorsque la recherche est vide.

## 8.2 Recherche et images accessibles

**Avant**

```
<form action="/artists" method="GET" class="search-form-overlay">
    <section class="search-bar">
        <input type="search" name="q" class="search" ...>
        <button type="submit" class="search-btn">Search</button>
    </section>
</form>
```


**Après**

```
<form action="/artists" method="GET"
      class="search-form-overlay" id="artist-search">
    <div class="search-bar">
        <label class="sr-only" for="artist-query">Artist name</label>
        <input id="artist-query" type="search" name="q" class="search" ...>
        <button type="submit" class="search-btn">Search</button>
    </div>
</form>
```


Le champ dispose d'un label programmatique lié par for/id. Les images utiles reçoivent une description concise (par exemple 'Portrait of Queen'), tandis que les images purement décoratives du carrousel utilisent alt="" pour éviter du bruit dans les lecteurs d'écran.

## 8.3 Carrousel : clavier, pause et mouvements réduits

**Avant - autoplay essentiellement orienté souris**

```
function startAutoPlay() {
    autoPlayInterval = setInterval(nextSlide, 3000);
}

carousel.addEventListener('mouseenter', stopAutoPlay);
carousel.addEventListener('mouseleave', startAutoPlay);
```


**Après - clavier, focus et préférence utilisateur**

```
const reducedMotion =
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;

carousel.addEventListener('keydown', function(event) {
    if (event.key === 'ArrowRight') nextSlide();
    else if (event.key === 'ArrowLeft') prevSlide();
});

carousel.addEventListener('focusin', stopAutoPlay);
carousel.addEventListener('focusout', startAutoPlay);

if (reducedMotion) {
    setPausedByUser(true);
} else {
    startAutoPlay();
}
```


## 8.4 Tests de recette accessibilité

- Exactement un DOCTYPE et un body rendus sur la page d'accueil.

- Présence du skip link, de main, de la navigation nommée et du label de recherche.

- Présence des contrôles accessibles du carrousel.

- Structure des titres sur les résultats artistes.

- Lien de récupération après une recherche sans résultat.

- Attribut alt présent sur toutes les images rendues.

- Cibles de navigation principales répondant en HTTP 200.

- Feuille CSS d'accessibilité réellement servie par le routeur.

**Exemple - test de structure réelle**

```
if got := strings.Count(body, "<!DOCTYPE html>"); got != 1 {
    t.Fatalf("home rendered %d document declarations, want exactly 1", got)
}
if got := strings.Count(body, "<body>"); got != 1 {
    t.Fatalf("home rendered %d body elements, want exactly 1", got)
}
```


# 9. Pourquoi la suite de tests est considérée complète pour le périmètre du projet

Le mot "complet" doit être compris relativement aux exigences et risques identifiés du projet, pas comme une preuve mathématique qu'aucun bug ne pourra jamais exister. La couverture est complète pour les tickets #11, #31 et #12 car chaque famille de risque attendue dispose d'au moins un test adapté.

| Risque | Niveau de test | Couverture |
| --- | --- | --- |
| Fonctions de transformation | Tests unitaires | Tri, assemblage, identifiants absents |
| Client HTTP | Tests unitaires + httptest | 200, non-200, JSON invalide, timeout, annulation |
| Routes | Tests d'intégration | Accueil, artistes, API, 400/404/405 |
| Templates réels | Tests d'intégration | Compatibilité données -> HTML, régression .LastDate/.Dates |
| Ressources statiques | Tests d'intégration | CSS/JS servis et non vides |
| Recherche | Tests d'intégration | Partielle, exacte, résultat absent |
| Erreurs internes | Robustesse | 500 propre même après erreur de template |
| Concurrence | Robustesse + race detector | Rafale contrôlée, absence de data race détectée |
| Stabilité réseau | Robustesse | Timeout, contexte annulé, erreurs répétées |
| Accessibilité structurelle | Recette automatisée | Landmarks, labels, alt, navigation, structure HTML |


## 9.1 Complémentarité des tests

- Un test unitaire localise rapidement une erreur de fonction, mais ne prouve pas qu'un template réel utilise le bon champ.

- Un test d'intégration peut détecter une rupture entre packages, routes et templates, mais ne suffit pas à prouver le comportement sous concurrence.

- Le race detector recherche les accès mémoire concurrents problématiques, mais ne remplace pas les scénarios de timeout ou de blocage.

- Les tests d'accessibilité vérifient les marqueurs structuraux mesurables ; ils ne prétendent pas remplacer une inspection visuelle humaine sur tous les navigateurs.

> Exemple concret - Le bug .LastDate/.Dates n'aurait pas été trouvé par un simple test de tri des données. Il a été détecté lorsque la route a réellement exécuté le vrai template avec les structures Go actives. C'est précisément l'intérêt de combiner plusieurs niveaux de test.


# 10. Validation continue (CI)

Les Pull Requests sont soumises à une chaîne de contrôle automatisée avant fusion. Les étapes pertinentes sont :

- Tests de synchronisation GitHub/Jira sous Node.js.

- gofmt -l . : aucun fichier Go non formaté.

- go vet ./... : analyse statique standard Go.

- go test -race ./... : exécution de la suite Go avec détecteur de data races.

- go build ./... : compilation complète.

Un exemple important est la CI #72 de la PR #44 : gofmt et go vet passaient, mais go test -race a arrêté la chaîne sur l'incompatibilité .LastDate/.Dates. Après correction via #43 et resynchronisation de #44, la CI combinée est redevenue verte. Cela démontre que la CI n'a pas été utilisée comme simple formalité : elle a réellement empêché une régression de rejoindre main.

# 11. Limites et ce que les tests ne prétendent pas prouver

- Pas de benchmark industriel ni de test avec des milliers d'utilisateurs réels.

- Pas d'infrastructure cloud, Kubernetes ou monitoring payant.

- Pas de garantie absolue sur tous les navigateurs, lecteurs d'écran et tailles d'écran possibles.

- Pas de dépendance aux réponses de l'API Groupie Tracker distante dans les tests unitaires : l'isolation est volontaire.

- La PR #47 doit encore être approuvée et fusionnée pour que sa recette fasse partie de main dans l'état décrit ici.

# 12. Conclusion

La phase de stabilisation a transformé une validation essentiellement fonctionnelle en une validation multi-niveaux. Les changements ont corrigé une incompatibilité réelle de template, restauré les tests supprimés pendant la refonte, séparé un faux positif de test d'un vrai bug produit, ajouté des protections réseau, fiabilisé les erreurs 500, vérifié la concurrence et ajouté une recette structurelle d'accessibilité. La démarche privilégie des corrections ciblées, mesurables et vérifiables plutôt qu'une refonte générale.


# Annexe A - Références techniques

**PR #42 - Front end : **https://github.com/groupie-tracker-zone01/groupie-tracker/pull/42

**PR #43 - recherche serveur et résultats multiples : **https://github.com/groupie-tracker-zone01/groupie-tracker/pull/43

**PR #44 - tests essentiels #11 : **https://github.com/groupie-tracker-zone01/groupie-tracker/pull/44

**PR #46 - robustesse #31 : **https://github.com/groupie-tracker-zone01/groupie-tracker/pull/46

**PR #47 - recette UI/accessibilité #12 : **https://github.com/groupie-tracker-zone01/groupie-tracker/pull/47

**Ticket #11 : **https://github.com/groupie-tracker-zone01/groupie-tracker/issues/11

**Ticket #31 : **https://github.com/groupie-tracker-zone01/groupie-tracker/issues/31

**Ticket #12 : **https://github.com/groupie-tracker-zone01/groupie-tracker/issues/12

# Annexe B - Chronologie courte

| Date | Evénement |
| --- | --- |
| 09/09/2026 | PR #42 fusionnée : refonte front/architecture ; les anciens tests Go sont absents après cette fusion. |
| 10/09/2026 | PR #43 fusionnée : recherche serveur simplifiée, résultats multiples, correction .LastDate/.Dates. |
| 10/09/2026 | PR #44 fusionnée : tests essentiels restaurés et complétés ; ticket #11 fermé. |
| 10/09/2026 | PR #46 fusionnée : robustesse, timeouts, concurrence et rendu 500 ; ticket #31 fermé. |
| 10/09/2026 | PR #47 ouverte : recette UI/accessibilité ; CI verte, revue humaine en attente au moment de rédaction. |
