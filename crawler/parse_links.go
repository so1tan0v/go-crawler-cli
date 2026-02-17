package crawler

import (
	"bytes"
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractPageLinks(baseURL string, htmlBytes []byte) ([]string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var out []string

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("href"); ok {
			if abs := normalizeLink(base, v); abs != "" {
				if _, ok := seen[abs]; ok {
					return
				}
				seen[abs] = struct{}{}
				out = append(out, abs)
			}
		}
	})

	sort.Strings(out)

	return out, nil
}

func normalizeLink(base *url.URL, raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "#") {
		return ""
	}

	low := strings.ToLower(s)
	if strings.HasPrefix(low, "mailto:") || strings.HasPrefix(low, "tel:") || strings.HasPrefix(low, "javascript:") {
		return ""
	}

	u, err := url.Parse(s)
	if err != nil {
		return ""
	}

	abs := base.ResolveReference(u)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return ""
	}

	abs.Fragment = ""
	if abs.Path == "/" {
		abs.Path = ""
		abs.RawPath = ""
	}

	return abs.String()
}
