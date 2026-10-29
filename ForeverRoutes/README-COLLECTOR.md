# Questdaten aus dem Forever-Client

Der Collector gehört ausschließlich zu ForeverRoutes. Er sammelt nach dem Login
Titel und verfügbare Ziele aus dem Questlog und fragt bekannte Quest-IDs der
mitgelieferten Routen gedrosselt über `C_QuestLog.RequestLoadQuestByID` ab. Pro
Sitzung wird jede ID einmal angefragt; bei Bedarf `/frcapture 971` erneut verwenden.
Timeouts und negative Antworten werden nie als entfernte Quests gespeichert.
Der Collector läuft ausschließlich auf 1.60.x und trennt Build, Region und Sprache.

`/frcapture` zeigt den Bestand. Beim Ausloggen oder `/reload` schreibt WoW den
JSON-Export in `WTF/Account/<Account>/SavedVariables/ForeverRoutes.lua`. Es werden
keine Account-, Charakter- oder Chatdaten gesammelt und nichts hochgeladen.

Im ForeverRoutes-Checkout:

```sh
python tools/import_client_quests.py '/Pfad/zu/SavedVariables/ForeverRoutes.lua'
# Optional: Änderungen der Datei alle 60 Sekunden übernehmen.
python tools/import_client_quests.py '/Pfad/zu/SavedVariables/ForeverRoutes.lua' --watch
```

Der Import liest Daten, führt niemals Lua aus und schreibt atomar nach
`data/sources/client-quests.json`. Er ist wiederholbar und behält getrennte
Build-/Regions-/Sprachstände. Nach Aufnahme dieser Quelldatei in den Checkout
verwenden `build-data` und die bestehenden Update-Läufe sie automatisch. Der
lokale Watcher überträgt keine Dateien selbstständig zu GitHub.

Der Routen-Datensatz übernimmt derzeit nur bestätigte englische Questtitel für
bereits bekannte IDs (neuester Build, bevorzugt EU, dann enUS). Sonstige Felder
bleiben als Beobachtungen erhalten. Erhaltene XP/Geld werden mit Spielerlevel und
Zeitpunkt gespeichert, niemals als Basisbelohnung. UI-Kartenpositionen werden nicht
als Weltkoordinaten interpretiert. Zieltexte ohne Ziel-ID ersetzen keine
Routenziele. Neue Quest-IDs bleiben im Rohbestand, bis ihre Routenanforderungen
geklärt sind. Questgeber, Voraussetzungen und Drop-Raten sind noch nicht vollständig
aus dem Client erfasst. Kein Anspruch auf vollständige serverseitige Questdaten.

Referenz: Forever-Branch von Gethe/wow-ui-source,
`Blizzard_APIDocumentationGenerated/QuestLogDocumentation.lua`.
