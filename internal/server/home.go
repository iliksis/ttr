package server

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"
)

//go:embed templates/home.html.tmpl
var homeTemplateFS embed.FS

var homeTemplate = template.Must(template.ParseFS(homeTemplateFS, "templates/home.html.tmpl"))

// sortField names a column the roster table can be sorted by.
type sortField string

const (
	sortByName sortField = "name"
	sortByTTR  sortField = "ttr"
	sortByQTTR sortField = "qttr"
)

// defaultSortDir is the direction a column sorts in the first time it's
// selected, chosen per-column so a fresh click lands on the more useful
// order (names read alphabetically, ratings read strongest-first).
var defaultSortDir = map[sortField]string{
	sortByName: "asc",
	sortByTTR:  "desc",
	sortByQTTR: "desc",
}

func parseSortField(s string) sortField {
	switch sortField(s) {
	case sortByName, sortByTTR, sortByQTTR:
		return sortField(s)
	default:
		return sortByQTTR
	}
}

// parseSortDir validates a dir query value, falling back to the given
// column's own default direction when it's absent or invalid.
func parseSortDir(s string, field sortField) string {
	if s == "asc" || s == "desc" {
		return s
	}
	return defaultSortDir[field]
}

// orderByClause maps a validated sort field/direction into SQL. Rating
// columns push NULLs (no snapshot captured yet) to the end regardless of
// direction, via the "is this NULL" boolean sorting before the value.
func orderByClause(field sortField, dir string) string {
	switch field {
	case sortByName:
		return "p.last_name COLLATE NOCASE " + dir + ", p.first_name COLLATE NOCASE " + dir
	case sortByTTR:
		return "(latest_ttr IS NULL), latest_ttr " + dir
	default:
		return "(latest_qttr IS NULL), latest_qttr " + dir
	}
}

type homePlayerRow struct {
	ID        int64
	Name      string
	TTRValue  string
	TTRDate   string
	QTTRValue string
	QTTRDate  string
}

type homeViewModel struct {
	Players   []homePlayerRow
	SortLinks struct {
		Name string
		TTR  string
		QTTR string
	}
}

// handleHome renders the Viewer's home page: every Player with their most
// recent TTR and QTTR Rating snapshot, sortable by name, TTR, or QTTR.
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	field := parseSortField(r.URL.Query().Get("sort"))
	dir := parseSortDir(r.URL.Query().Get("dir"), field)

	players, ok := selectRows[playerSummary](s, w, playerSummaryQuery+orderByClause(field, dir), ttrRatingType, qttrRatingType)
	if !ok {
		return
	}

	vm := homeViewModel{Players: make([]homePlayerRow, len(players))}
	for i, p := range players {
		ttrValue, ttrDate := formatRating(p.LatestTTR, p.LatestTTRAt)
		qttrValue, qttrDate := formatRating(p.LatestQTTR, p.LatestQTTRAt)
		vm.Players[i] = homePlayerRow{
			ID:        p.ID,
			Name:      p.FirstName + " " + p.LastName,
			TTRValue:  ttrValue,
			TTRDate:   ttrDate,
			QTTRValue: qttrValue,
			QTTRDate:  qttrDate,
		}
	}
	vm.SortLinks.Name = sortLink(field, dir, sortByName)
	vm.SortLinks.TTR = sortLink(field, dir, sortByTTR)
	vm.SortLinks.QTTR = sortLink(field, dir, sortByQTTR)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := homeTemplate.Execute(w, vm); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "render failed")
	}
}

// sortLink builds the href for a column header: clicking the already-active
// column flips its direction, clicking any other column switches to it at
// that column's default direction.
func sortLink(currentField sortField, currentDir string, column sortField) string {
	dir := defaultSortDir[column]
	if column == currentField {
		if currentDir == "asc" {
			dir = "desc"
		} else {
			dir = "asc"
		}
	}
	return "/?sort=" + string(column) + "&dir=" + dir
}

// formatRating renders a Rating snapshot's value and captured date, or "-"
// placeholders when no snapshot has been captured yet.
func formatRating(value *int, capturedAt *string) (valueStr, dateStr string) {
	if value == nil {
		return "-", ""
	}
	date := ""
	if capturedAt != nil && len(*capturedAt) >= 10 {
		date = (*capturedAt)[:10]
	}
	return strconv.Itoa(*value), date
}
