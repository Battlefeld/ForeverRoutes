# Zeitstrahl-Optimierung: Stand und Validierung

Historischer Prüfstand v2.1. Die aktuelle vollständige Kaderprüfung und neu berechneten Routendateien stehen in [roster-v22.md](roster-v22.md).

Implementiert im Routenplaner, Modellrevision `foreverroutes.timeline-model.v2.1`. ForeverSim wurde nicht geändert. Die ursprünglichen Guide-/Routendateien bleiben eigenständige Eingaben; neue Ergebnisse heißen ausdrücklich `timeline`.

## Was die Suche entscheidet

Das Ziel ist die kürzeste simulierte Zeit bis zum Ziellevel, ohne eine feste Grenze von 14 Stunden. Ereignisse werden sekundengenau vorwärts simuliert. Quests sind Kandidaten; nicht alle müssen ausgeführt werden. Mindestlevel, Vorquests, Questlog, Reise, Kampf, XP, Berufsstatus, Ausrüstungseffekte und Bufftimer wirken auf spätere Schritte.

- Mehrere unabhängige Startdateien und ein kompatibles gespeichertes Ergebnis konkurrieren miteinander.
- Für bis zu acht geeignete Dungeons werden sämtliche Teilmengen einschließlich der leeren Menge untersucht. Für Level 20 sind beim Untoten vier Dungeons vor dem Ziellevel möglich: RF, HDW, RoL und Todesminen. Das ergibt **16 Kombinationen**. Ein Dungeon mit Mindestlevel 20 beginnt nicht mehr vor dem Ziel Level 20.
- Eine Teilmenge wird an einer begrenzten Auswahl von Einfügepositionen geprüft. Das ist kein Optimalitätsbeweis. Bei kleinem Zeitbudget zeigt `DungeonEnumerationComplete=false`, dass die Prüfung nicht abgeschlossen wurde. Bei mehr als acht Dungeons wird eine begrenzte Auswahl verwendet.
- Questketten und größere Abschnitte können entfernt und neu zusammengesetzt werden. Annahmen und Abgaben eines Questhubs können gemeinsam verschoben werden. Verbesserungen erhöhen die Auswahlwahrscheinlichkeit einer Änderungsfamilie; Neustarts helfen bei Stillstand.
- Die bisherige Suche mit zeitweilig langsameren Zwischenlösungen bleibt erhalten. Der beste zulässige Kandidat wird dabei nie verworfen.
- Berufsvorbereitung, ihr frühester Zeitpunkt, opportunistische oder gezielte Trainerbesuche, Zauberstabnutzung und Schlafsackpausen sind getrennte, im Ergebnis gespeicherte Entscheidungen. `on`/`off`/`auto` begrenzen die Auswahl. Dieselbe gespeicherte Vorbereitung hat unter festen und freien Vorgaben dieselbe physische Zeitbewertung.
- Wiederholbare Quests können einmalig eingeplant werden; es gibt keine wiederholte XP-Schleife. Ein pauschaler Ausschluss höherer Questlevel trotz erreichbarem Mindestlevel wurde entfernt. Die tatsächlichen Voraussetzungen und Gegner entscheiden über die Machbarkeit.

## Korrigiertes Grinden

Der alte Grindbaustein tötete pauschal 30 Gegner. Dadurch konnte ein Zwischenlevel überschritten und der folgende Abschnitt übersprungen werden. Er berechnet jetzt die für den nächsten angeforderten Levelschritt benötigten Kills, höchstens 30 je Baustein. Rested-XP und der aktive XP-Bonus werden berücksichtigt. Die Vorwärtssimulation bleibt für den tatsächlichen Fortschritt maßgebend.

## Vorbereitung und Buffs

Der Schlafsack gibt keinen Bonus durch bloßen Besitz. Die Route braucht die abgeschlossene Beschaffung beziehungsweise bestätigten Live-Besitz und eine tatsächliche `sleep`-Aktion. Diese bezahlt das Aufstellen und 60 Sekunden je Stack. Cooldown und Ablaufdatum werden im Zustand geführt. Der Bonus wirkt auf Quest- und Kill-XP; bei einem Ablauf während eines Dungeonlaufs wird die betroffene Kill-XP anteilig berechnet. Für aggregierte Dungeon-XP wird eine gleichmäßige Verteilung über die modellierte Kampfzeit angenommen, da die tatsächliche Reihenfolge fehlt.

Das Profil verwendet vorläufig +1 % je Stack, höchstens drei Stacks, 1 Stunde Wiederbenutzung und 2 Stunden Buffdauer. Der Spell-Tooltip und der Guide widersprechen sich bei Dauer und Effektanzeige. Deshalb sind alle Werte konfigurierbar und ein Vergleich mit 1 Stunde ist enthalten. 10 Sekunden Aufstellen sind geschätzt. [Quellen und Annahmen](../sources/timeline-model-sources.json).

Pausen werden an bekannten Gasthäusern/Hauptstädten und bekannten Gegner-/Elitecamps abgelehnt. Diese Prüfung verwendet konservative Geometrie, keine vollständigen Gebietsflags. Ein bestätigtes Live-Ruhegebiet wird zusätzlich berücksichtigt.

Berufsmaterialien, Geld, Auktionshausversorgung, Berufsslotwechsel, Essen und überlappende Camping-/Schlafsackpausen sind noch kein vollständiges Ressourcenmodell. Die vorhandenen gebündelten Vorbereitungs- und Kampfkosten bleiben Schätzungen. Fehlende Questdaten bleiben gesperrt. Diese Arbeit macht aus dem Modell keine Ingame-Zeitmessung.

## Dauerhaftes bestes Ergebnis

Ein Checkpoint enthält Ereignisse, Vorbereitung, Zauberstabregel und die neu geprüfte Zeit. Sein Kontext umfasst Modellrevision, Datensatz, Navigation, vollständiges Kampfprofil, Charakter, Ziel, Pflichtquests, erlaubte Optionen, Dungeonvorgaben und gegebenenfalls bestätigte Rückmeldedaten.

Beim Laden wird die Route erneut simuliert; die gespeicherte Sekundenangabe wird nicht vertraut. Ein inkompatibler Kontext wird abgelehnt. Geschrieben wird über eine temporäre Datei und Umbenennen. Eine exklusive Dateisperre schützt auch konkurrierende Prozesse. Ein langsamerer Schreiber ersetzt keine schnellere gültige Route. Eine nicht freigegebene Sperre wird als Fehler gemeldet, nicht still gelöscht.

Auch `improve -optimize` verwendet nun standardmäßig einen Checkpoint neben der Ausgabedatei. Alle fehlgeschlagenen Optimierungsworker führen zu einer Fehlermeldung statt zu einem scheinbar erfolgreichen alten Ergebnis.

## Reproduktion

```sh
go test ./...
go vet ./...
go test -race ./route ./cmd/foreverroutes -run 'TestTimeline|TestCheckpoint|TestSleepingBag|TestPreparation|TestFeedback|TestGuideIntermediate|TestLiveDoesNot|TestRouteModel' -count=1

go run ./cmd/foreverroutes timeline \
  -race Undead -class Warlock -target 20 \
  -profile data/profiles/warlock.json \
  -seeds data/routes/undead-warlock.json,data/routes/undead-warlock-dungeons.json,data/audits/no-wc-seeds/dungeon_wand.json \
  -dungeons -camping auto -wands auto -sleeping-bag auto \
  -budget 3m -out data/optimized/undead-warlock.json
```

Ohne HDW: `-exclude-dungeons wailing-caverns`. Verbindliche Läufe: `-require-dungeons ragefire-chasm,ruins-of-lordaeron`. Ohne Dungeons: `-dungeons=false`. Feste Optionen: `-wands on` beziehungsweise `-sleeping-bag on`. Die Schlafsackvorgabe verlangt die Beschaffung; die Nutzung bleibt eine Zeitentscheidung. Für einen kontrollierten Zeitvergleich `-bag-duration 1h` und `2h` in getrennten Ausgaben/Checkpoints verwenden.

Weitere Rechenzeit auf demselben Kontext benutzt denselben Checkpoint automatisch. Bekannte gute Ergebnisse können ebenfalls als Seeds dienen: `data/audits/timeline-v2-winners/*.json`. Ein freier Suchlauf kann so gültige Gewinner fester Varianten übernehmen, anstatt ihre Verbesserung erneut entdecken zu müssen. Ein Checkpoint mit anderen festen Vorgaben wird nicht direkt wiederverwendet; sein Ergebnis kann als neu zu prüfender Seed dienen.

Ein festes Zeitbudget kann je Rechner zu anderen Routen führen. `-iterations` begrenzt zusätzlich die zufälligen Schritte. Gleicher Seed und Iterationsgrenze sind bei ausreichend großem Zeitbudget reproduzierbar.

## Geprüfte Ergebnisse: Untoter Hexer 1–20

Acht getrennte Integrationsläufe mit je 180 Sekunden Suchbudget, Seed 20261027 und GOMAXPROCS=1 haben Level 20 erreicht. Jeder Lauf hat alle tatsächlichen Aktionen nachgespielt, Pflicht-Klassenquests und gewählte Optionen geprüft und den Checkpoint erneut geladen. Alle 16 Dungeon-Teilmengen wurden jeweils untersucht. Diese Untersuchung beweist keine global optimale Reihenfolge oder Questwahl.

[Maschinenlesbare Ergebnisse](timeline-v2-validation.json), [wiederverwendbare Gewinner](timeline-v2-winners/).

| Vorgabe, 2h-Buffprofil | Beste Modellzeit |
|---|---:|
| Frei, nach Übernahme der gültigen bekannten Gewinner | **11:14:41** |
| Ohne Zauberstab und ohne Schlafsack | 11:28:26 |
| Mit Zauberstab, ohne Schlafsack | 11:14:41 |
| Ohne Zauberstab, mit Schlafsackbeschaffung | 12:50:51 |
| Mit Zauberstab und Schlafsackbeschaffung | 12:34:34 |

Die freie beste bekannte Route behält RF → HDW → RoL, benutzt einen Zauberstab und bereitet Camping später vor. Sie beschafft keinen Schlafsack. Der freie einzelne Suchlauf hatte zunächst 11:25:05 gefunden; die gültige bessere Zauberstabvariante wurde anschließend als Seed übernommen und als freier Checkpoint gespeichert. Das zeigt sowohl die Grenze einer kurzen Suche als auch den Nutzen eines gemeinsamen Bestands bekannter gültiger Lösungen.

Bei gleicher Route und Vorbereitung verlängert das 1h- statt 2h-Profil die getestete Zauberstab-/Schlafsackroute von 12:34:34 auf 12:37:28. Die beiden anderen geprüften Routen hatten keine Schlafpause und bleiben gleich. Separat neu optimierte 1h-Läufe haben andere Suchverläufe; ihre Zeiten sind kein isolierter Buffeffekt.

Die älteren Tabellen in `data/audits` gehören zum damaligen Modellstand. Nach der Grindkorrektur und neuen Vorbereitungsentscheidungen sind sie keine unveränderte Kontrollgruppe. Statische Routen für die gesamte Klassen-/Völkerliste wurden mit dieser Änderung nicht neu veröffentlicht.

Optionaler voller Integrationslauf je Szenario (`auto`, `no_wand`, `wand`, `bag`, `wand_bag`, `auto_1h`, `bag_1h`, `wand_bag_1h`):

```sh
FR_TIMELINE_SCENARIO=auto FR_TIMELINE_OUTPUT=/tmp/timeline-proof GOMAXPROCS=1 \
  go test ./route -run '^TestTimelineShippedBenchmark$' -count=1 -v -timeout=12m
# Nach allen Szenarien: bekannte gültige Gewinner übernehmen und identische Routen mit 1h/2h nachspielen.
FR_TIMELINE_POOL=/tmp/timeline-proof \
  go test ./route -run '^TestTimelinePoolAndDurationSensitivity$' -count=1 -v
```
