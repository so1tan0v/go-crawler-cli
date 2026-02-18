package crawler

import (
	"bytes"
	"code/internal/domain"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

func extractAssets(baseURL string, htmlBytes []byte) ([]domain.Asset, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var out []domain.Asset

	add := func(typ, raw string) {
		if abs := normalizeLink(base, raw); abs != "" {
			if _, ok := seen[abs]; ok {
				return
			}

			seen[abs] = struct{}{}
			out = append(out, domain.Asset{
				URL:        abs,
				Type:       typ,
				StatusCode: 0,
				SizeBytes:  0,
				Error:      "",
			})
		}
	}

	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("src"); ok {
			add("image", v)
		}
	})

	doc.Find("script[src]").Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("src"); ok {
			add("script", v)
		}
	})

	doc.Find(`link[rel="stylesheet"][href]`).Each(func(_ int, s *goquery.Selection) {
		if v, ok := s.Attr("href"); ok {
			add("style", v)
		}
	})

	return out, nil
}
