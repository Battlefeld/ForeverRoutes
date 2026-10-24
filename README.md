# ForeverRoutes

> **Datengrundlage:** ForeverRoutes nutzt Daten von **[ForeverDB.net](https://foreverdb.net)** (Data from ForeverDB.net),
> QuestieDB Forever, EverythingQuests Forever, [WarcraftDB Forever](https://forever.warcraftdb.com) und der
> Weltdatenbank von [cmangos classic-db](https://github.com/cmangos/classic-db). Herkunft und Revision jeder Quelle stehen
> im Datensatz selbst (`sources`). ForeverDB ist nicht mit Blizzard Entertainment verbunden.

Routenplaner für WoW Forever, Level 1–30: welche Quests annehmen, wo was töten, wann abgeben und wann Grinden schneller
ist. Go-Modul ohne Fremdabhängigkeiten. Der Planer läuft offline als Kommandozeilenwerkzeug und ist als Bibliothek für
ForeverCore.exe gebaut (Paket `route`); dort liefert später MythicSim (ForeverSim) die echten Kampfzeiten.

## Stand

- Datensatz mit 1758 Quests, 2366 NPCs, 348 Objekten, 59 Flugmeistern, 45 Gasthäusern, Zeppelinen und Schiffen.
- Orc Hexer und Mensch Magier sind die Ausgangstests: Beide erreichen Level 30 mit vollständigen, nachgeprüften Routen.
- **Die Stundenangaben sind nicht kalibriert.** Das eingebaute Kampfmodell (`DefaultProfile`) ist eine Schätzung. Es
  bewertet Grinden im Vergleich zu Quests etwa doppelt so gut wie die Wirklichkeit: Ein Guide-Ablauf (RestedXP) und
  reines Grinden landen bis Level 21 im Modell fast gleichauf. Darum grinden die Routen heute zu viel. Echte Zahlen kommen
  über ForeverCore aus MythicSim (`KillProfile`); mit `guide` lassen sich Guides unter denselben Regeln nachgehen und so
  das Modell prüfen.
- Bekannte Datenlücken (siehe `foreverroutes audit`): 91 Quests brauchen Forever-eigene Gegenstände, deren Beutequelle
  keine Quelle kennt; 82 brauchen hergestellte oder gekaufte Gegenstände; 44 werden über einen Gegenstand gestartet;
  Ruf-, Gebiets- und Zauberziele sind noch nicht modelliert. Diese Quests bleiben markiert und werden nicht geraten.

## Befehle

```sh
go test ./...                      # alle Tests, inkl. Ausgangstest Orc Hexer / Mensch Magier (~3 min)
go test -short ./...               # ohne die langen Planungsläufe

go run ./cmd/foreverroutes plan -race Orc -class Warlock -out orc.json
go run ./cmd/foreverroutes plan -race Human -class Mage -target 30 -restarts 4 -out human.json
go run ./cmd/foreverroutes replay -race Orc -class Warlock -route orc.json
go run ./cmd/foreverroutes guide -race Orc -class Warlock -events guide.json   # Guide unter denselben Regeln
go run ./cmd/foreverroutes audit                                              # was der Datensatz kennt und was fehlt
```

`guide` liest eine Liste von Schritten `[{"op":"accept","quest":4641},{"op":"complete","quest":4641},
{"op":"turnin","quest":4641},{"op":"level","level":6}]`. Guide-Dateien werden nicht eingecheckt (Lizenzen der Guides).

### Daten aktuell halten

```sh
FOREVERDB_API_KEY=… go run ./cmd/foreverroutes update   # alle Quellen prüfen, bei Neuem neu bauen und prüfen
go run ./cmd/foreverroutes build-data                     # Datensatz exakt aus data/sources/lock.json neu bauen
```

`update` fragt die neuesten Stände von QuestieDB, EverythingQuests und cmangos ab (Git), liest die Questliste von
ForeverDB über die API (höchstens einmal pro Stunde, 2,1 s Abstand) und holt für Quests, denen noch Angaben fehlen,
die Clientdaten von WarcraftDB (1,5 s Abstand, Zwischenspeicher). Ändert sich am Datensatz etwas, wird er neu gebaut und
gegen den bisherigen geprüft (keine großen Verluste an Quests, NPCs, Positionen); nur dann werden die Dateien
geschrieben. Neue Commits upstream, die am Inhalt nichts ändern, lassen alles, wie es ist.

Auf GitHub läuft das als Workflow `update` (beim Start von WoW über ForeverCore angestoßen, sonst täglich): Bei
Änderungen laufen alle Tests und der Ausgangstest, dann wird auf `main` gepusht und der Datensatz mit SHA-256 als Release
veröffentlicht. Schlägt etwas fehl, bleibt alles beim alten Stand und ein Issue „Data update failed“ beschreibt den
Grund. Der Workflow `data` prüft bei jeder Änderung, dass der Datensatz byte-genau aus den festgeschriebenen Quellen
entsteht.

## Aufbau

| Pfad | Inhalt |
|---|---|
| `route/` | Planer: Welt je Charakter, Zustand und Regeln (`Apply`/`Replay`), Reisen (Laufen, Flugmeister, Zeppelin, Schiff, Ruhestein), Suche, Guide-Nachgang, Kampfmodell |
| `dataset/` | Format des Datensatzes (`foreverroutes.dataset.v1`), Laden und Schreiben |
| `xp/` | Classic-XP-Regeln: Quest-XP nach Levelabstand, Kill-XP (cmangos), Levelkurve |
| `internal/update/` | Quellen prüfen, neu bauen, Ergebnis prüfen (`update`, `build-data`) |
| `internal/sources/` | Importer: QuestieDB (Lua-Datentabellen), ForeverDB (API), EverythingQuests |
| `internal/merge/` | Zusammenführen der Questquellen feldweise mit Quelle und Vorrangregeln |
| `internal/build/` | Bau des Datensatzes aus Quest-Datenbank, cmangos und WarcraftDB; Umrechnung Kartenprozent → Weltyards |
| `internal/lua/` | Leser für Lua-Tabellenliterale (führt keinen Code aus) |
| `internal/sqldump/` | Leser für den cmangos-SQL-Dump |
| `internal/warcraftdb/` | Abruf und Format der WarcraftDB-Ergänzung |
| `cmd/foreverroutes/` | Kommandozeile |
| `data/routes-data.json.gz` | der fertige Datensatz |
| `data/sources/` | festgeschriebene Stände der Quellen (`lock.json`), ForeverDB-Stand, WarcraftDB-Ergänzung, einzeln belegte Werte |

## Modell in Kürze

- Positionen in Weltyards je Kontinent. Forever-Positionen (Kartenprozent) werden je Karte über NPCs umgerechnet, die in
  beiden Systemen bekannt sind (mittlerer Fehler ~1 Yard). Fehlt Forever eine Position, gilt cmangos.
- Ziele werden an den dichtesten Spawn-Gruppen erledigt (bis zu 4 je Kreatur); die Route nimmt die schnellste.
  Kills zählen für alle Quests im Log, Respawns werden abgewartet, geleerte Gruppen erholen sich über die Respawnzeit.
- Reisen: Laufen (7 Yards/s, Wegfaktor 1,2), Flugmeister ab Entdeckung (im Vorbeigehen innerhalb 60 Yards), Zeppeline und
  Schiffe mit Zeiten aus der cmangos-Transporttabelle, Ruhestein (10 s, 1 h Abklingzeit, gebunden am letzten Gasthaus).
- Suche: Bausteine (eine Quest, ein Questgebiet in drei Größen, Grinden) bewertet nach dem Gewinn gegenüber Grinden;
  die besten werden einige Entscheidungen weit vorausgespielt; mehrere Läufe, das beste Ergebnis wird nachgeprüft.
  Ergebnis ist die beste gefundene Route, kein bewiesenes Optimum.

## Lizenz

Code: MIT (siehe `LICENSE`). Daten: siehe [`data/README.md`](data/README.md); der Datensatz enthält aus cmangos
classic-db abgeleitete Werte (GPL-3.0).
