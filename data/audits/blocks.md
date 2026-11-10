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
  bearbeiten); erst wenn nichts mehr nah ist, geht es dorthin, wo für den Weg am meisten zu tun ist, auf Wunsch mit
  Vorausspielen der Kandidaten. Die Quests kommen
  von außen, der Sweep bestimmt nur die Reihenfolge. Jede Route läuft durch denselben Nachspiel-Prüfer wie alle anderen.
- Befehl `blocks`: vergleicht RestedXP-Reihenfolge und Blockrouten mit demselben Modell bis zur selben Stufe, dazu mit
  halber und doppelter Kampfzeit.

## Vorquests

146 Quests im Datensatz hatten Vorquests aus ForeverDBs Ketten, die keine Pflicht sind; 90 davon haben jetzt eine
„möglichst vorher“-Quest. Gegen die Reihenfolge der 56 RestedXP-Ketten geprüft: vorher 42 Quests, die RestedXP annimmt,
bevor die Vorquest des Datensatzes abgegeben ist, nachher 7, keine neue. Die 7 stammen aus anderen Quellen oder sind
echte Vorquests (786, 1963, 2379, 2842, 5041, 5625, 5627) und müssen einzeln geprüft werden.

Offen: Ob eine der 90 „möglichst vorher“-Quests in Forever doch Pflicht ist, zeigt nur das Spiel.

## Ergebnis (solo)

| Route | bis 12 | bis 20 | Laufen (20) | Kampf (20) | davon Grinden (20) | Quests (20) |
|---|---:|---:|---:|---:|---:|---:|
| RestedXP-Reihenfolge, Ketten wie im Datensatz | 5,24 h | 16,09 h | 4,09 h | 11,36 h | 6,00 h | 90 |
| RestedXP-Reihenfolge, Ketten korrigiert | 5,06 h | 15,24 h | 4,36 h | 10,22 h | 4,87 h | 108 |
| Blöcke, dieselben Quests wie RestedXP | 5,16 h | 15,35 h | 6,15 h | 8,82 h | 0,84 h | 124 |
| Blöcke, alle Quests der Gebiete | 4,93 h | 15,37 h | 6,41 h | 8,56 h | 0,00 h | 124 |
| Blöcke, eigene Auswahl | 4,50 h | 15,37 h | 6,41 h | 8,56 h | 0,00 h | 124 |

Zum Vergleich: Die ausgelieferte Solo-Route des bisherigen Planers steht bei 14,13 h bis 20 (87 Quests).

- Die korrigierten Ketten allein bringen der RestedXP-Reihenfolge 18 Quests mehr und 0,85 h bis 20.
- Bis 20 liegen Blockroute und RestedXP gleichauf (1 % Unterschied). Die Blockroute macht mehr Quests und grindet
  nicht, läuft dafür deutlich mehr (6,2 h gegen 4,4 h). Beim Laufen ist der Sweep klar schlechter als die Handarbeit
  von RestedXP.
- Bis 12 ist die Blockroute mit eigener Auswahl 11 % schneller. Bis 20 hat das Weglassen einzelner Quests nichts
  gebracht; die eigene Auswahl ist dort die Route durch alle Quests der Gebiete.
- Züge zu einem anderen Ort vorauszuspielen (`Look`) hat die Laufzeit kaum gesenkt. Der Hebel liegt woanders:
  Runden über mehrere Hubs und das Bündeln von Abgaben.
- Der bisherige Planer bleibt bis 20 vorn. Die Blockroute ist ein Durchstich, kein Ersatz.

## Was den Vergleich schief macht

- Die RestedXP-Zeile ist RestedXP, soweit der Datensatz folgen kann. Bis 12 kann er 20 Guide-Schritte nicht nachspielen,
  bis 20 sind es 38: Quest 786 gilt als nicht machbar (Vorquest 785), 96604 und 96821 haben nicht erfüllbare Ziele,
  96873 ist eine Berufsquest, 96876/96877 haben keinen erreichbaren Startgegenstand. Was fehlt, ersetzt der Nachspieler
  durch Grinden; das benachteiligt RestedXP.
- Die Kampfzeiten sind nicht kalibriert. Die Unterschiede zwischen den Routen liegen in der Größenordnung dieser
  Unsicherheit.
- Die Einstellungen des Sweeps werden pro Lauf aus einem kleinen Raster gewählt; je nach Raster schwankt eine Route um
  einige Prozent.

## Hängt es vom Spieltempo ab?

| Kampfzeit | RestedXP bis 12 | Blöcke (eigene Auswahl) bis 12 | RestedXP bis 20 | Blöcke (eigene Auswahl) bis 20 |
|---|---:|---:|---:|---:|
| halb | 3,91 h | 3,64 h | 11,38 h | 12,04 h |
| normal | 5,06 h | 4,50 h | 15,24 h | 15,37 h |
| doppelt | 7,67 h | 6,55 h | 23,02 h | 22,21 h |

Bis 12 bleibt die Rangfolge gleich. Bis 20 hängt sie vom Tempo ab: Wer schnell tötet, fährt mit der grindlastigeren
RestedXP-Reihenfolge besser (6 % schneller), wer langsam tötet, mit der Blockroute (4 % schneller). Das Kampfmodell
lässt sich für die Routenwahl also nicht ganz abschalten.

## Dungeon-Blöcke

Mit `-dungeons` läuft ein Dungeon als Block: Seine Quests und was zu ihnen führt, werden unterwegs angenommen, der Run
ist ein Kandidat unter den Orten, zu denen es als Nächstes geht, danach wird abgegeben. Rohdaten:
[bis 20 mit Dungeons](blocks-orc-shaman-20-dungeons.json).

| Route bis 20 | Zeit |
|---|---:|
| RestedXP-Reihenfolge (solo, ohne Dungeons) | 15,24 h |
| Blöcke, RestedXPs Quests und Dungeons | 14,00 h |
| Blöcke, alle Quests der Gebiete und Dungeons | 15,20 h |
| ausgelieferte Dungeon-Route des bisherigen Planers | 12,17 h |

Die Blockroute läuft Flammenschlund mit 13, Höhlen des Wehklagens mit 17 und die Ruinen von Lordaeron. Der Vergleich
mit RestedXP ist hier nur ein Anhalt, weil RestedXPs normaler Guide keine Dungeons läuft.

Flammenschlund in der Route mit RestedXPs Quests (Stufe 13, 43 Minuten):

| Quest | teilbar | vorher nötig | in der Route |
|---|---|---|---|
| 5722 Searching for the Lost Satchel | ja | | erledigt |
| 5723 Testing an Enemy's Strength | ja | | erledigt |
| 5724 Returning the Lost Satchel | im Dungeon erhalten | | erledigt |
| 5725 The Power to Destroy... | ja | | nicht angenommen (Unterstadt) |
| 5728 Hidden Enemies | ja | 5726, 5727 | erledigt |
| 5761 Slaying the Beast | ja | | erledigt |

„Teilbar“ stammt aus cmangos' Questdaten (`quest-flags-classic.json`, 4245 Quests, 2566 teilbar) und ist Classic-Stand,
für Forever nicht bestätigt; Forever-eigene Quests haben keine Angabe. Eine teilbare Quest mit Vorquest kann nur
annehmen, wer die Vorquest hat.

## Bestätigung in Forever

Von den 147 Quests, die RestedXP oder die Blockroute bis 20 nutzen, haben Spieler bei 141 den Questgeber im Spiel
gesehen; 3 sind Forever-Daten ohne gesehenen Questgeber, 2 nur Classic, 1 fehlt in ForeverDBs Liste.

## Noch nicht gebaut

- Feste, benannte Blöcke: Die „Blöcke“ hier sind die Aufenthalte, die der Sweep ergibt, noch keine gespeicherten
  Bausteine, die man frei aneinanderreiht.
- Weniger Laufen: Runden über mehrere Hubs planen und Abgaben bündeln.
- Dungeon-Quests aus anderen Hauptstädten gezielt holen, wenn die Route ohnehin dort vorbeikommt.
- Export ins Addon und der Collector-Hinweis „Quest wurde nicht angeboten“.
