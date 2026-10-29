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

Mit ForeverCore ab `0.6.0-autodata1` werden neue Questtitel bei einer freien
Verbindung automatisch und ohne `/reload` importiert. Gespeicherte Beobachtungen
werden beim Schreiben der SavedVariables ebenfalls automatisch eingelesen.
Die lokale Datenbank bleibt außerhalb der heruntergeladenen Routenpakete erhalten.
Der Live-Planer liest unterstützte Titel automatisch über `next -client-quests`;
ob ein Paket diese Funktion unterstützt, steht in seinem Bundle-Manifest
(`features: ["client-quests-v1"]`). Alte Planer erhalten den neuen Parameter nicht.
Ein ausstehender Datenabgleich erscheint im HUD-Status und im Kristall-Mouseover.

Optional für Entwicklung ohne laufenden Companion, im ForeverRoutes-Checkout:

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
