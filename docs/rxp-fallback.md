# Normaler RXP-Fallback 1–20

Der Fallback bleibt unabhängig von optimierten Routen. Das Paket `data/fallbacks/rxp.json.gz` enthält 30 normale Originalkapitel aus der in `data/sources/lock.json` festgeschriebenen RXP-Revision `48498a23d47f3b668c2c7f6d234429978c3cab86`. Quelle: https://github.com/RestedXP/RXPGuides . Lizenz: [CC BY-NC-SA 4.0](../data/fallbacks/RXP-LICENSE.txt).

Für jede der 56 unterstützten Volk-/Klassenkombinationen enthält es eine eigene Kapitelkette; Skyborne Allianz und Horde sind getrennt. Die Quellen sind die normalen Solo-Guides, einschließlich normaler Magier, ohne AoE-Ersatz. Die Kapiteltexte bleiben unverändert, einschließlich Orts-, Lehrer-, Gegenstands-, Geld-, Reise- und dynamischer Questanweisungen. Quellpfad, Zeile, Revision und Dateihashes stehen im Katalog. Unbekannte Bedingungen beziehungsweise Schreibfehler der Quelle werden für die zusätzliche Projektion konservativ ausgelassen und als Warnung geführt; die Originaltexte werden nicht umgeschrieben.

`next -route auto` lädt die passende aktuelle optimierte Route. Eine fehlende, veraltete, beschädigte oder falsch zugeordnete Route führt zum normalen RXP-Fallback. Eine ausdrücklich angegebene falsche Charakterroute wird mit Fehler abgelehnt. `-route ""` bleibt die freie Planung. Die Spieloberfläche wird erst im nächsten Arbeitsschritt erstellt.

Der CLI-Fallback und die Vergleichsläufe verwenden eine gekennzeichnete **Quest-Arbeitsprojektion**. Sie behalten Klassen-/Volksfilter und die gewählte Kapitelreihenfolge, bilden aber nicht die gesamte RXP-Laufzeit ab. Dynamische Guards, Geld-/Inventarbedingungen, genaue Trainerwege, ausdrücklich angeordnete Reisen und Parallelziele sind nicht vollständig als RXP-Interpreter implementiert. Datenlücken und ausgelassene Quests stehen in den Prüfberichten. Eine solche Modellzeit wird nicht als exakte Zeit des Originalguides ausgegeben.

`export-addon` liefert für das Ingame-UI:

- `ForeverRoutesData.routes`: nachgespielte optimierte Schritte;
- `ForeverRoutesData.rxpChapters`: unveränderte RXP-Kapitel mit Herkunft;
- `ForeverRoutesData.fallbacks`: Charakter-/Fraktionsketten bis 20;
- `ForeverRoutesData.SelectRoute(race, class, faction, dungeons, forceFallback)`: optimierte Route oder normaler RXP-Fallback. Fehlende Skyborne-Fraktion wird abgelehnt.

Das UI muss die RXP-Anweisungen einschließlich ihrer dynamischen Bedingungen auswerten; eine Questprojektion ersetzt diesen Interpreter nicht. Der Export enthält Daten und Auswahl, noch keine fertige Spieloberfläche. Releases enthalten das Katalogpaket, die Lizenz und die generierte Datei `addon/Routes.lua` im ForeverCore-Bundle.

Reproduzieren beziehungsweise prüfen:

```sh
python3 tools/import_rxp.py /path/to/pinned/RXPGuides --license /path/to/RXPGuides/LICENSE
python3 tools/import_rxp.py /path/to/pinned/RXPGuides --check
go run ./cmd/foreverroutes export-addon -out /tmp/Routes.lua
```

Originalinputs früherer optimierter Routen bleiben getrennt in `data/audits/roster-inputs.json.gz`. Die Optimierung darf ihre eigene Kopie ändern, nie den originalen Fallback.
