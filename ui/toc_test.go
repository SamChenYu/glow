package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
)

func TestParseHeaders(t *testing.T) {
	md := "# Title\n\nText\n\n## Section\n\n### Sub\n"
	headers := parseHeaders(md)
	if len(headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(headers))
	}
	if headers[0].title != "Title" || headers[0].level != 1 {
		t.Errorf("header 0: got %q level %d", headers[0].title, headers[0].level)
	}
	if headers[1].title != "Section" || headers[1].level != 2 {
		t.Errorf("header 1: got %q level %d", headers[1].title, headers[1].level)
	}
	if headers[2].title != "Sub" || headers[2].level != 3 {
		t.Errorf("header 2: got %q level %d", headers[2].title, headers[2].level)
	}
}

func TestParseHeadersSkipsCodeBlocks(t *testing.T) {
	md := "# Real\n\n```\n# Not a header\n```\n\n## Also Real\n"
	headers := parseHeaders(md)
	if len(headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(headers))
	}
}

func TestFindHeaderLinesWithGlamour(t *testing.T) {
	md := "# Main Title\n\nSome text.\n\n## Section One\n\nMore text.\n\n### Subsection A\n\nDetails.\n"

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath(styles.DarkStyle),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		t.Fatal(err)
	}

	rendered, err := r.Render(md)
	if err != nil {
		t.Fatal(err)
	}

	contentLines := strings.Split(rendered, "\n")

	t.Logf("Rendered content has %d lines:", len(contentLines))
	for i, line := range contentLines {
		t.Logf("  line %3d: %q", i, line)
	}

	headers := parseHeaders(md)
	if len(headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(headers))
	}

	findHeaderLines(headers, contentLines)

	for i, h := range headers {
		t.Logf("Header %d: %q -> renderedLine=%d", i, h.title, h.renderedLine)
		if h.renderedLine == -1 {
			t.Errorf("header %q was NOT matched to any rendered line", h.title)
		}
	}
}
