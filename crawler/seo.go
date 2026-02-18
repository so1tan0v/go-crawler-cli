package crawler

import (
	"bytes"
	"code/internal/domain"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var whitespaceRE = regexp.MustCompile(`\s+`)

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	s = whitespaceRE.ReplaceAllString(s, " ")
	return s
}

func extractSEO(htmlBytes []byte) domain.SEO {
	seo := domain.SEO{}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return seo
	}

	titleSel := doc.Find("title").First()
	if titleSel.Length() > 0 {
		seo.HasTitle = true
		seo.Title = cleanText(titleSel.Text())
	}

	descSel := doc.Find(`meta[name="description"]`).First()
	if descSel.Length() > 0 {
		seo.HasDescription = true
		if content, ok := descSel.Attr("content"); ok {
			seo.Description = cleanText(content)
		}
	}

	h1Sel := doc.Find("h1").First()
	if h1Sel.Length() > 0 {
		seo.HasH1 = true
	}

	return seo
}
