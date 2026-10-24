# Daten

| Datei | Inhalt |
|---|---|
| `routes-data.json.gz` | Datensatz des Planers (`foreverroutes.dataset.v1`), gebaut von `foreverroutes update` / `build-data` |
| `sources/lock.json` | festgeschriebene Stände: QuestieDB-, EverythingQuests- und cmangos-Commit, ForeverDB-Build und Prüfsumme |
| `sources/foreverdb-snapshot.json.gz` | ForeverDB-Questliste Level 1–30 mit Spielmetadaten (Data from ForeverDB.net) |
| `sources/warcraftdb-forever.json` | Questangaben von WarcraftDB Forever, die den anderen Quellen fehlen |
| `sources/manual-facts.json` | einzeln belegte Werte (XP, Level, wiederholbar) von WarcraftDB, OctoWow, ClassicDB, Wowhead und Beobachtungen aus ForeverDB, jeweils mit Link; übernommen aus dem früheren Werkzeug |

QuestieDB, EverythingQuests und cmangos werden beim Bau in den festgeschriebenen Commits geladen und nicht eingecheckt.

## Quellen und Hinweise

- **ForeverDB** — Data from [ForeverDB.net](https://foreverdb.net). Wo ForeverDB-Daten angezeigt werden, muss dieser
  Hinweis mit Link sichtbar sein. ForeverDB ist nicht mit Blizzard Entertainment verbunden.
- **QuestieDB** (https://github.com/Questie/QuestieDB) — Forever-Quests, NPCs, Objekte, Gegenstände; Classic-XP und
  Beutechancen als Classic-Referenz gekennzeichnet.
- **EverythingQuests** (https://github.com/wheelbarrel00/EverythingQuests, MIT) — Forever-Questgeber, Abgaben, Ziele,
  Voraussetzungen (teils aus AllTheThings).
- **WarcraftDB Forever** (https://forever.warcraftdb.com) — Spielclient-Questdaten (Ziele mit Mengen, XP, Level,
  Positionen). Abruf laut robots.txt erlaubt, mit 1,5 s Abstand und Zwischenspeicher.
- **cmangos classic-db** (https://github.com/cmangos/classic-db, GPL-3.0) — Classic-Weltdaten: Zielmengen,
  Beutechancen, Spawnpositionen, Respawnzeiten, Lebenspunkte und Schaden der Kreaturen, Flugmeister, Gasthäuser,
  Transporte. Classic, nicht Forever: Forever-Angaben haben Vorrang. Werte aus cmangos stehen unter GPL-3.0; damit
  steht auch `routes-data.json.gz` unter GPL-3.0.

Nichts wird erfunden. Was keine Quelle kennt, bleibt im Datensatz als unbekannt markiert (`unsupported` mit Grund).
