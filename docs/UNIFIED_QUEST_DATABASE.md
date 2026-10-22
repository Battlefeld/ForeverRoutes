# ForeverRoutes: Gemeinsame Questdatenbank Level 1–30

## Veröffentlichte Datensätze (separater Branch artifacts)

- catalog/level-1-30.json — gepinnte QuestieDB-Forever- und Classic-Referenzen für Quests, NPCs, HP, Gegenstände, Objekte und XP.
- foreverdb/quests-1-30.json — ForeverDB.net API-Questliste, Buildversion und über mehrere Läufe inkrementell ergänzte Einzelquest-Details.
- sources/everythingquests-forever.json — mit Git-SHA gepinnte EverythingQuests-Forever-Start-, Abgabe-, Ziel- und Questketten-Referenzen.
- unified/quests-1-30.json — alle Fakten feldweise vereint, inklusive Quellenname, Referenzqualität und Quellenlink.
- unified/quality-audit-1-30.json — fehlende Felder, Quellenkonflikte und laufende Qualitätsstatistik.
- unified/guide-coverage-1-30.json — Abdeckung pro vorhandener Guide-Quest-ID-Liste, kein künstlicher Geschwindigkeitsvergleich.

## Quellenpriorität

1. Tatsächlich in Forever beobachtete oder gemessene Werte.
2. Als Forever identifizierte Clientwerte.
3. QuestieDB-Forever-Strukturen sowie auswertbare Quest-Sammel-Addons.
4. Klassische HP, XP, Loot- und Questdaten als **vorläufig verwendbare Classic-Referenzwerte**.
5. Fehlende Daten bleiben unbekannt. Es werden keine Stückzahlen, Laufzeiten oder Dropchancen frei erfunden.

Die originale Classic-Quelle bleibt in jedem entsprechenden Datensatz sichtbar. Dokumentierte widersprechende Quellen werden im Qualitätsbericht als Konflikt protokolliert. ForeverDB wird mit dem erforderlichen Hinweis Data from https://foreverdb.net gekennzeichnet.

## Automatische Updates

Der GitHub-Workflow .github/workflows/unified-quests.yml läuft zweimal täglich (02:07 und 14:07 UTC) und kann manuell mit dem Eingabefeld refresh_sources gestartet werden. Er liest:

- den aktuellen QuestieDB-master-Commit und alle benötigten Quelltabellen;
- die ForeverDB API unter Wahrung des GitHub-Secrets FOREVERDB_API_KEY, 2,1 Sekunden Mindestabstand und max. 380 Netzwerkabfragen pro Aktualisierung. Bis zu 320 zuvor fehlende Questdetails werden pro regulärem Lauf ergänzt. Bereits archivierte Questdetails bleiben für spätere Läufe erhalten;
- den aktuellen EverythingQuests-main-Commit und dessen deklarative Forever-Daten, ohne Drittanbieter-Lua auszuführen.

Der Workflow startet mit Unit-Tests, prüft Schema, Questreferenzumfang und NPC-HP und speichert bei Erfolg alle Datensätze im eigenständigen artifacts-Branch. Ein fehlerhafter Lauf darf den letzten veröffentlichten, nutzbaren Datenstand nicht überschreiben.

## Erschlossene und weitere Vergleichsquellen

- https://github.com/Questie/QuestieDB — aktiver Hauptimport.
- https://foreverdb.net/developers — aktiver, durch Secret authentifizierter API-Import.
- https://github.com/wheelbarrel00/EverythingQuests — aktiver Commit-basierter Zusatzimport; enthält eigene und AllTheThings-basierte Daten.
- https://github.com/ATTWoWAddon/AllTheThings — umfangreiche Forever-Questketten; teils indirekt über EverythingQuests abgedeckt, nicht als unabhängige Beobachtung werten.
- https://github.com/TylerAkins/forever-quest-markers — AllTheThings-basierte Pins, daher teils redundant.
- https://www.curseforge.com/wow/addons/questdb — zusätzlicher LibQuestDB-Quest- und Zieltext-Bestand; noch kein integrierter stabiler Bulk-Importer.
- https://foreverdiff.gg/ — Clientdaten und Spielerbeobachtungen; Vergleichsquelle, kein hier zugesicherter Bulk-API-Zugang.
- https://www.wowhead.com/forever — nutzbar für belegbare manuelle Feldkorrekturen, nicht als pauschale ungeprüfte Web-Scraping-Quelle.

## Was zur Simulation noch fehlt

Einige Quests haben keine exakte Zahl für Kills oder gesammelte Gegenstände; die QuestieDB-Ziel-ID allein enthält keine Mengenangabe. Für genaue /played-Zeiten fehlen weiterhin echte begehbare Wege (auch Höhlen), Reisen/Flightmaster, Ruhestein/Deathskips, Respawn, Gruppeneffekte, Ausrüstung und Klassenkampfprofile.

Der Guide-Abgleich basiert bislang auf bekannten Quest-IDs der Leveling-Addons. Er misst Abdeckung und Auffälligkeiten, aber er erfindet keine Guideschritte oder objektiven Bestzeiten, solange diese Informationen nicht nachprüfbar vorliegen.

## Lizenz und Nutzungsrechte

Die originale ForeverRoutes-Implementierung ist MIT. Quelltabellen von Drittanbietern haben eigene Urheber- und Datenlizenzbestimmungen. Externe Daten dürfen nicht automatisch unter ForeverRoutes MIT weiterlizenziert oder ohne erforderliche Attribution veröffentlicht werden. Alles bleibt vorerst private Arbeitsreferenz.