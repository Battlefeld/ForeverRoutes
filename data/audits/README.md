# Questprüfung bis Mindestlevel 20

Die aktuelle Code-, Simulations- und Routenprüfung aller 56 Kombinationen mit neu berechneten Ausgaben steht in [roster-v22.md](roster-v22.md). Die Quest-Einzelprüfung und ihre dokumentierten Datenlücken unten bleiben davon getrennt.

Geprüft wird der gesamte ausgelieferte Datensatz mit Mindestlevel ≤20: **1908 Quests**, einschließlich höherstufiger Quests, die schon früher angenommen werden dürfen. Jede Quest steht mit Quellen, Ausschlussgründen und Unsicherheiten in [quests-through-20.json](quests-through-20.json). Der Check baut die 56 ausgelieferten Volk-/Klassenkontexte auf, einschließlich beider Skyborne-Fraktionen und Dungeonbetrieb.

Dies ist eine Quellen- und Modellprüfung. Eine vollständige Prüfung jeder Quest im laufenden Spiel und eine Kalibrierung ihrer Zeiten stehen aus.

## Ergebnis

| Status | Quests | Bedeutung |
|---|---:|---|
| Modellierbar bis 20 | 1431 | Mindestens ein passender Kontext kann Ziele und Vorquests bis 20 lösen |
| Wiederholbar, einmalig modelliert | 19 | Kein wiederholtes XP-Farmen; Folgereihen bleiben zugänglich |
| Bis 20 nicht machbar | 152 | Ziele oder Voraussetzungen sind im gewählten Modell bis 20 nicht erfüllbar |
| Außerhalb des Levelprofils | 127 | 54 Feiertags-/Eventquests, 66 andere Berufquests, 7 PvP-/Raidquests |
| Datenlücke oder nicht auflösbar | 179 | 46 explizite Datenlücken und 133 fehlende Positionen/Quellen oder blockierte Vorquestketten |

Die [offene Liste](open-quests-through-20.md) nennt alle 179 ungelösten Fälle. Diese sind gesperrt. 68 Quests haben weiterhin unbekannte XP und werden konservativ mit 0 bewertet. Bei 72 Quests ist die Ereignisdauer nur ein Interaktionsansatz; Eskorte, Skriptwartezeit oder zusätzliche Kämpfe müssen im Spiel gemessen werden. Auch modellierbar bedeutet daher keine garantierte Echtzeit.

## Korrekturen

Gegenüber `dd2c371f03d44b2d1bcdda7311219137c808430d` ändern sich 1128 Quests bis Mindestlevel 20 einschließlich ihrer Herkunftsangaben; **300 Quests ändern sich inhaltlich**. Die [Änderungsübersicht](changes-through-20.json) zählt Felder und Quest-IDs. Unter anderem ändern sich Ziele bei 211 Quests, Startorte bei 84, Abgabeorte bei 89 und XP bei 2 Quests. Diese Mengen überschneiden sich.

- Clientziele ersetzen veraltete Referenzziele; im Spiel beobachtete Werte behalten Vorrang. Fehlende Mengen werden nicht pauschal auf 1 gesetzt. Aus einer passenden, belegten Referenz darf die Menge übernommen werden.
- Gesprächs-, Heilungs-, Emote- und Benutzungsziele werden über die tatsächliche Aktion und das passende NPC-Ziel erkannt. Keine Kill-XP, Beute oder Respawnkosten für Gespräche. Für mehrere Interaktionen werden unterschiedliche Ziele benötigt.
- Neeru Fireblade in `5727` ist ein Gespräch. Lazy Peons, Priest-/Paladin-Interaktionen, die Gespräche mit Bewohnern und Fraktionsführern sowie fünf Deathguards in Discipline sind entsprechend modelliert.
- Bartleby und die drei Novice Warriors von Sparring Match sind Duelle mit Kampfzeit, ohne gewöhnliche Kill-XP oder Beute. Body and Heart erfordert Lunaclaw und das anschließende Gespräch mit seinem Geist am richtigen Ort für beide Völker.
- Fehlende Itemquellen gelten nicht mehr aufgrund verschiedener NPCs als kostenlos mitgegebene Liefergegenstände.
- Eine fehlende Erwähnung im Guide entfernt keine Vorquest. Alternative Voraussetzungen werden nicht einzeln als vermeintlich unnötig gelöscht. Der Optimierer bewertet neue Quests zusammen mit ihren Vorquests; für enthaltene Dungeons werden externe Questketten gezielt ausprobiert.
- Ein Dungeonlauf heilt keine unbekannten Mengen oder fehlenden Quellen außerhalb der Instanz. Der garantierte Startgegenstand aus einer Instanz kann eine Quest dort freischalten, etwa The Unsent Letter.
- WarcraftDB ergänzt nun auch reine Clientquests. 111 frisch abgerufene Questdatensätze ergänzen die vorhandene Quelldatei. Positionen verwenden vorrangig die tatsächlichen Client-Kartenrahmen.
- Abgelehnte Aktionen verändern die Simulationszeit und den Campingzustand nicht mehr.

## Nachweis und Reproduktion

```sh
go test ./...
go vet ./...
go test -race ./route -run 'TestInteraction|TestDuel|TestRejected|TestUnknownDungeon|TestRepeatablePrerequisite|TestShippedQuestAudit|TestChain' -count=1
go run ./cmd/foreverroutes check-quests -target 20 -out data/audits/quests-through-20.json
go run ./cmd/foreverroutes build-data -cache /tmp/foreverroutes-cache -out rebuilt.json.gz
```

Der erneute Aufbau aus denselben gepinnten Quellen ist JSON- und bytegleich zum ausgelieferten Gzip-Datensatz (SHA-256 `77bc65167ba44d86e0b55f918ef56a49ffd0011d390e8de5fa47654514687db8`).

Die Tests decken unter anderem Neerus echte Gesprächsaktion und das Freischalten von `5728`, die vollständige Optimierungskette `5726→5727→5728`, Duelle ohne Kill-XP, getrennte Interaktionsziele, die Lunaclaw-Varianten, einmalige wiederholbare Vorquests und gesperrte unbekannte Dungeonziele ab. Die vorhandenen vollständigen Planungstests für Orc-Hexer und Menschen-Magier bis 30 bestehen ebenfalls.

## Untoter Hexer, 1–20

Die acht angefragten Varianten wurden mit dem korrigierten Datensatz erneut optimiert und sämtliche tatsächlichen Aktionen nachgespielt. Alle erreichen 20 mit den Pflicht-Klassenquests und den gewählten Optionen. [Messwerte und Prüfflags](undead-warlock-through-20.json).

| Variante | Modellzeit |
|---|---:|
| Solo | 13:52:33 |
| Solo + Zauberstab | 13:42:08 |
| Solo + Schlafsack | 15:37:47 |
| Solo + Zauberstab + Schlafsack | 14:41:04 |
| Mit Dungeons | 11:55:21 |
| Mit Dungeons + Zauberstab | 11:36:33 |
| Mit Dungeons + Schlafsack | 13:38:23 |
| Mit Dungeons + Zauberstab + Schlafsack | 12:45:04 |

Alle vier Dungeonvarianten laufen RF → WC → RoL und schließen jetzt **5726, 5727 und 5728** ab. Power to Destroy wird in drei Varianten behalten; Leaders of the Fang wurde in diesen Suchläufen nicht behalten. Das ist ein Suchergebnis, kein Beweis für das globale Zeitoptimum. Die Quest ist für passende Kontexte modellierbar.

Der Schlafsack beinhaltet die Beschaffung; sein späterer XP-Bonus ist noch nicht modelliert. Das Kampfprofil ist das vorhandene Warlock-Profil mit Starterwaffe und ohne Talente; außerhalb seiner Tabelle wird extrapoliert. Lauf- und Gruppensuchzeiten sind Schätzungen. Die bisherigen statischen Routendateien wurden nicht für alle Völker und Klassen neu erzeugt; ihre gespeicherten Stunden müssen neu berechnet werden.

Optional reproduzierbarer Testlauf je Variante (`solo`, `solo_wand`, `solo_bag`, `solo_wand_bag`, `dungeon`, `dungeon_wand`, `dungeon_bag`, `dungeon_wand_bag`):

```sh
FR_BENCH_SCENARIO=dungeon_wand FR_BENCH_OUTPUT=/tmp/quest-audit-runs GOMAXPROCS=1 \
  go test ./route -run '^TestUndeadQuestAuditBenchmark$' -v -count=1 -timeout=10m
```

Je Startseed 30 Sekunden Verbesserung, dann 180 Sekunden Timelinesuche mit Seed 20261027. Ein Zeitbudget kann je Rechner zu anderen besten Routen führen. Die Ergebnisdatei enthält den SHA-256 des verwendeten Datensatzes.

## Direkter Vergleich ohne HDW

[Ergebnisse und Reproduktion für RF + RoL mit oder ohne HDW](without-wailing-caverns.md). Beide Seiten wurden zusätzlich mit gleichem Zeitbudget optimiert.

## Neuer Optimiererstand

[Zeitstrahl-Optimierung und neue Validierung](timeline-v2.md). Die bisherigen Zeiten auf dieser Seite bleiben historische Ergebnisse ihres damaligen Modellstands.
