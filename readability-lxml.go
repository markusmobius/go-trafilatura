package trafilatura

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-shiori/dom"
	"github.com/markusmobius/go-trafilatura/v2/internal/etree"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	readabilityLxmlUnlikely = regexp.MustCompile(`(?i)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup|tweet|twitter`)
	readabilityLxmlPossible = regexp.MustCompile(`(?i)and|article|body|column|main|shadow`)
	readabilityLxmlPositive = regexp.MustCompile(`(?i)article|body|content|entry|hentry|main|page|pagination|post|text|blog|story`)
	readabilityLxmlNegative = regexp.MustCompile(`(?i)button|combx|comment|com-|contact|figure|foot|footer|footnote|form|input|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget`)
	readabilityLxmlVideo    = regexp.MustCompile(`(?i)https?://(?:www\.)?(?:youtube|vimeo)\.com`)
)

func readabilityLxmlAttribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Namespace == "" && attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func readabilityLxmlTextLength(node *html.Node) int {
	return utf8.RuneCountInString(trim(dom.TextContent(node)))
}

func readabilityLxmlLinkDensity(node *html.Node) float64 {
	length := 0
	for _, link := range dom.GetElementsByTagName(node, "a") {
		length += readabilityLxmlTextLength(link)
	}
	return float64(length) / float64(max(readabilityLxmlTextLength(node), 1))
}

func readabilityLxmlClassWeight(node *html.Node) float64 {
	weight := 0.0
	for _, name := range []string{"class", "id"} {
		value := readabilityLxmlAttribute(node, name)
		if readabilityLxmlPositive.MatchString(value) {
			weight += 25
		}
		if readabilityLxmlNegative.MatchString(value) {
			weight -= 25
		}
	}
	return weight
}

func readabilityLxmlInitialScore(node *html.Node) float64 {
	score := readabilityLxmlClassWeight(node)
	switch node.Data {
	case "div", "article":
		score += 5
	case "pre", "td", "blockquote":
		score += 3
	case "address", "ol", "ul", "dl", "dd", "dt", "li", "form", "aside":
		score -= 3
	case "h1", "h2", "h3", "h4", "h5", "h6", "th", "header", "footer", "nav":
		score -= 5
	}
	return score
}

func readabilityLxmlRemoveUnlikely(root *html.Node) {
	for _, node := range dom.GetElementsByTagName(root, "*") {
		attributes := readabilityLxmlAttribute(node, "class") + " " + readabilityLxmlAttribute(node, "id")
		if node.Data != "html" && node.Data != "body" && readabilityLxmlUnlikely.MatchString(attributes) && !readabilityLxmlPossible.MatchString(attributes) {
			etree.Remove(node, true)
		}
	}
}

func readabilityLxmlTransformDivs(root *html.Node) {
	for _, node := range dom.GetElementsByTagName(root, "div") {
		containsBlock := false
		for _, child := range dom.GetElementsByTagName(node, "*") {
			for _, prefix := range []string{"a", "blockquote", "dl", "div", "img", "ol", "p", "table", "ul"} {
				if strings.HasPrefix(child.Data, prefix) {
					containsBlock = true
					break
				}
			}
			if containsBlock {
				break
			}
		}
		if !containsBlock {
			node.Data, node.DataAtom = "p", atom.P
		}
	}
	for _, node := range dom.GetElementsByTagName(root, "div") {
		leading := etree.Text(node)
		if strings.TrimSpace(leading) != "" {
			etree.SetTextSlot(node, etree.TextValue{})
			paragraph := dom.CreateElement("p")
			etree.SetText(paragraph, leading)
			node.InsertBefore(paragraph, node.FirstChild)
		}
		children := dom.Children(node)
		for index := len(children) - 1; index >= 0; index-- {
			child := children[index]
			tail := etree.Tail(child)
			if strings.TrimSpace(tail) != "" {
				etree.SetTail(child, "")
				paragraph := dom.CreateElement("p")
				etree.SetText(paragraph, tail)
				node.InsertBefore(paragraph, child.NextSibling)
			}
			if child.Data == "br" {
				etree.Remove(child, true)
			}
		}
	}
}

func readabilityLxmlScores(root *html.Node) (map[*html.Node]float64, *html.Node) {
	scores := make(map[*html.Node]float64)
	var order []*html.Node
	for _, node := range etree.Iter(root, "p", "pre", "td") {
		parent := node.Parent
		if parent == nil || parent.Type != html.ElementNode {
			continue
		}
		content := trim(dom.TextContent(node))
		length := utf8.RuneCountInString(content)
		if length < 25 {
			continue
		}
		grandparent := parent.Parent
		if grandparent != nil && grandparent.Type != html.ElementNode {
			grandparent = nil
		}
		for _, ancestor := range []*html.Node{parent, grandparent} {
			if ancestor == nil {
				continue
			}
			if _, found := scores[ancestor]; !found {
				scores[ancestor] = readabilityLxmlInitialScore(ancestor)
				order = append(order, ancestor)
			}
		}
		score := 2 + float64(strings.Count(content, ",")) + min(float64(length)/100, 3)
		scores[parent] += score
		if grandparent != nil {
			scores[grandparent] += score / 2
		}
	}
	var best *html.Node
	for _, node := range order {
		scores[node] *= 1 - readabilityLxmlLinkDensity(node)
		if best == nil || scores[node] > scores[best] {
			best = node
		}
	}
	return scores, best
}

func readabilityLxmlArticle(best *html.Node, scores map[*html.Node]float64) *html.Node {
	threshold := max(scores[best]*0.2, 10)
	siblings := []*html.Node{best}
	if best.Parent != nil {
		siblings = dom.Children(best.Parent)
	}
	frame := dom.CreateElement("html")
	body := etree.SubElement(frame, "body")
	output := etree.SubElement(body, "div")
	for _, sibling := range siblings {
		score, scored := scores[sibling]
		include := sibling == best || (scored && score >= threshold)
		if !include && sibling.Data == "p" {
			density := readabilityLxmlLinkDensity(sibling)
			content := etree.Text(sibling)
			length := utf8.RuneCountInString(content)
			include = (length > 80 && density < 0.25) || (length <= 80 && density == 0 && (strings.Contains(content, ". ") || strings.HasSuffix(content, ".")))
		}
		if include {
			etree.Append(output, sibling)
		}
	}
	return output
}

func readabilityLxmlSanitize(root *html.Node, scores map[*html.Node]float64) {
	for _, node := range etree.Iter(root, "h1", "h2", "h3", "h4", "h5", "h6") {
		if readabilityLxmlClassWeight(node) < 0 || readabilityLxmlLinkDensity(node) > 0.33 {
			etree.Remove(node, true)
		}
	}
	for _, node := range etree.Iter(root, "form", "textarea") {
		etree.Remove(node, true)
	}
	for _, node := range etree.Iter(root, "iframe") {
		if readabilityLxmlVideo.MatchString(readabilityLxmlAttribute(node, "src")) {
			etree.SetText(node, "VIDEO")
		} else {
			etree.Remove(node, true)
		}
	}
	top := root
	for top.Parent != nil {
		top = top.Parent
	}
	allowed := make(map[*html.Node]bool)
	nodes := etree.Iter(top, "table", "ul", "div", "aside", "header", "footer", "section")
	for index := len(nodes) - 1; index >= 0; index-- {
		node := nodes[index]
		if allowed[node] {
			continue
		}
		weight := readabilityLxmlClassWeight(node)
		if weight+scores[node] < 0 {
			etree.Remove(node, true)
			continue
		}
		if strings.Count(dom.TextContent(node), ",") >= 10 {
			continue
		}
		paragraphs := len(dom.GetElementsByTagName(node, "p"))
		images := len(dom.GetElementsByTagName(node, "img"))
		items := len(dom.GetElementsByTagName(node, "li")) - 100
		embeds := len(dom.GetElementsByTagName(node, "embed"))
		inputs := 0
		for _, input := range dom.GetElementsByTagName(node, "input") {
			if readabilityLxmlAttribute(input, "type") != "hidden" {
				inputs++
			}
		}
		length := readabilityLxmlTextLength(node)
		density := readabilityLxmlLinkDensity(node)
		remove := (paragraphs != 0 && float64(images) > 1+float64(paragraphs)*1.3) ||
			(items > paragraphs && node.Data != "ol" && node.Data != "ul") ||
			float64(inputs) > float64(paragraphs)/3 ||
			(length < 25 && (images == 0 || images > 2)) ||
			(weight < 25 && density > 0.2) || (weight >= 25 && density > 0.5) ||
			(embeds == 1 && length < 75) || embeds > 1
		if remove {
			etree.Remove(node, true)
		} else if length == 0 {
			following, preceding := 0, 0
			for sibling := node.NextSibling; sibling != nil; sibling = sibling.NextSibling {
				if sibling.Type == html.ElementNode {
					following = readabilityLxmlTextLength(sibling)
					if following > 0 {
						break
					}
				}
			}
			for sibling := node.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
				if sibling.Type == html.ElementNode {
					preceding = readabilityLxmlTextLength(sibling)
					if preceding > 0 {
						break
					}
				}
			}
			if following+preceding > 1000 {
				for _, child := range etree.Iter(node, "table", "ul", "div", "section") {
					allowed[child] = true
				}
			} else {
				etree.Remove(node, true)
			}
		}
	}
}

func readabilityLxmlXMLLength(root *html.Node) int {
	escapedLength := func(value string, attribute bool) int {
		length := 0
		for _, character := range value {
			switch {
			case character == '&':
				length += 5
			case character == '<' || character == '>':
				length += 4
			case attribute && character == '"':
				length += 6
			case character == '\r' || (attribute && (character == '\n' || character == '\t')):
				length += 5
			default:
				length++
			}
		}
		return length
	}
	length := escapedLength(etree.Tail(root), false)
	pending := []*html.Node{root}
	for len(pending) > 0 {
		node := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		switch node.Type {
		case html.TextNode:
			length += escapedLength(node.Data, false)
		case html.CommentNode:
			length += 7 + utf8.RuneCountInString(node.Data)
		case html.ElementNode:
			tagLength := utf8.RuneCountInString(node.Data)
			if node.FirstChild == nil {
				length += tagLength + 3
			} else {
				length += tagLength*2 + 5
			}
			for _, attribute := range node.Attr {
				length += 4 + utf8.RuneCountInString(attribute.Key) + escapedLength(attribute.Val, true)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			pending = append(pending, child)
		}
	}
	return length
}

func cloneReadabilityLxml(source *html.Node) *html.Node {
	clone := &html.Node{Type: source.Type, DataAtom: source.DataAtom, Data: source.Data, Namespace: source.Namespace, Attr: append([]html.Attribute(nil), source.Attr...)}
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		clone.AppendChild(cloneReadabilityLxml(child))
	}
	return clone
}

func extractReadabilityLxml(source *html.Node) *html.Node {
	if source == nil {
		return nil
	}
	root := cloneReadabilityLxml(source)
	if root.Type == html.DocumentNode {
		for child := root.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode {
				root = child
				break
			}
		}
	}
	for _, node := range etree.Iter(root, "script", "style", "fencedframe") {
		etree.Remove(node, true)
	}
	for _, ruthless := range []bool{true, false} {
		if ruthless {
			readabilityLxmlRemoveUnlikely(root)
		}
		readabilityLxmlTransformDivs(root)
		scores, best := readabilityLxmlScores(root)
		if best != nil {
			root = readabilityLxmlArticle(best, scores)
		} else if ruthless {
			continue
		} else {
			for child := root.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.ElementNode && child.Data == "body" {
					root = child
					break
				}
			}
		}
		readabilityLxmlSanitize(root, scores)
		if !ruthless || readabilityLxmlXMLLength(root) >= 250 {
			break
		}
	}
	return root
}
