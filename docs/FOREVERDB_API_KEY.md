# ForeverDB API: Key sicher hinterlegen

ForeverRoutes nutzt ein **Repository Actions Secret** mit dem exakten Namen `FOREVERDB_API_KEY`. Der geheime Wert wird **nicht** im Chat, Quellcode, Git-Commit oder in Workflow-Logs gespeichert.

1. GitHub: https://github.com/Battlefeld/ForeverRoutes/settings/secrets/actions
2. **New repository secret** wählen.
3. Name: `FOREVERDB_API_KEY`
4. Value: deinen von ForeverDB freigeschalteten persönlichen API-Key einfügen und **Add secret**.
5. Über https://github.com/Battlefeld/ForeverRoutes/actions/workflows/foreverdb-key-check.yml → **Run workflow** den manuellen Verbindungstest ausführen.

Die Prüfung ruft ausschließlich `GET https://foreverdb.net/api/v1/meta` ab und sendet den Key gemäß https://foreverdb.net/developers als `X-API-Key`-Header. Weder Key noch Rohantwort werden protokolliert. Ein fehlender oder abgelehnter Key führt zu einem klaren Fehlstatus.

**Wichtig:** Das Aktivieren des Secrets allein importiert keine Daten. Der vollständige ForeverDB-Questsynchronisierer wird separat über einen geprüften Adapter eingebaut; er erhält dann denselben Secret-Namen. Bestehende Builds und ForeverSim bleiben davon unberührt.
