# Hexenmeister-Kampfanbindung ohne Änderung an ForeverSim

Stand: 24.10.2026. Die Anbindung ist implementiert und als Diagnosemodus getestet. **Eine validierte Bestzeit von Level 1–30 ist noch nicht erreicht.**

Alle Änderungen liegen in ForeverRoutes. Vor und nach der Arbeit waren die drei ForeverSim-Branchköpfe (`main`, `claude/focused-lovelace-hqyfx4`, `artifacts`) identisch. Kein Commit, keine Branchänderung und keine Dateiänderung wurde an ForeverSim vorgenommen. Die Kopien der fünf eingebundenen Quelldateien entsprechen ihren ursprünglichen Git-Blobs.

## Ergebnis

Der neue Routenadapter verwendet die vorhandene Lua-Referenzroutine für Kampfschaden. In einem separaten Prozess ergänzt er Ressourcenweitergabe und Kostenbuchführung. Mana und Leben werden nicht zwischen Gegnern aufgefüllt. Begleitermana, Abklingzeiten und globale Abklingzeit bleiben erhalten. Erholung kostet ihre explizite Dauer; Tod und nicht getötete Gegner werden als bezahlte Fehlschläge behandelt.

Bei identischen frischen Eingaben stimmen die vollständigen Kampfergebnisse mit der ursprünglichen Lua-Routine überein: geprüft auf Level 10, 15, 20, 25 und 30 mit jeweils drei Seeds. Das ist eine Softwaregleichheitsprüfung mit synthetischen Werten, keine Messung echter Spielzeit. Eine Gleichheit mit der produktiven MythicSim-EXE wird nicht behauptet.

Der optionale Routenmodus ersetzt Quest-Kampfkosten direkt durch diese Ergebnisse. Fehlende Charakter-/Gegnerwerte und nicht belegte Grindorte halten ihn an. Der Anforderungsmodus sammelt aus allen vier Hexenmeister-Routen 246 unterschiedliche Gegner-IDs einschließlich der gekennzeichneten generierten Fortsetzungen. Die dabei erreichte Stufe 30 stammt aus dem bisherigen Diagnosemodell und ist keine 1–30-Bestätigung des neuen Kampfadapters.

## Prüfung

65 Tests bestanden. Darunter: alle bisherigen 36 Klassen-/Routenkombinationen; unveränderte Quell-Hashes; Ressourcenweitergabe zwischen und innerhalb separater Prozesse; Begleitermana; Abklingzeiten; bezahlte Erholung und Fehlschläge; echte Einbindung in Questarbeit ohne Rückgriff auf die alte Killzeittabelle; unveränderte Ergebnisse beim reinen Sammeln von Datenanforderungen.

## Noch offen

Die gepinnte Schnittstelle blockiert Level 1–4 und nähert 5–9 teilweise an. Für einen belastbaren Vergleich fehlen außerdem echte Charakter-/Gegner-/Erholungsdaten, die Validierung gegen aufgezeichnete Levelruns, die produktive Engine-Gegenprüfung sowie verbleibende Mechaniken und Routenaktionen. Begleiterleben/Bedrohung, Waffentimer und Verbrauchsvorräte sind noch nicht vollständig abgebildet. Daher bleiben Routensieger und validierte Zeiten gesperrt.

Reproduktion und Eingabevertrag: `../benchmarks/2026-10-23-orc-warlock/combat-adapter.md`. Benötigte Gegner-/Levelprofile: `2026-10-24-warlock-combat-requests.json.gz`.
