# ForeverSim-Rückmeldung: vorbereitete, gesperrte Schnittstelle

ForeverSim selbst wird durch diese Änderung nicht bearbeitet. Die Routen-Engine kann zukünftig Rückmeldungen von ForeverCore entgegennehmen. **Im jetzigen Stand ist diese Funktion ausgeschaltet.** Ein Datum, Levelaufstieg, Ausrüstungsupgrade oder eingehender Datensatz aktiviert sie nicht.

Die Aktivierung verlangt unabhängig voneinander:

- ausdrücklich bestätigten Release,
- ausdrückliche Zustimmung des Nutzers **nach** dem Release,
- dokumentierten Release-Build und Bestätigungszeitpunkt,
- aktivierten lokalen Schalter.

[Die mitgelieferte Beispieldatei](../data/sources/feedback-policy.example.json) enthält ausschließlich deaktivierte beziehungsweise leere Werte. Es wird keine Zustimmung vorweggenommen. Die Bestätigung kommt aus einer lokalen, von ForeverCore verwalteten Einstellung, nicht aus dem ForeverSim-Nutzdatenpaket.

`next` liest eine solche Einstellung nur bei explizit gesetztem `-feedback-policy PFAD`. Ohne diesen Parameter bleibt die Schnittstelle gesperrt. Eine nicht vollständig bestätigte Datei wird abgelehnt. Eine Nutzdatenrückmeldung bei gesperrter Schnittstelle erzeugt einen Hinweis und verändert keine Kampfzeiten.

## Vertrag für die spätere Integration

Der bestehende `live`-Zustand kann zusätzliche Felder enthalten:

```json
{
 "equipmentRevision": "hash-of-equipped-items",
 "buildRevision": "hash-of-talents-rotation-and-combat-buff-context",
 "restedXP": 450,
 "hasSleepingBag": false,
 "bagStacks": 0,
 "bagBuffRemaining": 0,
 "bagCooldownRemaining": 0,
 "combatFeedback": {
  "schema": "foreverroutes.combat-feedback.v1",
  "gameBuild": "must-match-the-confirmed-release-build",
  "character": {"race": "Undead", "class": "Warlock"},
  "equipmentRevision": "hash-of-equipped-items",
  "buildRevision": "hash-of-talents-rotation-and-combat-buff-context",
  "source": "foreversim",
  "kills": [
   {"npc": 123, "playerLevel": 12, "fightSeconds": 14.5, "restSeconds": 3.0, "samples": 0}
  ]
 }
}
```

Die Zahlen im Beispiel sind ausschließlich Vertragsbeispiele, keine kalibrierten Mobwerte.

- Der Sender liefert den tatsächlichen Charakter, Ausrüstungsstand, Talent-/Rotations-/Buffkontext und Clientbuild.
- Ein Upgrade erzeugt eine neue `equipmentRevision`; relevante Talent-/Rotations-/Buffänderungen eine neue `buildRevision`.
- Zeiten gelten ausschließlich für denselben Charakter, Ausrüstungs- und Buildkontext, Spielerlevel und NPC. Es gibt keine pauschale Übertragung auf andere Gegner oder Level.
- Simulierte Werte verwenden `source=foreversim`. Beobachtete Werte verwenden `source=measured` und mindestens drei Samples. Nichtpositive, negative, unendliche und NaN-Zeiten werden abgelehnt.
- Geplante bekannte Ausrüstungsupgrades, etwa ein neuer Zauberstab, verwerfen die alte Ausrüstungsrevision. Bis zu einer passenden bestätigten Rückmeldung bleibt das feste Vergleichsprofil mit seinen vorhandenen Modifikatoren die Grundlage.
- Fehlende Angaben zu Zauberstab oder Berufen werden nicht mehr allein aus dem Level erfunden. Bestätigte Rested-XP und aktive Schlafsacktimer können übernommen werden.
- ForeverCore muss einen abgeschlossenen Zustand je Planung übergeben. Ein laufend benutztes `World`-Objekt darf nicht gleichzeitig durch eingehende Updates verändert werden; die nächste Planung erhält den nächsten konsistenten Zustand.

Der Sender und die Aktivierung im Spiel werden erst nach Release und erneuter ausdrücklicher Zustimmung verbunden. Die aktuellen Tests verwenden für reale Routen weiterhin die eingefrorenen Warlock-Profildaten. Positive Freigaben existieren nur in synthetischen Schnittstellentests.
