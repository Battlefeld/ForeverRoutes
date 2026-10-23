# Getrennte Routenprüfung – 23.10.2026

Die Originalrouten von RestedXP, Journey, ForeverWisp und Polestar wurden nicht bearbeitet, zusammengeführt oder optimiert. Dieser Snapshot ergänzt ausschließlich Importer, gemeinsame Questfakten und einen Prüfer. Jeder Guide behält eine eigene, nach Quelle benannte ID, seine Schrittfolge, Bedingungen und optionalen Schritte. Fingerprints vor und nach jeder Prüfung kontrollieren, dass der Prüfer den eingelesenen Guide nicht verändert.

## Was funktioniert

- 134 unabhängig gespeicherte Guides: Journey 41, ForeverWisp 8, RestedXP 28, Polestar 57. Dies sind Kapitel und Varianten, **keine 134 vollständigen Routen von Level 1 bis 30**. Guides außerhalb des Profils oder Zielbereichs werden als `not_applicable` markiert.
- Quellformate bleiben getrennte Adapter. Verschachtelte Journey-Bedingungen, RestedXP-Zielindizes und Schrittsteuerung sowie Wisp-Klassen, optionale Schritte und Gruppenmetadaten werden im Eingabeformat aufbewahrt.
- Rasse, Klasse und Fraktion sind Parameter. Horde-Orc-Warlock und Allianz-Human-Mage wurden getrennt geprüft; sämtliche Guide-Fingerprints blieben dabei unverändert. Der mitgelieferte Welt-Ausschnitt deckt nur Horde ab; beim Allianz-Bericht steht deshalb `worldAuditAvailable: false`.
- 56 zuvor fehlende Zielmengen wurden aus ausdrücklich nummerierten RestedXP-Zielangaben ergänzt. Die Herkunft bleibt als `guide_reference` markiert. Diese Questfakten gelten für jede Route, die dieselbe Quest verwendet. Zwei widersprüchliche Angaben zu Quest 99130 bleiben gemeldet; bestehende Werte werden beibehalten.
- TBC/WotLK-Bedingungen werden auf Forever nicht als aktive Quests behandelt. Negative RestedXP-Abgabe-IDs bleiben als bedingte Abgaben erhalten. Unbekannte dynamische Bedingungen werden weder als falsch geraten noch bedingungslos ausgeführt.
- 22 gezielte Tests bestanden: 9 Python-Tests und 13 Node-Tests, einschließlich unterschiedlicher Quest-Reihenfolgen, unveränderter Eingaben, verschachtelter Versionsbedingungen und fehlender Ausführungsadapter.

## Was noch fehlt

Dies ist eine **Vorprüfung**, keine fertige universelle Zeitsimulation. Der bestehende Diagnose-Replay verarbeitet nur einen Teil der benötigten Aktionen. Erforderliche Adapter für Reise-/Ruhesteinaktionen, Inventar, Berufe, dynamische Bedingungen, Schleifen und einzelne Questziele sind noch offen. Fehlende Ausführung und Daten werden pro Guide als Blocker ausgegeben. Alle für die beiden Testprofile relevanten Guides sind derzeit blockiert. Es gibt keine veröffentlichte Zeitrangliste.

Die nächsten Implementierungen gehören in die gemeinsamen Ausführungsregeln beziehungsweise den jeweiligen Quelladapter. Die Questreihenfolge und Wege der Originalrouten bleiben dabei die Eingabe. Ein Testlauf darf Schritte nur nach den ursprünglichen Profil- und Laufzeitbedingungen überspringen. Fehlende Funktionen rechtfertigen keine Umplanung einer Route.

Der Wisp-Download enthält acht Starter-Guides und Verweise auf nicht mitgelieferte Fortsetzungen. Daraus lässt sich keine vollständige Level-1–30-Route rekonstruieren. Das ursprüngliche ZIP war am Ende abgeschnitten; die verwendeten Lua-Dateien wurden aus vollständigen lokalen ZIP-Einträgen mit geprüfter Länge und CRC32 gewonnen.

## Reproduzieren

Benötigt werden Python 3.10+ und Node.js 18+. Im Verzeichnis dieses Snapshots:

```sh
python3 -m unittest discover -s tests -p 'test_guide*.py' -v
python3 -m unittest discover -s tests -p 'test_source_guides.py' -v
node --test tests/test_route_contract.cjs
node scripts/check_routes.cjs --catalog events/catalog.json.gz --quests data/quests.json.gz --world data/world-horde.json.gz --facts data/objective-facts.json --race Orc --class Warlock --faction Horde --output reports/replayed-horde.json
node scripts/check_routes.cjs --catalog events/catalog.json.gz --quests data/quests.json.gz --world data/world-horde.json.gz --facts data/objective-facts.json --race Human --class Mage --faction Alliance --output reports/replayed-alliance.json
```

`--source journey|wisp|rxp|polestar` und `--guides ID,ID` begrenzen die Auswahl, ohne Guides zu verbinden. `--scenario profil.json` übernimmt zusätzliche Profilparameter. `--with-optional` aktiviert die bereits in der Quelle enthaltenen optionalen Schritte. Das komprimierte Welt-JSON enthält nur Fakten für die Vorprüfung und reicht nicht für eine Zeitsimulation.

Die erneute Erstellung des Katalogs aus Originaldateien benötigt zusätzlich die jeweiligen Addons und den festgeschriebenen Polestar-Checkout. Die Importer liegen bei; private Originaldateien müssen nicht veröffentlicht werden. `scripts/import_polestar_catalog.cjs` setzt wie der frühere Diagnose-Lauf `.comparison/data` und einen Polestar-Checkout neben dem Repository voraus. `POLESTAR_ROOT` überschreibt dessen Pfad.

## Herkunft

- Voriger Diagnose-Snapshot: `../2026-10-23-orc-warlock/`, Commit `d90c952251f1be5ef63dec6b0ddf23f46f21b806`; unverändert erhalten.
- Polestar: `eb7b47840f032bf711a3c8d7b1df9f3f6551b990`, GPL-3.0; dessen Quellcode wurde nicht verändert.
- RestedXP: `9536b5a9ee7e55745618a57dad43df2c4312ea16`; Originaldateien unverändert.
- QuestieDB Forever: `1f538335cfbb240c14a97c340f48c555fdc8a52b`, ergänzt um die vorhandenen Forever-/WarcraftDB-Fakten und die bereits freigegebene Kennzeichnung von 160 unbekannten XP-Belohnungen als vorläufig 0. Diese sind keine verifizierten Nullbelohnungen.
- Journey `4.00.02`; ForeverWisp `0.32.20-beta.1`. Eingabe-Fingerprints stehen im Manifest.

`events/catalog.json.gz` enthält die eingelesenen Guide-Fakten, `data/` die gemeinsame Faktenbasis, `reports/` die getrennten Prüfberichte und `validation.json` die kompakte Validierung. Diese Daten sind kein neues gemeinsames Routenskript.
