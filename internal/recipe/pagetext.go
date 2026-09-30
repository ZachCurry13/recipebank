package recipe

import (
	"bytes"
	"strings"

	xhtml "golang.org/x/net/html"
)

// skipTags hold no recipe text worth sending to the AI.
var skipTags = map[string]bool{"script": true, "style": true, "noscript": true, "svg": true, "nav": true,
	"header": true, "footer": true, "aside": true, "form": true, "iframe": true, "button": true}

// PageText is a page's readable text (for the AI when a page has no JSON-LD
// recipe), cut to about max characters.
func PageText(page []byte, max int) (title, text string) {
	doc, err := xhtml.Parse(bytes.NewReader(page))
	if err != nil {
		return "", ""
	}
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if b.Len() > max {
			return
		}
		if n.Type == xhtml.ElementNode {
			if n.Data == "title" && n.FirstChild != nil && title == "" {
				title = strings.TrimSpace(n.FirstChild.Data)
			}
			if skipTags[n.Data] {
				return
			}
		}
		if n.Type == xhtml.TextNode {
			if t := strings.Join(strings.Fields(n.Data), " "); t != "" {
				b.WriteString(t)
				b.WriteByte('\n')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	text = b.String()
	if len(text) > max {
		text = text[:max]
	}
	return title, text
}
