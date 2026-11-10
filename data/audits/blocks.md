# Routen aus Gebietsblöcken: erster Durchstich gegen RestedXP

Stand 10.11.2026, Ork-Schamane, solo. Zahlen sind Modellzeiten des bestehenden Nachspiel-Prüfers, keine Ingame-Messung.
Rohdaten: [bis 12](blocks-orc-shaman-12.json), [bis 20](blocks-orc-shaman-20.json).

```sh
go run ./cmd/foreverroutes blocks -race Orc -class Shaman -target 12 -out data/audits/blocks-orc-shaman-12.json
```

## Was gebaut ist

- `tools/foreverdb_full.py` und der Workflow `foreverdb full list`: die ganze Questliste von ForeverDB (alle Stufen),
  abgelegt in `data/audits/foreverdb-full.json.gz`. Data from ForeverDB.net (https://foreverdb.net).
- Paket `blocks`: liest diese Liste, stuft jede Quest ein (im Spiel gesehen, Forever-Daten, nur Classic) und schreibt die
  aus ForeverDBs Ketten übernommenen Vorquests um: „Complete first“ bleibt Pflicht, „Offered after“ wird „möglichst
  vorher“, „Chain link“ wird optional. Der ausgelieferte Datensatz bleibt unverändert; die Regel gilt nur im Speicher.
- `route.Sweep`: baut eine Route aus Aufenthalten. Was nah ist, wird zuerst erledigt (abgeben, annehmen, Ziel
  bearbeiten); erst wenn nichts mehr nah ist, geht es dorthin, wo für den Weg am meisten zu tun ist. Die Quests kommen
  von außen, der Sweep bestimmt nur die Reihenfolge. Jede Route läuft durch denselben Nachspiel-Prüfer wie alle anderen.
- Befehl `blocks`: vergleicht RestedXP-Reihenfolge und Blockrouten mit demselben Modell bis zur selben Stufe, dazu mit
  halber und doppelter Kampfzeit.

## Vorquests

146 Quests im Datensatz hatten Vorquests aus ForeverDBs Ketten, die keine Pflicht sind; 90 davon haben jetzt eine
„möglichst vorher“-Quest. Gegen die Reihenfolge der 56 RestedXP-Ketten geprüft: vorher 42 Quests, die RestedXP annimmt,
bevor die Vorquest des Datensatzes abgegeben ist, nachher 7, keine neue. Die 7 stammen aus anderen Quellen oder sind
echte Vorquests (786, 1963, 2379, 2842, 5041, 5625, 5627) und müssen einzeln geprüft werden.

Offen: Ob eine der 90 „möglichst vorher“-Quests in Forever doch Pflicht ist, zeigt nur das Spiel.

## Ergebnis

| Route | bis 12 | bis 20 | Laufen (20) | Kampf (20) | davon Grinden (20) | Quests (20) |
|---|---:|---:|---:|---:|---:|---:|
| RestedXP-Reihenfolge, Ketten wie im Datensatz | 5,24 h | 16,09 h | 4,09 h | 11,36 h | 6,00 h | 90 |
| RestedXP-Reihenfolge, Ketten korrigiert | 5,06 h | 15,24 h | 4,36 h | 10,22 h | 4,87 h | 108 |
| Blöcke, dieselben Quests wie RestedXP | 5,17 h | 15,76 h | 6,41 h | 8,98 h | 0,85 h | 124 |
| Blöcke, alle Quests der Gebiete | 5,04 h | 15,87 h | 6,80 h | 8,67 h | 0,00 h | 122 |
| Blöcke, eigene Auswahl | 4,50 h | 14,79 h | 5,87 h | 8,55 h | 0,00 h | 119 |

Zum Vergleich: Die ausgelieferte Solo-Route des bisherigen Planers steht bei 14,13 h bis 20 (87 Quests).

- Die korrigierten Ketten allein bringen der RestedXP-Reihenfolge 18 Quests mehr und 0,85 h bis 20.
- Mit denselben Quests ist die Blockreihenfolge 2 bis 3 % langsamer als RestedXP: Sie läuft deutlich mehr (6,4 h gegen
  4,4 h bis 20) und kämpft dafür weniger. Der Sweep ist beim Laufen noch klar schlechter als die Handarbeit von RestedXP.
- Mit eigener Auswahl (alle Quests der Gebiete, dann nacheinander weggelassen, was die Route schneller macht) ist die
  Blockroute 11 % (bis 12) und 3 % (bis 20) schneller als RestedXP.
- Der bisherige Planer bleibt bis 20 vorn. Die Blockroute ist ein erster Durchstich, kein Ersatz.

## Was den Vergleich schief macht

- Die RestedXP-Zeile ist RestedXP, soweit der Datensatz folgen kann. Bis 12 kann er 20 Guide-Schritte nicht nachspielen,
  bis 20 sind es 38: Quest 786 gilt als nicht machbar (Vorquest 785), 96604 und 96821 haben nicht erfüllbare Ziele,
  96873 ist eine Berufsquest, 96876/96877 haben keinen erreichbaren Startgegenstand. Was fehlt, ersetzt der Nachspieler
  durch Grinden; das benachteiligt RestedXP.
- Die Kampfzeiten sind nicht kalibriert. Die Unterschiede zwischen den Routen (2 bis 11 %) liegen in der Größenordnung
  dieser Unsicherheit.
- Die eigene Auswahl ist auf dieses Modell hin ausgedünnt und damit ihm gegenüber im Vorteil.

## Hängt es vom Spieltempo ab?

| Kampfzeit | RestedXP bis 12 | Blöcke (eigene Auswahl) bis 12 | RestedXP bis 20 | Blöcke (eigene Auswahl) bis 20 |
|---|---:|---:|---:|---:|
| halb | 3,91 h | 3,73 h | 11,38 h | 11,53 h |
| normal | 5,06 h | 4,50 h | 15,24 h | 14,79 h |
| doppelt | 7,67 h | 6,55 h | 23,02 h | 21,90 h |

Bis 12 bleibt die Rangfolge gleich. Bis 20 kippt sie bei halber Kampfzeit: Wer schnell tötet, fährt mit der
grindlastigeren RestedXP-Reihenfolge besser, wer langsam tötet, mit der Blockroute. Das Kampfmodell lässt sich für die
Routenwahl also nicht ganz abschalten.

## Bestätigung in Forever

Von den 147 Quests, die RestedXP oder die Blockroute bis 20 nutzen, haben Spieler bei 141 den Questgeber im Spiel
gesehen; 3 sind Forever-Daten ohne gesehenen Questgeber, 2 nur Classic, 1 fehlt in ForeverDBs Liste.

## Noch nicht gebaut

- Dungeon-Blöcke (Flammenschlund mit Vorquests und Annahmeliste) und das Merkmal teilbar / nicht teilbar.
- Feste, benannte Blöcke: Die „Blöcke“ hier sind die Aufenthalte, die der Sweep ergibt, noch keine gespeicherten
  Bausteine, die man frei aneinanderreiht.
- Weniger Laufen: Runden über mehrere Hubs planen statt immer zum nächsten Ziel.
- Export ins Addon und der Collector-Hinweis „Quest wurde nicht angeboten“.
