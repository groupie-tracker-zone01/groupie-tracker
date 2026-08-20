# Groupie Tracker

Projet de groupe réalisé dans le cadre de Zone01 Normandie.

[![CI Go](https://github.com/groupie-tracker-zone01/groupie-tracker/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/groupie-tracker-zone01/groupie-tracker/actions/workflows/ci.yml)

## Suivi de l’équipe

- [Tableau des tickets](https://github.com/orgs/groupie-tracker-zone01/projects/1)
- [Plan visuel des branches et des merges](https://groupie-tracker-zone01.github.io/groupie-tracker/)
- [Network Graph Git réel](https://github.com/groupie-tracker-zone01/groupie-tracker/network)
- [Jira SCRUM](https://maugerlonguemareleandre.atlassian.net/issues/?jql=project%20%3D%20SCRUM%20ORDER%20BY%20key%20ASC)

## Lancer le projet

Le serveur utilise le port `8080` par défaut :

```sh
go run .
```

La page temporaire est ensuite accessible à l'adresse <http://localhost:8080>.

Pour utiliser un autre port :

```sh
PORT=9090 go run .
```

## Vérifier le projet

```sh
gofmt -w .
go test ./...
```

## Organisation

- Responsable d'équipe : Léandre Mauger-Longuemare
- Dépôt officiel de rendu : Zone01 (remote `origin`)
- Dépôt de collaboration : GitHub (remote `github`)
- Le travail est organisé avec les tickets GitHub.

## Workflow

1. Choisir ou se faire attribuer un ticket.
2. Créer une branche dédiée depuis `main`.
3. Développer uniquement le contenu du ticket.
4. Ouvrir une Pull Request vers `main`.
5. Faire relire la Pull Request avant fusion.
6. Supprimer la branche après fusion.

## Conventions de branches

- `feature/<numero>-<description>` pour une fonctionnalité
- `fix/<numero>-<description>` pour une correction
- `docs/<numero>-<description>` pour la documentation

## Règles d'équipe

- Ne pas développer directement sur `main`.
- Une Pull Request doit correspondre à un ticket.
- Garder les commits courts, explicites et centrés sur une seule modification.
- Mettre à jour sa branche avec `main` avant la fusion.
