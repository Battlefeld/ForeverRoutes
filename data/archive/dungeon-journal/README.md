# Dungeon-Journal (Archiv)

Daten aus dem früheren Addon ForeverQuests 1-30 (Repository `Battlefeld/forever-addons`, Commit `c292d96`, Stand
15.10.2026), unverändert übernommen, bevor das Repository gelöscht wurde. Sie werden vom Planer **noch nicht** gelesen;
sie sind die Grundlage, sobald Dungeons in die Routen kommen.

| Datei | Inhalt |
|---|---|
| `Dungeons.lua`, `Bosses.lua`, `DungeonEntrances.lua`, `DungeonRoutes.lua` | Dungeons, Bosse mit Stufen, Beute, Eingänge |
| `QuestChains.lua`, `QuestMaps.lua`, `QuestRewards.lua`, `QuestShareability.lua`, `QuestUpdates.lua` | Dungeonquests: Ketten, Kartenpunkte, Belohnungen/XP, Teilbarkeit, Änderungen in Forever |
| `DungeonCatalog.lua`, `DungeonLoot.lua`, `DungeonQuestData.lua` | geprüfte Zuordnungen Level 1-30 (Dungeon ↔ Quest, Boss-Beute, Zugangsketten) |
| `SOURCES.txt` | Quellenliste je Dungeon (wowforevertalents.com, foreverchanges.pro, wowf.io, wowsrc.com, Stand 22.09.2026) |

Die Dateien sind Lua-Datentabellen eines WoW-Addons. Wer sie verwendet, prüft die Angaben gegen aktuelle Quellen:
Teile sind Classic-Zuordnungen, die in Forever noch nicht bestätigt sind (so in `SOURCES.txt` vermerkt).
