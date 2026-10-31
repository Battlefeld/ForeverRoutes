# Lizenzen und Quellen

ForeverRoutes besteht aus eigenem Code und aus Daten, die aus fremden Quellen abgeleitet sind. Jeder Teil behält die
Lizenz seiner Quelle. Die festgeschriebenen Stände (Commits, Builds) stehen in `data/sources/lock.json`, Herkunft und
Revision jeder Quelle zusätzlich im Datensatz selbst (`sources`).

## Was unter welcher Lizenz steht

| Teil | Lizenz | Lizenztext |
|---|---|---|
| Eigener Code: Planer (`route/`, `dataset/`, `cmd/`, `internal/`), Werkzeuge (`tools/`), Ingame-Addon (`ForeverRoutes/`, ohne `Data/Routes.lua`), Grafiken | MIT, © 2026 Battlefeld | [`LICENSE`](LICENSE) |
| Datensatz `data/routes-data.json.gz`, Laufgitter `data/walks.json.gz`, Wegtabelle `data/walktable.json.gz`, Routen `data/routes/`, Profile `data/profiles/`, Prüfberichte `data/audits/`, das exportierte `Routes.lua` | GPL-3.0 (abgeleitet aus GPL-3.0-Quellen, siehe unten) | [`LICENSES/GPL-3.0.txt`](LICENSES/GPL-3.0.txt) |
| Normale RestedXP-Kapitel `data/fallbacks/rxp.json.gz` und die daraus übernommenen Routenabschnitte | CC BY-NC-SA 4.0 (keine kommerzielle Nutzung, Weitergabe unter gleichen Bedingungen) | [`data/fallbacks/RXP-LICENSE.txt`](data/fallbacks/RXP-LICENSE.txt) |

Wer nur den Code weiterverwendet (ohne die Daten), braucht nur die MIT-Lizenz.

## Quellen

| Quelle | Was daraus stammt | Lizenz |
|---|---|---|
| [QuestieDB](https://github.com/Questie/QuestieDB) | Quests, NPCs, Objekte, Spawns, Spieler-Traces | GPL-3.0 nach Angabe des Questie-Teams (siehe Hinweis unten) |
| [cmangos classic-db](https://github.com/cmangos/classic-db) | Weltdatenbank: Gegner, Beute, Quest-Verknüpfungen, Dungeon-Trash | GPL-3.0 |
| [Polestar](https://github.com/fitzforge/polestar) forever-db und Forever-Guides | Forever-Quests, Ziele, NPCs, Spawns, Skyborne, Reisezeiten, Guide-Abläufe | GPL-3.0 |
| [NorgannonDB](https://github.com/zmarotrix/NorgannonDB) | Dungeons, Bosse, Beute (Instanzen aus DungeonJournal by Exehn) | GPL-3.0 |
| [Shortest Path Forever](https://github.com/cjber/shortest-path-forever) | Navigationskarten, mit denen Laufgitter und Wegtabelle gemessen sind; Flug- und Schiffszeiten | GPL-3.0-or-later |
| [RestedXP RXPGuides](https://github.com/RestedXP/RXPGuides) | Normale Kapitel als Rückfall-Route, Guide-Fakten | CC BY-NC-SA 4.0 |
| [EverythingQuests](https://github.com/wheelbarrel00/EverythingQuests) | Forever-Quests | MIT |
| [AEGIS: Pathfinder (Forever)](https://github.com/Torchlite-bit/Aegis_Pathfinder_Forever) | Zielorte, Geber- und Abgabeorte, fehlende XP | MIT |
| [EG Link output](https://github.com/JIVESCORP/eg-link-output-wowf) | Gemessene Dropchancen und Zielorte. „Data from Efficient Games (https://efficient.games), licensed under CC BY 4.0.“ Die Werte sind zusammengefasst und gefiltert. | CC BY 4.0 |
| [ForeverDB.net](https://foreverdb.net) | Questliste mit Spielmetadaten. Data from ForeverDB.net. | keine Lizenz genannt; Nutzung mit Quellenangabe |
| [WarcraftDB Forever](https://forever.warcraftdb.com), [WCLBox](https://wowforever.wclbox.com), [ForeverChanges](https://foreverchanges.pro) | Einzelne Spielangaben (Gegenstandsquellen, Questart, XP), jeweils mit Link | keine Lizenz genannt; einzelne Fakten mit Quellenangabe |
| WarcraftDB, OctoWow, ClassicDB, Wowhead Forever (`data/sources/manual-facts.json`, `quest-fixes.json`) | Einzeln belegte Werte, jeder mit Link | einzelne Fakten mit Quellenangabe |

**Hinweis zu QuestieDB:** QuestieDB hatte lange keine Lizenz. Das Questie-Team erlaubt das Weiterverwenden unter
GPL-3.0, kann das aber nicht für jeden früheren Beitragenden zusagen (siehe deren `LICENSE.md`). ForeverRoutes nutzt die
Daten unter GPL-3.0 und nennt die Quelle. Name und Logo von Questie werden nicht verwendet.

## Spielinhalte

World of Warcraft, die Namen, Texte und Daten des Spiels gehören Blizzard Entertainment. ForeverRoutes ist ein
inoffizielles Fan-Werkzeug und steht in keiner Verbindung zu Blizzard Entertainment, zum Betreiber von WoW Forever
oder zu den oben genannten Projekten.
