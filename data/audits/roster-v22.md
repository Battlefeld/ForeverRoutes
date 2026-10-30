# Gesamtprüfung aller Kombinationen bis Level 20

Modellrevision: `foreverroutes.timeline-model.v2.2`. Geprüft wird der vollständige ausgelieferte Forever-Kader: 56 Volk-/Klassenkontexte, einschließlich getrennter Skyborne-Fraktionen. Das ist eine Prüfung des Codes und des Simulationsmodells, keine Ingame-Kalibrierung.

## Umfang und Nachweise

Ergebnis: **56/56 Kombinationen**, **404/404 optimierte Szenarien** und **112/112 ausgelieferte Routen** bestehen die Prüfung. Kein optimiertes Ereignis wird übersprungen; Zeit, XP und Aktionsnachgang stimmen überein. [Maschinenlesbare Abschlussprüfungen](roster-v22-checks.json).

Die vollständigen Go-Tests, gezielte Tests auf parallele Zugriffe, `go vet`, Windows-AMD64-Build, strikter Routencheck, Lua-Export einschließlich Fallbackauswahl sowie sämtliche SHA-256-Werte des 129-Dateien-Bundles wurden erfolgreich geprüft. Der Datensatz stimmt bytegenau mit dem Neuaufbau aus den gepinnten Quellen überein. Alle lokalen Markdown-Verweise und JSON-Dokumente sind gültig.

- 404 Optimierungsszenarien: Solo und optionale Dungeons, jeweils ohne Zauberstab/Schlafsack, mit Schlafsack und freier Auswahl; für die 17 Zauberstab-Kontexte zusätzlich Zauberstab und Zauberstab plus Schlafsack. Camping wird als Vorbereitung verglichen.
- 112 separat gekennzeichnete RXP-Questprojektionen. Ihre Auslassungen bleiben sichtbar; Originalkapitel werden nicht optimiert oder überschrieben.
- Jede gewählte optimierte Route muss Level 20 erreichen, alle Pflicht-Klassenreihen abschließen und die ausgewählten Optionen einhalten. Tatsächliche Aktionen werden erneut nachgespielt; Zeit und XP müssen übereinstimmen. Optimierte Ausgaben enthalten keine übersprungenen Ereignisse.
- Der vollständige Prüflauf samt Ereignislisten aller Varianten liegt in [roster-v22.json.gz](roster-v22.json.gz), die lesbare Zusammenfassung in [roster-v22-summary.json](roster-v22-summary.json). Die 112 frei gewählten Routen werden unter `data/routes` ausgeliefert.
- Gefrorene Eingaberouten: [roster-inputs.json.gz](roster-inputs.json.gz), Ausgangscommit `b90282dc5c510c3aef596c83dbec8dc7eae15a2a`. Bekannte frühere Untoten-Hexer-Ergebnisse werden zusätzlich als unabhängige Seeds geprüft, damit eine bereits bekannte gute Route nicht vergessen wird.

## Behobene Fehler

Die alten 112 Routendateien enthielten zusammen 8754 gemeldete übersprungene Ereignisse ([ursprünglicher Diagnosebericht](legacy-roster-diagnostics.json.gz)). Dazu gehörten doppelte Annahmen/Abgaben sowie inzwischen nicht mehr modellierbare Quests. Erledigte beziehungsweise bereits angenommene Quests sind nun idempotent; optimierte Ausgaben werden aus den tatsächlich ausgeführten Aktionen aufgebaut und vollständig erneut geprüft. Der Originalfallback bleibt davon getrennt.

`EventsOf` verlor bislang XP innerhalb eines Levels und zog beim ersten bearbeiteten Ziel sämtliche Ziele derselben Quest vor. Einzelziele und genaue XP-Zwischenstände bleiben jetzt erhalten. Überflüssige Ereignisse und bereits abgeschnittene Enden werden gemeinsam entfernt und anschließend nachgespielt; das vermeidet teures wiederholtes Nachspielen jeder wirkungslosen Quellzeile.

Der Questzähler des Guide-Nachgangs zählt jetzt auch innerhalb eines Dungeonlaufs angenommene und abgeschlossene Quests. Bereits vor einem Live-Nachgang erledigte Quests werden dabei nicht erneut gezählt.

Ein wiederverwendetes Kampfprofil teilte veränderliche Fallback-/Randwert-Caches zwischen Welten. Jede Welt besitzt nun eine eigene Instanz dieser Caches. Profilkurven, Modellgrenzen, Charakter-/Fraktionszuordnung und Live-Fakten werden geprüft. Doppelte aktive Quests, negative Fortschritte, widersprüchliche Level/XP, ungültige Positionen und nichtendliche Werte werden abgelehnt.

Acht Priesterkurven auf Level 32, 34, 38 und 39 meldeten nach längerer Kampfdauer weniger kumulierten Schaden. Beim Laden wird eine ausdrücklich gekennzeichnete konservative monotone Hüllkurve verwendet: Frühere Schadenswerte werden abgesenkt, niemals spätere angehoben. Die eingefrorenen Quelldateien und ForeverSim bleiben unverändert. Alle Klassenprofile liefern über Level 1–60 und passende ausgelieferte NPCs endliche positive Kampfzeiten.

`replay` übernimmt Charakter, Vorbereitungen, Schlafsackmodell und Dungeonbetrieb aus gespeicherten Routen. `extend` übernimmt zusätzlich den gewählten Zauberstab-Umweg. `timeline` verweigert falsch zugeordnete Charakterseeds. Dungeonaktionen prüfen auch die obere Stufengrenze. Der Routencheck überspringt beschädigte Dateien nicht mehr still und kann unvollständige Routen nicht als erfolgreiche Prüfung durchlassen.

Der Kadervergleich übernimmt neu gefundene Solo-Ergebnisse auch in den Kandidatenpool für optionale Dungeons. Passende bekannte Varianten werden erneut unter denselben Modellparametern geprüft und behalten. Die abschließende Prüfung erzwingt: freie Auswahl ist nie langsamer als eine bekannte passende feste Variante; das Erlauben von Dungeons darf die bekannte schnellere Solo-Route nicht verdrängen.

## Fallback und Veröffentlichung

Der normale RXP-Fallback enthält 30 unveränderte Kapitel mit 56 passenden Kapitelketten bis 20. Quelle, Revision, Dateihashes und Lizenz werden mitgeliefert. `next -route auto` kann darauf zurückfallen; der Addon-Export enthält vollständige Originalanweisungen und eine Auswahlfunktion. [Vertrag und Grenzen](../../docs/rxp-fallback.md).

Die Releaseprüfung spielt alle 112 ausgelieferten Routen mit den jeweiligen Modellparametern nach. Angegebene Zeiten müssen zum aktuellen Datensatz passen; Pflichtreihen, einmalige Dungeons innerhalb ihrer Stufen, Zeitbilanz und erneuter Aktionsnachgang werden geprüft. Nach Änderungen an Daten oder Regeln können veraltete Routen dadurch keine automatische Veröffentlichung passieren. Lokale Markdown-Verweise und sämtliche JSON-Dateien werden zusätzlich geprüft. Der RXP-Katalog wird gegen die festgeschriebenen Quellkapitel reproduziert.

ForeverSim-Livefeedback bleibt vollständig deaktiviert. Es benötigt weiterhin bestätigten Release und ausdrückliche spätere Nutzerzustimmung. Keine Einstellung wird mit diesem Push vorweggenommen.

## Grenzen

Die Optimierung liefert die beste gefundene Modellroute, keinen Beweis eines globalen Optimums. Je Suchlauf sind 12 Sekunden für die Suchphasen eingestellt; Vorbereitung, erforderliche Ketten, Nachspielung und Kostenberichte benötigen zusätzliche Zeit. Bereits gespeicherte Szenarien können nach erneuter physischer Validierung fortgesetzt werden. Dungeons sind auf Teilmengen geprüft, ihre Platzierung bleibt eine begrenzte Suche; nicht jede Kombination muss im Zeitbudget vollständig untersucht werden.

Klassenkostenberichte nutzen bei der Gesamtprüfung direkte Modellvergleiche ohne die bisher zusätzliche Acht-Sekunden-Suche je Vergleich. Das ändert keine gemessene Route und verhindert, dass die Diagnose hunderte unnötige Suchläufe startet.

Kampf-, Reise-, Ereignis-, Berufs- und Materialkosten bleiben teilweise Schätzungen. Für alle Völker einer Klasse gilt dasselbe eingefrorene Kampfprofil; Volksfähigkeiten, vollständige Ausrüstung, Talente und zustandsabhängige Klassenfähigkeiten werden damit nicht vollständig separat gerechnet. Pflicht-Klassenquests werden abgeschlossen; das Einfügefenster von zwei Leveln ist eine Heuristik, keine garantierte Abschlussfrist. Der Schlafsack verwendet das ausgewiesene Zwei-Stunden-Modell; eine Vergleichsrechnung mit einer Stunde ist im vorherigen Zeitstrahlbericht dokumentiert.

XP-Bufffood und gemeinsames Ausruhen für Camping/Schlafsack sind nicht vollständig als eigene Inventar- und Ablaufentscheidungen modelliert.

Die 179 ausdrücklich dokumentierten Quest-Datenlücken verschwinden nicht durch einen erfolgreichen Routentest. Solche Quests geben keine erfundenen Belohnungen und bleiben ausgeschlossen. Eine RXP-Projektionszeit ist wegen nicht vollständig ausgeführter dynamischer RXP-Anweisungen keine exakte Originalguide-Spielzeit. RXP-Auslassungen stehen vollständig in der komprimierten Evidenz.

## Reproduzieren

```sh
FR_ROSTER_REBUILD=/tmp/roster FR_ROSTER_BUDGET=12s GOMAXPROCS=6 \
  go test ./cmd/foreverroutes -run '^TestRebuildRoster$' -parallel 6 -count=1 -timeout 60m -v
# Unterbrochene Szenarien validieren und fortsetzen: zusätzlich FR_ROSTER_RESUME=1
python3 tools/apply_roster_review.py /tmp/roster
go test ./...
go test -race ./route ./cmd/foreverroutes ./fallback -run 'TestTimeline|TestCheckpoint|TestGuideKeeps|TestSharedProfile|TestNextUsesRXP|TestLiveRejects'
go vet ./...
python3 tools/check_refs.py
go run ./cmd/foreverroutes check-routes -out /tmp/route-check.md
go run ./cmd/foreverroutes export-addon -out /tmp/Routes.lua
```

Ein Suchlauf hängt bei einem Zeitbudget von der Rechnerleistung ab. Die geprüften gespeicherten Ergebnisse lassen sich dagegen mit `TestShippedRoutesMatchCurrentModel` exakt nachspielen. [Aktuelle Zeiten je Kombination](../routes/README.md).
