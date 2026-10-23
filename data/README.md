# Daten

| Datei | Inhalt | Herkunft |
|---|---|---|
| `routes-data.json.gz` | Datensatz des Planers (`foreverroutes.dataset.v1`) | gebaut mit `foreverroutes build-data` aus den Dateien unten und cmangos classic-db |
| `sources/merged-quests-1-30.json.gz` | zusammengeführte Quest-Datenbank Level 1–30 (`foreverroutes.unified_quests.v1`) | früheres ForeverRoutes-Werkzeug, Branch `artifacts`, Commit `7d66f93074d538a9f9e056c4ee712d176219d1f1`, unverändert |
| `sources/warcraftdb-forever.json` | Questangaben, die den anderen Quellen fehlen | `foreverroutes fetch-warcraftdb`, WarcraftDB Forever, Stand 23.10.2026 |

## Quellen und Hinweise

- **ForeverDB** — Data from [ForeverDB.net](https://foreverdb.net). Wo ForeverDB-Daten angezeigt werden, muss dieser
  Hinweis mit Link sichtbar sein. ForeverDB ist nicht mit Blizzard Entertainment verbunden.
- **QuestieDB Forever**, **EverythingQuests Forever** — Questgeber, Abgaben, Ziele und Positionen (über die
  zusammengeführte Datenbank, jede Angabe mit Quelle).
- **WarcraftDB Forever** (https://forever.warcraftdb.com) — Spielclient-Questdaten (Ziele mit Mengen, XP, Level,
  Positionen). Abruf laut robots.txt erlaubt, mit 1,5 s Abstand und Zwischenspeicher.
- **cmangos classic-db** (https://github.com/cmangos/classic-db, GPL-3.0) — Classic-Weltdaten: Zielmengen,
  Beutechancen, Spawnpositionen, Respawnzeiten, Lebenspunkte und Schaden der Kreaturen, Flugmeister, Gasthäuser,
  Transporte. Classic, nicht Forever: Forever-Angaben haben Vorrang. Werte aus cmangos stehen unter GPL-3.0; damit
  steht auch `routes-data.json.gz` unter GPL-3.0.

Nichts wird erfunden. Was keine Quelle kennt, bleibt im Datensatz als unbekannt markiert (`unsupported` mit Grund).
