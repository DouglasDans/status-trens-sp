package source

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// MetroURL is the "Direto do Metrô" page of the Metrô website. It has no JSON
// behind it: the status is rendered into the HTML on the server.
const MetroURL = "https://www.metro.sp.gov.br/wp-content/themes/metrosp/direto-metro.php"

// maxPage bounds what is read from the page (the real one is about 40 KB).
const maxPage = 1 << 20

// Metro reads the lines listed on "Direto do Metrô": each one is an
// li.linha inside ol.direto-metro, with the operator's description inside
// the HTML of the info tooltip. A layout change fails the whole source; one
// broken line is only skipped.
type Metro struct {
	url    string
	client *http.Client
	logger *slog.Logger
}

func NewMetro(url string, client *http.Client, logger *slog.Logger) *Metro {
	return &Metro{url: url, client: client, logger: logger}
}

func (m *Metro) Name() string { return "metro" }

var backgroundColor = regexp.MustCompile(`background-color:\s*(#[0-9a-fA-F]{3,8})`)

func (m *Metro) Fetch(ctx context.Context) (Snapshot, error) {
	doc, err := m.page(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", m.Name(), err)
	}
	list := find(doc, func(n *html.Node) bool { return n.Data == "ol" && hasClass(n, "direto-metro") })
	if list == nil {
		return Snapshot{}, fmt.Errorf("%s: line list (ol.direto-metro) not found", m.Name())
	}

	var readings []Reading
	for li := range list.ChildNodes() {
		if li.Type != html.ElementNode || !hasClass(li, "linha") {
			continue
		}
		r, err := m.line(li)
		if err != nil {
			m.logger.Warn("skipping line", "source", m.Name(), "error", err)
			continue
		}
		readings = append(readings, r)
	}
	if len(readings) == 0 {
		return Snapshot{}, fmt.Errorf("%s: no valid line in the list", m.Name())
	}
	return Snapshot{Source: m.Name(), Readings: readings}, nil
}

func (m *Metro) page(ctx context.Context) (*html.Node, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	if mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil || mediaType != "text/html" {
		return nil, fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, maxPage))
	if err != nil {
		return nil, fmt.Errorf("parsing page: %w", err)
	}
	return doc, nil
}

func (m *Metro) line(li *html.Node) (Reading, error) {
	number := child(li, "linha-numero")
	code, err := strconv.Atoi(text(number))
	if err != nil || code <= 0 {
		return Reading{}, fmt.Errorf("line number %q", text(number))
	}
	label := text(child(li, "linha-situacao"))
	state, ok := status.Normalize(label)

	var colorHex string
	if c := backgroundColor.FindStringSubmatch(attr(number, "style")); c != nil {
		colorHex = c[1]
	}
	return Reading{
		Code:        code,
		Name:        lineName(text(child(li, "linha-nome"))),
		ColorHex:    colorHex,
		State:       state,
		Recognized:  ok,
		Label:       label,
		Description: cleanDescription(label, tooltipDescription(attr(child(li, "linha-info"), "data-bs-title"))),
	}, nil
}

// tooltipDescription reads div.description from the tooltip, which is HTML
// stored inside an attribute.
func tooltipDescription(tooltip string) string {
	if tooltip == "" {
		return ""
	}
	nodes, err := html.ParseFragment(strings.NewReader(tooltip), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return ""
	}
	for _, n := range nodes {
		if d := find(n, func(n *html.Node) bool { return hasClass(n, "description") }); d != nil {
			return text(d)
		}
	}
	return ""
}

// find returns the first element under n (n included) that matches.
func find(n *html.Node, match func(*html.Node) bool) *html.Node {
	for d := range n.Descendants() {
		if d.Type == html.ElementNode && match(d) {
			return d
		}
	}
	if n.Type == html.ElementNode && match(n) {
		return n
	}
	return nil
}

func child(n *html.Node, class string) *html.Node {
	return find(n, func(n *html.Node) bool { return hasClass(n, class) })
}

func hasClass(n *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attr(n, "class")), class)
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// text joins the text under n, with whitespace collapsed.
func text(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	for d := range n.Descendants() {
		if d.Type == html.TextNode {
			b.WriteString(d.Data)
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
