# Routenmethode für Level 20–40

Stand: 7. November 2026. Der Elite-/Quelldatenfix v2.5 ist veröffentlicht und lokal installiert. Dieser Vergleich und der optionale `extend -portfolio`-Prototyp sind Vorbereitung; die veröffentlichten 112 Routen enden weiterhin bei 20. Alle Zeiten unten sind Modellzeiten, keine Ingame-Messungen oder bewiesenen Optima.

## Was die anderen tatsächlich entscheiden

Untersucht wurden öffentliche Enginequellen, keine kostenpflichtigen Guidepakete. Keine fremden Guideanweisungen wurden in diesen Prototyp kopiert. Die Aussagen betreffen die genannten Ausführungspfade, nicht jede Funktion der jeweiligen Projekte.

| Projekt / eingefrorener Stand | Beobachteter Mechanismus | Nutzen für uns |
| --- | --- | --- |
| [RestedXP 8c13da7](https://github.com/RestedXP/RXPGuides/blob/8c13da7b70ffebc9bbea61129c34e6a716906df7/GuideWindow.lua#L698-L806) | `GuideWindow.lua` 698–806 verarbeitet `completewith`, Sticky-Schritte und nummerierte Guidefortsetzung. `functions.lua` 4329ff/4427ff prüft Questfortschritt; 7702ff markiert Gruppen-/Solo-Schritte. | Vorbereitete Rundläufe mit optionalen Nebenaufgaben; Gruppenbedingungen als harte Zulassung, nicht als kleine Zeitstrafe. |
| [Guidelime 683ef72](https://github.com/max-ri/Guidelime/blob/683ef722093a7fb1ba75fe8ee188554dea400b82/Guidelime/CurrentGuide.lua#L967-L1019) | `Guidelime/CurrentGuide.lua` 967–1019 prüft Annahme-/Abgabe-Verfügbarkeit und aktiviert Schritte anhand vorheriger unerledigter Pflichtschritte. Das README beschreibt separat geladene, verfasste Guides. | Erledigte Quests überspringen, ohne notwendige Vorquesten oder noch nicht angenommene Ziele zu erfinden; optionale Aufgaben dürfen eine Hauptetappe begleiten. |
| [WoW-Pro 177a6c4](https://github.com/Ludovicus-Maior/WoW-Pro-Guides/blob/177a6c4e81a55af57dca111a8f0b20b87f305dd6/WoWPro/WoWPro_Parser.lua#L462-L588) | `WoWPro/WoWPro_Parser.lua` definiert `PRE`, `LEAD`, `S`/`US`, `RANK`; `WoWPro_Broker.lua` 1904ff wählt den nächsten Guideschritt und 3543ff überspringt Schritte oberhalb des konfigurierten Rangs. | Klare Pflicht-/Optional-Ziele und getrennte Varianten für Vollständigkeit. `RANK` ist hier eine Guideeinstellung, kein NPC-Eliterang. |

In diesen Pfaden wird überwiegend eine vorgegebene Reihenfolge unter Bedingungen ausgeführt. Daraus folgt keine gemessene globale Routenoptimalität. Die Qualität der verfassten Rundläufe kommt hinzu; die Engine allein erzeugt sie nicht. Das ist eine Schlussfolgerung aus den untersuchten Pfaden, keine Aussage über interne Werkzeuge der Guideautoren.

Ergänzende Fundstellen: [RXP Gruppen-/Solomarkierungen](https://github.com/RestedXP/RXPGuides/blob/8c13da7b70ffebc9bbea61129c34e6a716906df7/functions.lua#L7702-L7735), [RXP Questbedingungen](https://github.com/RestedXP/RXPGuides/blob/8c13da7b70ffebc9bbea61129c34e6a716906df7/functions.lua#L4329-L4493), [WoW-Pro Rangfilter](https://github.com/Ludovicus-Maior/WoW-Pro-Guides/blob/177a6c4e81a55af57dca111a8f0b20b87f305dd6/WoWPro/WoWPro_Broker.lua#L3543-L3559), [Guidelime Guidekonzept](https://github.com/max-ri/Guidelime/blob/683ef722093a7fb1ba75fe8ee188554dea400b82/README.md).

## Abgleich mit unserer gesamten Routenpipeline

| Bereich | Bereits vorhanden | Konsequenz für 20–40 |
| --- | --- | --- |
| Import / Auflösung | `internal/build`, gepinnte Quellen, RXP-Import, Quellen- und Eliteprüfung | Guidekoordinate allein bleibt kein Beleg für Bodenloot; alte Classic-Quests nicht ungeprüft auf Forever übertragen. |
| Welt / Zustand | `route/world.go`, `state.go`, `danger.go`, Questlogkapazität, Vorquesten, Fortschritt, max. Gegnerlevel | Dieselben harten Regeln für jede Alternative verwenden. |
| Kandidaten | `plan.go`: einzelne Quests, gemeinsame Gegner, Hubs; `chains.go`: vollständige Ketten | Nicht erneut einen simplen Nächste-Quest-Entscheider bauen. |
| Langfristige Auswahl | `live_stages.go`: Zonenetappen mit mehreren Abzweigungen und vier Stunden Horizont | Bereits vorhandene Etappenlogik nutzen; Rundläufe als zusätzliche Kandidaten, nicht als erzwungene Reihenfolge behandeln. |
| Optimierung | `optimize.go`, `portfolio.go`: unabhängige Seeds, Dungeonvarianten, Vorbereitung, große Änderungen, verifizierter bisheriger Gewinner | Fremde Guideprojektionen und eigene Rundläufe getrennt bewerten; Herkunft und unveränderte Referenz behalten. |
| Liveanpassung | `live.go`, `live_window.go`, `live_block.go`; ForeverCore `routenext.go` und Addon-Anbindung | Wirkliche Position, erledigte Quests und Teilfortschritte gelten vor einem idealisierten Guidezustand. |
| Erweiterung | `cmd/foreverroutes/extend.go` spielt die 1–20-Route nach und plant ab deren Endzustand | Hier fehlte Vielfalt: verschiedene Seeds führten bei `PlanAhead` zur gleichen Auswahl, da es `pick(..., run=0, ...)` verwendet. |

Der neue Prototyp greift am letzten Punkt an. Er verändert die Liveauswahl und die ausgelieferten Routen nicht.

## Konkrete Methode

1. Die tatsächlich verwendete 1–20-Route vollständig nachspielen. Endposition, XP-Überhang, erledigte/aktive Quests, Flugpunkte und Bindung übernehmen. Bei einem Livecharakter stattdessen `StateFrom` verwenden. Ein leerer Level-20-Charakter ist kein vergleichbarer Startzustand.
2. Mehrere vollständige Fortsetzungen aus genau diesem Zustand berechnen: bisherige Suche (6 Kandidaten / 3 weitere Entscheidungen), kurz (6/1), breit (12/3), tief (6/6). Jede Variante endet am gleichen Ziellevel, nicht nach einer unterschiedlich ergiebigen Stunde.
3. Aktionen vollständig mit `Apply` nachspielen lassen. Nicht erreichte Ziellevel, verlorene erforderliche Klassenquests und illegale Aktionen verwerfen. Bei gleicher Zeit die frühere Variante behalten. Die bisherige gültige Variante bleibt damit erhalten, wenn Zusatzsuche nichts verbessert.
4. Für die Produktionsrunde zusätzlich **lizenzierte und auf Forever geprüfte Rundläufe** als unabhängige Seeds in den vorhandenen Optimierer einspeisen. Rundlaufdaten benötigen Quest-IDs, Voraussetzungen, Fraktion/Klasse, Level-/Gegnergrenzen, Ein-/Ausstieg, Pflicht-/Optional-Ziele und Herkunft. Nur noch offene Ziele übernehmen; keine bereits erledigten Questbelohnungen nochmals zählen. Dieser Import neuer 20–40-Rundläufe ist noch nicht implementiert.
5. Auf Etappen 20–25, 25–30, 30–35 und 35–40 diagnostizieren, aber den Gewinner zusätzlich bis 40 vergleichen: Eine Abgabe oder Flugpunktentdeckung kann sich erst später auszahlen. Keine Etappe allein nach kurzfristigen XP/s freigeben.

`PlanContinuation` implementiert Schritte 1–3 über die bestehende Erweiterung. Es verwendet mehr Rechenarbeit als die Baseline; der Versuch belegt den Nutzen zusätzlicher Kandidaten, keine bessere Suchqualität bei gleichem CPU-Budget. Vier feste Strategien und maximal 4.000 Entscheidungen pro Strategie begrenzen die Versuchsgröße. Es gibt noch kein Zeitlimit dieser Offlinefunktion.

## Reproduktion und Messungen

Eingabe: v2.5-Datensatz SHA-256 `f8a5fb9dd4ea2bc55d937d91e4408cae0b6854f187d916a95b9e0557ecc1cfaf`, veröffentlichter Quellstand `de5e3207ed37cddc96d2d683bc7817b9f35e5abb`, dessen originale Solo-Routen und Klassenprofile. Vollständige Eingabehashes und Einzelmessungen stehen in [continuation-v25.json](../data/audits/continuation-v25.json).

```sh
go run ./cmd/foreverroutes extend -route data/routes/orc-shaman.json -target 40 -portfolio -budget 0s -out candidate.json -report trials.json
go run ./cmd/foreverroutes check-quests -target 40 -out quest-readiness.json
```

`-budget 0s` schaltet hier nur die nachgeschaltete lokale Verbesserung aus, nicht die vier Planungsdurchläufe. Die Berichtssekunden zählen ab dem nachgespielten Ende der 1–20-Route; sie enthalten nicht erneut die Startstrecke. Die tatsächliche Anfangs-XP kann über der Schwelle von Level 20 liegen.

| Charakter / Ziel | Baseline, Stunden ab Ende 1–20 | Portfolio, Stunden | Gewinner | Änderung |
| --- | ---: | ---: | --- | ---: |
| Orc Schamane / 25 | 10,26 | 10,26 | baseline | 0 % |
| Orc Schamane / 40 | 57,26 | 54,63 | deep | −4,58 % |
| Mensch Krieger / 40 | 80,41 | 80,41 | baseline | 0 % |
| Orc Hexenmeister / 40 | 59,04 | 59,04 | baseline | 0 % |

Das Ergebnis ist gemischt: Die tiefere Suche verbessert nur einen der drei untersuchten 20–40-Fälle. Beim Schamanen bis 25 verliert sie sogar gegen die Baseline. Es rechtfertigt den Vergleich unabhängiger Kandidaten, nicht die pauschale Umstellung auf tiefe Suche. Drei Kontexte sind noch kein Rosterbenchmark und enthalten keinen Dungeonvergleich. Die ausgegebene erweiterte Guideprojektion durchläuft anschließend zusätzlich `FollowGuide`/`Improve`; ihre Gesamtzeit ist deshalb nicht einfach der Messwert dieses separaten Kandidatenvergleichs.

Die vier ausgegebenen Guideprojektionen liegen ausschließlich im [komprimierten Auditarchiv](../data/audits/continuation-v25-candidates.json.gz), nicht in den ausgelieferten Routen. Die Abschlussläufe mit Klassenquestprüfung lieferten identische Einzelmessungen wie die ersten Läufe. Regressionstests prüfen die unveränderte Baseline, das unveränderte Ausgangsquestlog, Aktions-Replay, ungültige Ziele und das Verwerfen einer Fortsetzung ohne erforderliche Klassenfreischaltung.

## Was vor einer Freigabe noch fehlt

Der Quellaudit mit Ziel 40 untersucht 2.920 Datensätze über 56 Charakterkontexte. Im Teilbereich **Questlevel 20 bis einschließlich 40** liegen 1.317 Quests: 847 `modelled`, 13 einmalig modellierte Wiederholungsquests, 145 `data_gap`, 235 `unresolved`, 35 bis Ziel 40 nicht ausführbar, 10 nicht gewählte Gruppen-/PvP-Quests und 32 außerhalb des Levelprofils. `modelled` bedeutet nicht in jedem Kontext solo machbar und nicht ingame verifiziert. Der Filter verwendet das Questlevel, nicht das Mindestannahmelevel.

Die Klassenprofile verwenden Starterwaffen und keine individuelle Talent-/Ausrüstungsentwicklung. Klassenfreischaltungen nach den Startketten benötigen eine neue Prüfung; insbesondere ist die heutige Pflichtliste kein vollständiger Meilensteinkatalog bis 40. Reisen nutzen das bestehende Lauf-/Flug-/Schiffs-/Ruhesteinmodell; individuelle Reitfähigkeit und Kauf-/Trainingsbudget werden nicht als vollständige dynamische Entwicklung modelliert. Auch Drops, respawnende Ziele, Escort-/Eventdauer und ungemessene Wege bleiben Fehlerquellen.

Freigabe deshalb erst nach: vollständiger 56-Kontexte-Solo-Prüfung (Dungeonvarianten separat), Replay und XP-Ledger jeder Route; Vergleich mit bisheriger gültiger Route unter gleichen Bedingungen; Prüfung der notwendigen Klassenfreischaltungen und unbekannten Questquellen; repräsentativen Ingame-Messungen von Kampf, Regeneration und Reise. Bis dahin keine Aussage „20–40 fertig“ und keine automatisch aktivierten Prototyprouten im Addon.
