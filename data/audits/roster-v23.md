# Gesamtprüfung aller Kombinationen bis Level 20 (v2.3)

Modellrevision: `foreverroutes.timeline-model.v2.3`. Gegenüber [v2.2](roster-v22.md) ändert sich nur das Laufen: Wege
zwischen den Orten, die Routen anlaufen, kommen aus der gemessenen Wegtabelle (`data/walktable.json.gz`,
[Messung](../../tools/walktable/README.md)), alle anderen aus dem neu kalibrierten 200-Yard-Gitter. Das ist eine
Prüfung des Codes und des Simulationsmodells, keine Ingame-Kalibrierung.

## Ergebnis

**56/56 Kombinationen**, **404/404 optimierte Szenarien**: Jede gewählte Route erreicht Level 20, schließt alle
Pflicht-Klassenreihen ab, hält die gewählten Optionen ein, überspringt kein Ereignis, und ihr erneuter Aktionsnachgang
mit dem finalen Modell stimmt in Zeit und XP. Je Suchlauf 30 Sekunden Suchbudget (v2.2: 12), Seed 20261027, auf 4
Kernen. [Variantenübersicht](roster-v23-summary.json), [vollständige Evidenz](roster-v23.json.gz),
[ausgelieferte Routen](../routes/README.md).

| Mittelwert über 56 Kombinationen | v2.2 | v2.3 |
|---|---:|---:|
| Solo | 13,97 h | 13,71 h (−1,8 %) |
| Optionale Dungeons | 12,63 h | 12,44 h (−1,5 %) |

Einzelne Kombinationen weichen in beide Richtungen ab (bis 53 Minuten schneller, bis 56 Minuten langsamer): genauere
Laufwege (Brücken, Stockwerke, Umwege um Berge kosten jetzt, was sie kosten; das 9 % zu lange Gitter ist korrigiert)
und eine andere Suche. Die Zeiten sind Modellzeiten; Kampf, Dungeonläufe und Gruppensuche bleiben Schätzungen bis zum
[Ingame-Test](../../docs/ingame-test.md).

## Reproduzieren

```sh
FR_ROSTER_REBUILD=/tmp/roster FR_ROSTER_BUDGET=30s GOMAXPROCS=4 \
  go test ./cmd/foreverroutes -run '^TestRebuildRoster$' -parallel 4 -count=1 -timeout 300m -v
FR_ROSTER_FINALIZE=/tmp/roster go test ./cmd/foreverroutes -run '^TestFinalizeRosterEvidence$' -count=1 -v
python3 tools/apply_roster_review.py /tmp/roster
go test ./...
```
