# Routen aus Gebietsblöcken: erster Durchstich gegen RestedXP

Stand 10.11.2026, Ork-Schamane. Zahlen sind Modellzeiten des bestehenden Nachspiel-Prüfers, keine Ingame-Messung.
Rohdaten: [bis 12](blocks-orc-shaman-12.json), [bis 20](blocks-orc-shaman-20.json).

```sh
go run ./cmd/foreverroutes blocks -race Orc -class Shaman -target 12 -out data/audits/blocks-orc-shaman-12.json
```

## Was gebaut ist

- `tools/foreverdb_full.py` und der Workflow `foreverdb full list`: die ganze Questliste von ForeverDB (alle Stufen),
  abgelegt in `data/audits/foreverdb-full.json.gz`. Data from ForeverDB.net (https://foreverdb.net).
- Paket `blocks`: liest diese Liste und stuft jede Quest ein (im Spiel gesehen, Forever-Daten, nur Classic). Die
  Kettenregeln, die es anfangs nur im Speicher anwandte, stehen inzwischen im Datensatz (siehe Vorquests).
- `route.Sweep`: baut eine Route aus Aufenthalten. Was nah ist, wird zuerst erledigt (abgeben, annehmen, Ziel
  bearbeiten); erst wenn nichts mehr nah ist, geht es dorthin, wo für den Weg am meisten zu tun ist, auf Wunsch mit
  Vorausspielen der Kandidaten. Die Quests kommen
  von außen, der Sweep bestimmt nur die Reihenfolge. Jede Route läuft durch denselben Nachspiel-Prüfer wie alle anderen.
- Befehl `blocks`: vergleicht RestedXP-Reihenfolge und Blockrouten mit demselben Modell bis zur selben Stufe, dazu mit
  halber und doppelter Kampfzeit.

## Vorquests

Seit dem Neubau des Datensatzes auf diesem Branch stecken die Regeln im Datensatz selbst
(`internal/build/chains.go`): „Complete first“ ist Pflicht, „Offered after“ wird weiche Vorquest (`preSoft`, 109
Quests), „Chain link“ ist keine. 26 Quests behalten eine Pflicht-Vorquest aus ForeverDBs Ketten. Der Live-Planer und
der Export rechnen damit ohne Zusatzschalter; die Zeilen „Ketten wie im Datensatz“ und „korrigiert“ sind deshalb jetzt
gleich. Vor dem Neubau widersprachen 42 Quests der Reihenfolge der 56 RestedXP-Ketten, mit den neuen Regeln 7.

Quest 786 (Thwarting Kolkar Aggression) hing an Quest 785, die keinen Questgeber hat und in ForeverDB fehlt; sie ist
jetzt ohne Vorquest (`quest-fixes.json`). Offen bleiben 1963, 2379, 2842, 5041, 5625 und 5627.

Offen: Ob eine der weichen Vorquests in Forever doch Pflicht ist, zeigt nur das Spiel (Collector, siehe unten).

## Ergebnis (solo)

| Route | bis 12 | bis 20 | Laufen (20) | Kampf (20) | davon Grinden (20) | Quests (20) |
|---|---:|---:|---:|---:|---:|---:|
| RestedXP-Reihenfolge | 4,97 h | 15,15 h | 4,35 h | 10,13 h | 4,78 h | 109 |
| Blöcke, dieselben Quests wie RestedXP | 5,65 h | 15,39 h | 6,43 h | 8,60 h | 0,90 h | 121 |
| Blöcke, alle Quests der Gebiete | 5,08 h | 15,50 h | 5,74 h | 9,34 h | 0,00 h | 131 |
| Blöcke, eigene Auswahl | 4,92 h | 14,65 h | 5,57 h | 8,67 h | 0,00 h | 135 |

Zum Vergleich: Die ausgelieferte Solo-Route des bisherigen Planers steht bei 14,13 h bis 20 (87 Quests).

- Mit eigener Auswahl ist die Blockroute bis 12 gleichauf mit RestedXP und bis 20 3 % schneller. Mit denselben Quests
  wie RestedXP ist sie langsamer (14 % bis 12, 2 % bis 20).
- Die Blockroute macht mehr Quests und grindet nicht, läuft dafür mehr (5,6 h gegen 4,4 h bis 20).
- Der bisherige Planer bleibt bis 20 vorn.

## Was den Vergleich schief macht

- Die RestedXP-Zeile ist RestedXP, soweit der Datensatz folgen kann. Bis 20 kann er 32 Guide-Schritte nicht
  nachspielen (96604 und 96821 haben nicht erfüllbare Ziele, 96873 ist eine Berufsquest, 96876/96877 haben keinen
  erreichbaren Startgegenstand). Was fehlt, ersetzt der Nachspieler durch Grinden.
- Die Kampfzeiten sind nicht kalibriert. Die Unterschiede zwischen den Routen liegen in der Größenordnung dieser
  Unsicherheit.
- Die Einstellungen des Sweeps werden pro Lauf aus einem kleinen Raster gewählt; je nach Raster schwankt eine Route um
  einige Prozent.

## Hängt es vom Spieltempo ab?

| Kampfzeit | RestedXP bis 12 | Blöcke (eigene Auswahl) bis 12 | RestedXP bis 20 | Blöcke (eigene Auswahl) bis 20 |
|---|---:|---:|---:|---:|
| halb | 3,85 h | 3,82 h | 11,33 h | 12,21 h |
| normal | 4,97 h | 4,92 h | 15,15 h | 14,65 h |
| doppelt | 7,48 h | 6,92 h | 22,84 h | 21,86 h |

Bis 20 hängt die Rangfolge vom Tempo ab: Wer schnell tötet, fährt mit der grindlastigeren RestedXP-Reihenfolge besser
(8 % schneller), wer langsam tötet, mit der Blockroute (4 % schneller). Das Kampfmodell lässt sich für die Routenwahl
also nicht ganz abschalten.

## Dungeon-Blöcke

Mit `-dungeons` läuft ein Dungeon als Block: Seine Quests und was zu ihnen führt, werden unterwegs angenommen, der Run
ist ein Kandidat unter den Orten, zu denen es als Nächstes geht, danach wird abgegeben. Rohdaten:
[bis 20 mit Dungeons](blocks-orc-shaman-20-dungeons.json).

| Route bis 20 | Zeit |
|---|---:|
| RestedXP-Reihenfolge (solo, ohne Dungeons) | 15,15 h |
| Blöcke, RestedXPs Quests und Dungeons | 14,19 h |
| Blöcke, eigene Auswahl und Dungeons | 14,95 h |
| ausgelieferte Dungeon-Route des bisherigen Planers | 12,17 h |

Der Vergleich mit RestedXP ist hier nur ein Anhalt, weil RestedXPs normaler Guide keine Dungeons läuft.

Je Run stehen im Bericht die Quests des Dungeons mit „teilbar“, den Quests davor und dem Stand in der Route
(`dungeonBlocks`). „Teilbar“ stammt aus cmangos' Questdaten (`quest-flags-classic.json`, 4245 Quests, 2566 teilbar)
und ist Classic-Stand, für Forever nicht bestätigt; Forever-eigene Quests haben keine Angabe. Eine teilbare Quest mit
Vorquest kann nur annehmen, wer die Vorquest hat.

## Bestätigung in Forever

Von den 166 Quests, die RestedXP oder die Blockroute bis 20 (solo) nutzen, haben Spieler bei 142 den Questgeber im
Spiel gesehen; 11 sind Forever-Daten ohne gesehenen Questgeber, 3 nur Classic, 10 fehlen in ForeverDBs Liste.

## 1 bis 40

Rohdaten: [bis 40 mit Dungeons](blocks-orc-shaman-40-dungeons.json), Blöcke in `data/blocks/orc-shaman-dungeons.json`.

```sh
go run ./cmd/foreverroutes blocks -target 40 -dungeons -quick -paces=false -zones all \
  -against data/routes-ext/orc-shaman-dungeons.json
```

| Route bis 40 (mit Dungeons) | Zeit | Laufen | Kampf | davon Grinden | Quests |
|---|---:|---:|---:|---:|---:|
| bisheriger Planer, fortgesetzte Route (`routes-ext`) | 75,14 h | 25,14 h | 47,83 h | 12,15 h | 275 |
| RestedXP bis 21, danach nur Grinden | 78,95 h | 6,52 h | 70,34 h | 65,08 h | 143 |
| Blöcke, RestedXPs Quests, Dungeons, sonst Grinden | 85,97 h | 24,56 h | 60,81 h | 39,31 h | 200 |
| Blöcke, alle Quests der Welt im Stufenfenster | 101,30 h | 45,51 h | 54,47 h | 0,00 h | 382 |
| Blöcke, eigene Auswahl | 95,24 h | 45,18 h | 48,73 h | 0,00 h | 388 |

- Bis 40 ist die Blockroute 14 bis 27 % langsamer als die fortgesetzte Route des bisherigen Planers. Sie läuft fast
  doppelt so viel, sobald sie alle Quests nimmt.
- Im Modell ist reines Grinden ab 21 fast so schnell wie die beste Route (79 h gegen 75 h). Solange das Kampfmodell
  nicht kalibriert ist, sagt der Vergleich ab 20 deshalb wenig über gute Questauswahl.
- Für lange Routen neu im Sweep: Quests nur im Stufenfenster (höchstens 5 unter, 3 über der eigenen Stufe), nur wenn
  ihre Ziele jetzt machbar sind (sonst füllte sich das Questlog und die Route blieb mit 32 stecken), ein Gebiet wird
  abgearbeitet, bevor das nächste beginnt (`Zones`), und Quests, die mehr kosten als bringen, fallen in einem
  Sammeldurchgang weg.
- Die Blockroute bis 40 läuft 14 Dungeons, vom Flammenschlund bis zu den Hügeln der Klingenhauer und dem
  Scharlachroten Kloster.

## Feste Blöcke

`blocks -blocks-out` schreibt die Route als benannte Blöcke in Reihenfolge (`data/blocks/<key>.json`: „Durotar 3“,
„Orgrimmar 6“ mit dem, was der Block annimmt, bearbeitet und abgibt, dazu Questnamen und teilbar ja/nein).
`blocks -from-blocks <datei>` spielt eine solche Datei in ihrer Reihenfolge ab. Blöcke lassen sich umstellen oder
kürzen; die Route wird danach wie jede andere nachgespielt und bleibt damit gültig. Abgespielt ist die Route etwas
langsamer als das Original (bis 12: 4,70 h statt 4,50 h), weil ein Block seine Ziele dann am Stück erledigt.

## Im Spiel testen

`blocks -route-out` schreibt die schnellste Blockroute als Routendatei (`data/blocks/routes/`), `export-addon -routes
data/blocks/routes` macht daraus die `Routes.lua` des Addons. Für den Ork-Schamanen liegen vor: solo bis 20 (15,0 h im
Export) und mit Dungeons bis 40 (80,5 h im Export).

Der Collector hält jetzt fest, wenn eine geplante Quest beim Questgeber nicht angeboten wird (`notOffered` in den
SavedVariables: Quest, NPC, Stufe, was stattdessen angeboten wurde), und sagt es einmal im Chat. Nur wenn das Spiel die
Angebotsliste mit Quest-IDs liefert und der Spieler am Ort des Schritts steht; `import_client_quests.py` übernimmt die
Beobachtungen. Ausgewertet werden sie noch nicht.

## Nachoptimieren

`blocks -polish 40s` setzt die lokalen Änderungen des bisherigen Optimierers (`Improve`) auf die Blockroute an. Bis 12
brachte das 1 bis 3 %. Auf die RestedXP-Reihenfolge angewandt wurde die Route langsamer (6,02 h statt 5,06 h): kein
Fehler, `Improve` ergänzt zuerst die Pflicht-Klassenquests des Schamanen (Totems), die RestedXPs normale Kapitel bis
12 auslassen, und misst gegen diese ergänzte Route. Die Beschreibung von `Improve` sagt das jetzt.

## Noch nicht gebaut

- Weniger Laufen: Runden über mehrere Hubs planen und Abgaben bündeln. Hier liegt der Rückstand, bis 40 deutlich.
- Grinden als gleichwertige Wahl im Sweep: Vorausspielen mit Grind-Kandidat (`Grind`) ist eingebaut, hat in den
  Läufen aber nie gewonnen.
- Auswertung der `notOffered`-Beobachtungen: daraus Vorquests im Datensatz nachtragen.
- Die übrigen Datenlücken, an denen RestedXP-Schritte scheitern (96604, 96821, 96873, 96876, 96877).
