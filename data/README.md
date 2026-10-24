# Daten

| Datei | Inhalt |
|---|---|
| `routes-data.json.gz` | Datensatz des Planers (`foreverroutes.dataset.v1`), gebaut von `foreverroutes update` / `build-data` |
| `sources/lock.json` | festgeschriebene Stände: QuestieDB-, EverythingQuests- und cmangos-Commit, ForeverDB-Build und Prüfsumme |
| `sources/foreverdb-snapshot.json.gz` | ForeverDB-Questliste Level 1–30 mit Spielmetadaten (Data from ForeverDB.net) |
| `sources/warcraftdb-forever.json` | Questangaben von WarcraftDB Forever, die den anderen Quellen fehlen |
| `sources/wclbox-items.json` | woher ein Gegenstand kommt (Gegner und Gebiet), gelesen von den Gegenstandsseiten der WCLBox; die Seite nennt keine Dropchance |
| `sources/foreverchanges-quests.json` | Zusammenfassung je offener Quest von ForeverChanges: Questart (normal, Elite, Dungeon) und der Gegner aus dem Zielsatz |
| `sources/manual-facts.json` | einzeln belegte Werte (XP, Level, wiederholbar, Questziele mit Mengen) von WarcraftDB, OctoWow, ClassicDB, Wowhead Forever und Beobachtungen aus ForeverDB, jeweils mit Link; übernommen aus dem früheren Python-Werkzeug und dem früheren Addon ForeverQuests |
| `archive/dungeon-journal/` | Dungeon-Daten des früheren Addons ForeverQuests (Bosse, Beute, Dungeonquests) mit Quellenliste; vom Planer noch nicht gelesen |

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
- **WCLBox** (https://wowforever.wclbox.com, „新手盒子·魔兽世界无限数据库“) — Gegenstandsseiten mit „Source: Gegner, Gebiet“.
  robots.txt erlaubt die Seiten, die Seite lädt automatische Leser ausdrücklich ein (llms.txt) und bittet um Nennung mit
  Seitenlink; jeder Eintrag behält seinen Link. Abruf: eine Anfrage pro Sekunde, nur für Gegenstände ohne andere Quelle.
- **ForeverChanges** (https://foreverchanges.pro) — Questseiten (Daten aus dem Spielclient und aus Wowheads Forever-Datenbank):
  Questart und Zielsatz. robots.txt erlaubt die Questseiten; eine Anfrage pro Sekunde, nur für offene Quests.
- **Polestar forever-db** (https://github.com/fitzforge/polestar, Ordner `forever-db`, GPL-3.0) — aus dem Forever-Spielclient
  gebaute Questdaten: Forever-XP (Dungeonquests mit Ziel im Dungeon wurden auf etwa das 2- bis 2,7-Fache angehoben),
  Level, Volks- und Klassenmasken mit den Skyborne-Bits, Questgeber, Abgaben, Ziele mit Mengen, Voraussetzungen, was einen
  Gegenstand droppt, Spawnpunkte und Kartenrahmen. Wo Polestar eine Quest kennt, gelten seine Clientwerte vor allen
  Referenzen außer im Spiel beobachteten Werten; Quests, die nur Polestar kennt, kommen hinzu. Ereignisziele (Gebiet
  erkunden, Eskorte) haben keinen Ort in den Daten und werden ohne sie geplant. Die Abreise der Skyborne von Zephras Isle
  (Luftschiff nach Mulgore, Skycutter nach Alterac und Portal nach Sturmwind) stammt aus Polestars Skyborne-Guides, die
  Fahrzeiten sind geschätzt. Stand und Commit stehen in `data/sources/lock.json` (`polestar`).
- **cmangos classic-db** (https://github.com/cmangos/classic-db, GPL-3.0) — Classic-Weltdaten: Zielmengen,
  Beutechancen, Spawnpositionen, Respawnzeiten, Lebenspunkte und Schaden der Kreaturen, Flugmeister, Gasthäuser,
  Transporte. Classic, nicht Forever: Forever-Angaben haben Vorrang. Werte aus cmangos stehen unter GPL-3.0; damit
  steht auch `routes-data.json.gz` unter GPL-3.0 (ebenso die Werte aus Polestars forever-db).

**Dropchancen:** Wo eine Quelle nur den Gegner nennt, nicht die Chance, steht im Datensatz `unknown` (Chance 0, kein Wert).
Der Planer nimmt dafür einen ausdrücklich gekennzeichneten Planungswert an (30 %) und meldet im Ergebnis, wie viele Schritte
davon abhängen (`assumedDropSteps`). Echte Werte kommen aus dem Mitschreiben im Spiel.

Nichts wird erfunden. Was keine Quelle kennt, bleibt im Datensatz als unbekannt markiert (`unsupported` mit Grund).
