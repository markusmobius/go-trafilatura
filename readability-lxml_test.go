package trafilatura

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	readability "github.com/markusmobius/go-readabilityV2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func renderReadabilityLxmlForTest(test *testing.T, node *html.Node) string {
	test.Helper()
	var output strings.Builder
	if err := html.Render(&output, node); err != nil {
		test.Fatal(err)
	}
	return output.String()
}

func serializeReadabilityLxmlTreeForTest(test *testing.T, root *html.Node) string {
	test.Helper()
	var output strings.Builder
	encoder := xml.NewEncoder(&output)
	encode := func(token xml.Token) {
		if err := encoder.EncodeToken(token); err != nil {
			test.Fatal(err)
		}
	}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		switch node.Type {
		case html.ElementNode:
			start := xml.StartElement{Name: xml.Name{Space: node.Namespace, Local: node.Data}}
			for _, attribute := range node.Attr {
				start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Space: attribute.Namespace, Local: attribute.Key}, Value: attribute.Val})
			}
			encode(start)
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				visit(child)
			}
			encode(start.End())
		case html.TextNode:
			encode(xml.CharData(node.Data))
		case html.CommentNode:
			encode(xml.Comment(node.Data))
		default:
			test.Fatalf("unexpected candidate node type: %d", node.Type)
		}
	}
	visit(root)
	if err := encoder.Flush(); err != nil {
		test.Fatal(err)
	}
	return output.String()
}

func TestReadabilityLxmlPythonControls(test *testing.T) {
	footer := strings.Repeat("Legal footer material should remain excluded. ", 15)
	for _, sample := range []struct{ source, expected string }{
		{"<html><body><div><p>This is a sufficiently long article paragraph, with a little more detail.</p></div></body></html>", "<div><div><p>This is a sufficiently long article paragraph, with a little more detail.</p></div></div>"},
		{"<html><body><div class=\"footer\"><p>" + footer + "</p></div><p>Short article.</p></body></html>", "<body><p>Short article.</p></body>"},
		{"<html><body><div>Leading article text with enough details.<span>Inline detail.</span>Trailing explanatory text.<br/>Final sentence.</div></body></html>", "<div><div><body><p>Leading article text with enough details.<span>Inline detail.</span>Trailing explanatory text.<br/>Final sentence.</p></body></div></div>"},
	} {
		source, err := html.Parse(strings.NewReader(sample.source))
		if err != nil {
			test.Fatal(err)
		}
		original := renderReadabilityLxmlForTest(test, source)
		actual := renderReadabilityLxmlForTest(test, extractReadabilityLxml(source))
		if actual != sample.expected {
			test.Errorf("expected %q, got %q", sample.expected, actual)
		}
		if renderReadabilityLxmlForTest(test, source) != original {
			test.Error("fallback mutated caller input")
		}
	}
}

func TestReadabilityLxmlFallbackOption(test *testing.T) {
	source, err := html.Parse(strings.NewReader("<html><body><p>" + strings.Repeat("A detailed article, with supporting evidence and explanation. ", 12) + "</p></body></html>"))
	if err != nil {
		test.Fatal(err)
	}
	mozilla, err := readability.FromDocument(source, nil)
	if err != nil {
		test.Fatal(err)
	}
	_, defaultCandidate := createFallbackGenerators(source, Options{})[0]()
	if renderReadabilityLxmlForTest(test, defaultCandidate) != renderReadabilityLxmlForTest(test, mozilla.Node) {
		test.Error("default fallback no longer matches Mozilla Readability")
	}
	_, nativeCandidate := createFallbackGenerators(source, Options{ReadabilityFallback: ReadabilityLxml})[0]()
	if renderReadabilityLxmlForTest(test, nativeCandidate) != renderReadabilityLxmlForTest(test, extractReadabilityLxml(source)) {
		test.Error("explicit readability-lxml option did not select the native port")
	}
	for _, mode := range []ReadabilityFallback{ReadabilityMozilla, ReadabilityLxml} {
		custom, read, distill := dom.CreateElement("div"), dom.CreateElement("div"), dom.CreateElement("div")
		options := Options{ReadabilityFallback: mode, FallbackCandidates: &FallbackCandidates{Readability: read, Distiller: distill, Others: []*html.Node{custom}}}
		generators := createFallbackGenerators(source, options)
		for index, expected := range []*html.Node{custom, read, distill} {
			_, actual := generators[index]()
			if actual != expected {
				test.Errorf("mode %d did not preserve supplied candidate %d", mode, index)
			}
		}
	}
}

func TestReadabilityLxmlPythonCandidateCorpus(test *testing.T) {
	path := os.Getenv("READABILITY_LXML_REFERENCE")
	if path == "" {
		test.Skip("set READABILITY_LXML_REFERENCE to lab/python_fallback.py candidate-fixtures output")
	}
	type referenceAttribute struct {
		Namespace string
		Key       string
		Value     string
	}
	type referenceNode struct {
		Kind     string
		Tag      string
		Data     string
		Attrs    []referenceAttribute
		Children []int
	}
	fromNodes := func(records []referenceNode) *html.Node {
		if len(records) == 0 {
			test.Fatal("empty reference tree")
		}
		nodes := make([]*html.Node, len(records))
		for index, record := range records {
			node := &html.Node{Data: record.Data}
			switch record.Kind {
			case "element":
				node.Type, node.Data, node.DataAtom = html.ElementNode, record.Tag, atom.Lookup([]byte(record.Tag))
			case "text":
				node.Type = html.TextNode
			case "comment":
				node.Type = html.CommentNode
			default:
				test.Fatalf("unsupported reference node kind: %s", record.Kind)
			}
			for _, attribute := range record.Attrs {
				node.Attr = append(node.Attr, html.Attribute{Namespace: attribute.Namespace, Key: attribute.Key, Val: attribute.Value})
			}
			nodes[index] = node
		}
		for index, record := range records {
			for _, child := range record.Children {
				nodes[index].AppendChild(nodes[child])
			}
		}
		return nodes[0]
	}
	file, err := os.Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(bufio.NewReaderSize(file, 256*1024))
	checked, mismatches := 0, 0
	for {
		var sample struct {
			Name          string
			Nodes         []referenceNode
			ExpectedNodes []referenceNode `json:"expected_nodes"`
		}
		if err := decoder.Decode(&sample); err == io.EOF {
			break
		} else if err != nil {
			test.Fatal(err)
		}
		actual := serializeReadabilityLxmlTreeForTest(test, extractReadabilityLxml(fromNodes(sample.Nodes)))
		expected := serializeReadabilityLxmlTreeForTest(test, fromNodes(sample.ExpectedNodes))
		if actual != expected {
			mismatches++
			if mismatches <= 20 {
				position := 0
				for position < len(actual) && position < len(expected) && actual[position] == expected[position] {
					position++
				}
				test.Logf("%s: first difference at %d; expected %q; actual %q", sample.Name, position, expected[max(0, position-80):min(len(expected), position+160)], actual[max(0, position-80):min(len(actual), position+160)])
			}
		}
		checked++
	}
	if checked == 0 || mismatches != 0 {
		test.Fatalf("%d mismatches in %d exact-input Python candidates", mismatches, checked)
	}
	test.Logf("Matched all %d exact-input Python candidates", checked)
}
