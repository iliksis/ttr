# TTR

Tracks table-tennis rating points for one club's roster, scraped from mytischtennis.de behind a login and viewed over time.

## Language

**Player**:
A member of the club whose ratings are tracked. Identified by their mytischtennis.de player id.
_Avoid_: Member, athlete, user (User is reserved for whoever operates the extension/webapp)

**Club**:
The single table-tennis club whose roster this project tracks. Not a modeled entity with its own id in this project's scope — just the fixed set of Players.

**Rating**:
A numeric score representing a Player's competitive strength on mytischtennis.de. Two kinds exist: TTR and QTTR.
_Avoid_: Score, ranking, points (use "rating points" only when referring to the raw number)

**TTR**:
A Player's standard rating on mytischtennis.de, updated after rated matches — in practice this means new values appear roughly daily.

**QTTR**:
A Player's DTTB-managed qualification rating on mytischtennis.de, distinct from TTR and tracked separately. Updated quarterly by the DTTB, not after individual matches — so its Rating snapshots are far sparser over time than TTR's.

**Rating snapshot**:
One TTR or QTTR value for one Player as observed at a point in time. The unit of data the Extension reports and the Server stores.
_Avoid_: Rating history (that's the ordered series of snapshots, not a single one), current rating (a snapshot is always timestamped, never "current" on its own)

**Capture**:
The Extension's act of intercepting a Rating value from mytischtennis.de's hidden API while the user browses the site, before it becomes a Rating snapshot sent to the Server.
_Avoid_: Scrape (scraping implies parsing rendered HTML; this project intercepts API responses instead)

**Ingestion**:
The Server's act of receiving and storing a Rating snapshot reported by the Extension, authenticated by the Ingestion key.
_Avoid_: Upload, sync

**Ingestion key**:
The static API key baked into the Extension's configuration and checked by the Server on every Ingestion request.
_Avoid_: API key alone (ambiguous with a future mytischtennis.de API key, if one is ever needed), token, secret

**Extension**:
The cross-browser (Chrome + Firefox) browser extension that rides the user's own mytischtennis.de session, Captures Rating values, and reports them to the Server as Rating snapshots.

**Server**:
The self-hosted backend (deployed to Fly.io) that exposes a REST API for Ingestion and for serving stored data to the Viewer.

**Viewer**:
The webapp that reads Rating snapshots from the Server and displays them, e.g. as history/trend views per Player.
_Avoid_: Webapp alone (Viewer names its role; "webapp" is just its delivery mechanism)
