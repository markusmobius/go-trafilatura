//go:build (re2_wasm || re2_cgo) && cgo

package regexp

import re2 "github.com/wasilibs/go-re2"

type Regexp struct {
	rx *re2.Regexp
}

func QuoteMeta(s string) string {
	return re2.QuoteMeta(s)
}

func Compile(str string) (*Regexp, error) {
	rx, err := re2.Compile(str)
	if err != nil {
		return nil, err
	}
	return &Regexp{rx}, nil
}

func MustCompile(str string) *Regexp {
	return &Regexp{re2.MustCompile(str)}
}

func (re *Regexp) Expand(dst []byte, template []byte, src []byte, match []int) []byte {
	return re.rx.Expand(dst, template, src, match)
}

func (re *Regexp) ExpandString(dst []byte, template string, src string, match []int) []byte {
	return re.rx.ExpandString(dst, template, src, match)
}

func (re *Regexp) Find(b []byte) []byte {
	return re.rx.Find(b)
}

func (re *Regexp) FindAll(b []byte, n int) [][]byte {
	return re.rx.FindAll(b, n)
}

func (re *Regexp) FindAllIndex(b []byte, n int) [][]int {
	return re.rx.FindAllIndex(b, n)
}

func (re *Regexp) FindAllString(s string, n int) []string {
	return re.rx.FindAllString(s, n)
}

func (re *Regexp) FindAllStringIndex(s string, n int) [][]int {
	return re.rx.FindAllStringIndex(s, n)
}

func (re *Regexp) FindAllStringSubmatch(s string, n int) [][]string {
	return re.rx.FindAllStringSubmatch(s, n)
}

func (re *Regexp) FindAllStringSubmatchIndex(s string, n int) [][]int {
	return re.rx.FindAllStringSubmatchIndex(s, n)
}

func (re *Regexp) FindAllSubmatch(b []byte, n int) [][][]byte {
	return re.rx.FindAllSubmatch(b, n)
}

func (re *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	return re.rx.FindAllSubmatchIndex(b, n)
}

func (re *Regexp) FindIndex(b []byte) (loc []int) {
	return re.rx.FindIndex(b)
}

func (re *Regexp) FindString(s string) string {
	return re.rx.FindString(s)
}

func (re *Regexp) FindStringIndex(s string) (loc []int) {
	return re.rx.FindStringIndex(s)
}

func (re *Regexp) FindStringSubmatch(s string) []string {
	return re.rx.FindStringSubmatch(s)
}

func (re *Regexp) FindStringSubmatchIndex(s string) []int {
	return re.rx.FindStringSubmatchIndex(s)
}

func (re *Regexp) FindSubmatch(b []byte) [][]byte {
	return re.rx.FindSubmatch(b)
}

func (re *Regexp) FindSubmatchIndex(b []byte) []int {
	return re.rx.FindSubmatchIndex(b)
}

func (re *Regexp) Longest() {
	re.rx.Longest()
}

func (re *Regexp) Match(b []byte) bool {
	return re.rx.Match(b)
}

func (re *Regexp) MatchString(s string) bool {
	return re.rx.MatchString(s)
}

func (re *Regexp) NumSubexp() int {
	return re.rx.NumSubexp()
}

func (re *Regexp) ReplaceAll(src, repl []byte) []byte {
	return re.rx.ReplaceAll(src, repl)
}

func (re *Regexp) ReplaceAllLiteral(src, repl []byte) []byte {
	return re.rx.ReplaceAllLiteral(src, repl)
}

func (re *Regexp) ReplaceAllLiteralString(src, repl string) string {
	return re.rx.ReplaceAllLiteralString(src, repl)
}

func (re *Regexp) ReplaceAllString(src, repl string) string {
	return re.rx.ReplaceAllString(src, repl)
}

func (re *Regexp) Split(s string, n int) []string {
	return re.rx.Split(s, n)
}

func (re *Regexp) String() string {
	return re.rx.String()
}

func (re *Regexp) SubexpIndex(name string) int {
	return re.rx.SubexpIndex(name)
}

func (re *Regexp) SubexpNames() []string {
	return re.rx.SubexpNames()
}
