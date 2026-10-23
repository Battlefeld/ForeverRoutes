# Alle Klassen × vier Routen: Prüfstand vom 23.10.2026

**Das Gesamtziel ist noch nicht erfüllt.** Alle 36 Kombinationen laufen jetzt als eigene Klassenprofile. Mit klar gekennzeichneten, generierten Fortsetzungen erreichen 36/36 Level 30. Es gibt weiterhin 0 validierte Sieger.

## Tatsächlich ausgeführt

Die Tabelle zeigt die erreichte Stufe im Diagnosemodell, nicht eine validierte oder vollständig ausgeführte Originalroute. Originalquellen ohne generierte Ergänzungen erreichen in 18/36 Fällen Level 30. Die Quellabdeckung ist eine separate Eigenschaft: Beispielsweise kann eine nominale 1–20-Quelle durch Modell-XP bis 22 gelangen.

| Klasse | Referenzvolk | RestedXP original | Journey original | Wisp original | Polestar original | Mit Ergänzungen bis 30 |
|---|---|---:|---:|---:|---:|---:|
| Krieger | Orc | 22 | 30 | 20 | 30 | 4/4 |
| Paladin | Human | 18 | 30 | 22 | 30 | 4/4 |
| Jäger | Orc | 22 | 30 | 20 | 30 | 4/4 |
| Schurke | Orc | 22 | 30 | 20 | 30 | 4/4 |
| Priester | Troll | 22 | 30 | 20 | 30 | 4/4 |
| Schamane | Orc | 22 | 30 | 20 | 30 | 4/4 |
| Magier | Troll | 22 | 30 | 20 | 30 | 4/4 |
| Hexenmeister | Orc | 22 | 30 | 20 | 30 | 4/4 |
| Druide | Tauren | 22 | 30 | 20 | 30 | 4/4 |

Es gab in beiden Durchläufen keine Prozessfehler. Alle 72 Kostenbücher schließen auf Gesamt-, Reise- und Grindzeit. Die generierten Fortsetzungen enthalten nach der Korrektur keine fehlgeschlagenen Questaktionen. Die weiterhin offenen Aktionen in den Originalteilen bleiben in jedem Ergebnis erhalten.

## Änderungen

- Eigene, geprüfte Klassen-/Volkprofile für alle neun Klassen, einschließlich einer Allianz-Welt für Paladine. Alle vier Quellen eines Profils verwenden dieselben Welt- und Zeitparameter.
- Kapitelwahl aus dem vollständigen Katalog und den klassenspezifischen RestedXP-Folgeverweisen; Polestar wird für explizite Klassen ebenfalls mit seinen vollständigen Klassenbedingungen geladen.
- Klassen-/Questbedingungen werden vor Bewegungen ausgewertet. Inaktive Klassenabstecher erzeugen dadurch keine Reisezeit mehr.
- Polestar-Kapitelwahl berücksichtigt den Beginn des nächsten Kapitels, damit eine nominale Kapitelgrenze den Lauf nicht vorzeitig beendet.
- Getrenntes, geprüftes Kostenbuch pro Aktionsart und Quell-/Ergänzungsanteil. Es verändert keine Enginezeit.
- Die Ergänzungsplanung versucht keine Elternquest abzugeben, deren Kinderbelohnungen noch fehlen.
- Vollständige Matrix in CI vor dem weiterhin strengen Abnahmegate. Fehlerzellen werden nicht aus dem Vergleich entfernt.

## Was eine belastbare schnellste Route noch verhindert

RestedXP und Wisp fehlen originale Folgekapitel in den gelieferten Quellen. Journey und RestedXP enthalten darüber hinaus noch ungelöste dynamische Bedingungen und nicht ausgeführte Aktionen. Auch Polestar enthält noch Modelllücken. Generierte Kapitel ersetzen diesen Nachweis nicht.

Die Engine verwendet weiterhin pauschale Klassenfaktoren für Kampf-/Erholungszeiten sowie generische Annahmen für einzelne Wege und Dienste. Fähigkeiten, Ränge, Ausrüstung, Talente, Begleiter und Ressourcen sind noch nicht als validierte Klassenmechanik für 1–30 integriert. Deshalb werden keine dieser Diagnosezeiten als plausible gemessene /played-Zeit und keine Route als belastbarer Sieger ausgegeben. Selbst ein behaupteter Parameter-Fit kann das Gate nicht öffnen.

Das fachliche Gate meldet korrekt Exitcode 2; der rein technische 36/36-Zieltest meldet Exitcode 0. Diese Ergebnisse haben verschiedene Aussagen.

## Nachweise und Reproduktion

- 50 Tests bestanden, einschließlich eines vollständigen 36-Kombinationen-Laufs bis 30, Kostenbuchprüfung und Schutz gegen erfundene Sieger.
- `2026-10-23-class-matrix-verification.json`: beide Matrizen mit Zeiten ausdrücklich als Diagnosewerten, Fehlerzahlen und Herkunft.
- `2026-10-23-class-matrix-evidence.json.gz`: vollständige Vergleichsergebnisse einschließlich ungelöster Bedingungen und einzelner Questfehler.
- Befehle und Grenzen: `../benchmarks/2026-10-23-orc-warlock/class-matrix.md`.

Der ältere Ordnername ist aus Kompatibilitätsgründen erhalten. Der neue Runner ist nicht auf Orc-Hexenmeister beschränkt.
