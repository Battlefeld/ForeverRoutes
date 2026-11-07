# Klassenbezogene Routenprüfung – Modell v2.4

Stand: 7. November 2026. Modell: `foreverroutes.timeline-model.v2.4`.

## Änderungen

- Neue CLI-Planungen wählen das eingefrorene Kampfprofil der jeweiligen Klasse automatisch neben dem Datensatz.
- Krieger, Schurke und Paladin erhalten keine erfundene dreisekündige Fernkampf-Anlaufzeit bei eingehendem Schaden.
- Klassenquestketten lehnen zyklische AND-/OR-Voraussetzungen ab. Eine gültige alternative Vorquest bleibt möglich;
  gemeinsame Vorquests werden nur einmal eingeplant.
- Bei einer Modelländerung werden bereits gültige Routen als Kandidaten unter dem neuen Modell erneut bewertet.
  Eine kürzere Neuberechnung darf eine bekannte, weiterhin ausführbare Lösung nicht allein wegen veränderter Zeitangaben verlieren.
- Bestehende Optimierungs-Checkpoints werden durch die neue Modellrevision getrennt.

## Umfang und Grenzen

Der ausgelieferte Umfang bleibt Stufe 1–20 für alle neun Klassen in 56 unterstützten Volk-/Klassen-/Fraktionskontexten.
Solo und optionale Dungeons bleiben getrennt; Zauberstab und Schlafsack werden in eigenen Vorbereitungsszenarien geprüft.

Das ist eine Optimierung innerhalb des bestehenden Modells, kein Beweis einer global optimalen Ingame-Route.
Kampfprofile verwenden weiterhin synthetische Charaktere ohne individuelle Ausrüstung und Talente.
Reise-, Kampf-, Erholungs- und Gruppensuchzeiten enthalten Schätzungen; es werden keine gemessenen Ingame-Zeiten behauptet.
Originale RestedXP-Kapitel und ihre Lizenzen bleiben unverändert.

## Reproduktion

```sh
python tools/seed_roster_resume.py .generated/class-routing-v24
FR_ROSTER_REBUILD="$PWD/.generated/class-routing-v24" FR_ROSTER_RESUME=1 FR_ROSTER_TOLERATE=1 FR_ROSTER_BUDGET=2s go test ./cmd/foreverroutes -run '^TestRebuildRoster$' -parallel 4 -count=1 -timeout 40m -v
FR_ROSTER_REPAIR="$PWD/.generated/class-routing-v24" go test ./cmd/foreverroutes -run '^TestRepairRosterScenarios$' -count=1 -timeout 40m
FR_ROSTER_FINALIZE="$PWD/.generated/class-routing-v24" go test ./cmd/foreverroutes -run '^TestFinalizeRosterEvidence$' -count=1 -timeout 40m
python tools/apply_roster_review.py .generated/class-routing-v24
go test -timeout 25m ./...
go vet ./...
```

Die maschinenlesbaren Ergebnisse werden nach erfolgreicher Gesamtprüfung als
`roster-v24.json.gz` und `roster-v24-summary.json` abgelegt. Der Release-Test spielt alle 112 ausgelieferten
Routen samt Pflichtklassenquests erneut nach und vergleicht ihre Zeitangaben mit dem aktuellen Modell.

## Ergebnis

- 56 Kontexte, 404 optimierte Vorbereitungsszenarien und 112 separate RXP-Arbeitsprojektionen vollständig erstellt.
- Alle 404 optimierten Szenarien bestanden den erneuten Aktions-Nachgang und die Zeitbilanz; keine Reparatur nötig.
- Alle 112 ausgelieferten Routen bestanden den Release-Nachgang unter Modell v2.4.
- Der zusätzliche Vergleich mit den vorherigen veröffentlichten Gewinnern bestätigt: Keine weiterhin ausführbare
  alte Lösung ist unter demselben korrigierten Modell schneller als die entsprechende neue ausgelieferte Route.
- Vollständige Go-Tests, `go vet`, Klassenprofil-/Live-Anfrage-Regressionen und Clientquest-Importtests bestanden.
- Lua-HUD-Harness: 113 Prüfungen bestanden; Questkollektor und generierter Lua-Routenbestand erfolgreich geladen.
- Addon-ZIP mit 27 Dateien einschließlich aller Lizenzen erfolgreich gebaut.

Diese Prüfungen sind offline ausgeführt. Ein Ingame-Durchlauf aller Klassen und Levelbereiche steht weiterhin aus.
