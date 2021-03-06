package creeper

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/moxar/arithmetic"
)

var(
	rx_funName = regexp.MustCompile(`^[a-z$][a-zA-Z]{0,15}`)
)

type Fun struct {
	Raw string
	Node *Node

	Name string
	Params []string

	Selection *goquery.Selection
	Result string

	PrevFun *Fun
	NextFun *Fun
}

func (f *Fun) Append(s string) (*Fun, *Fun) {
	f.NextFun = ParseFun(f.Node, s)
	f.NextFun.PrevFun = f
	return f, f.NextFun
}

func PowerfulFind(s *goquery.Selection, q string) *goquery.Selection {
	rx_selectPseudoEq := regexp.MustCompile(`:eq\(\d+\)`)
	if rx_selectPseudoEq.MatchString(q) {
		rs := rx_selectPseudoEq.FindAllStringIndex(q, -1)
		sel := s
		for _, r := range rs {
			iStr := q[r[0]+4:r[1]-1]
			i64, _ := strconv.ParseInt(iStr, 10, 32)
			i := int(i64)
			sq := q[:r[0]]
			q = strings.TrimSpace(q[r[1]:])
			sel = sel.Find(sq).Eq(i)
		}
		if len(q) > 0 {
			sel = sel.Find(q)
		}
		return sel
	} else {
		return s.Find(q)
	}
}

func (f *Fun) InitSelector() {
	if f.Node.IsArray || f.Node.IndentLen == 0 || f.Node.Page != nil {
		r := strings.NewReader(f.Node.Page.Body())
		doc, _ := goquery.NewDocumentFromReader(r)
		bud := PowerfulFind(doc.Selection, f.Params[0])
		if len(bud.Nodes) > f.Node.Index {
			f.Selection = PowerfulFind(doc.Selection, f.Params[0]).Eq(f.Node.Index)
		} else {
			f.Node.Page.Inc()
			f.Node.Reset()
			f.InitSelector()
			return
		}
	} else {
		f.Node.ParentNode.Fun.Invoke()
		f.Selection = PowerfulFind(f.Node.ParentNode.Fun.Selection, f.Params[0]).Eq(f.Node.Index)
	}
}

func (f *Fun) Invoke() string {
	switch f.Name {
	case "$": f.InitSelector()
	case "attr": f.Result, _ = f.PrevFun.Selection.Attr(f.Params[0])
	case "text": f.Result = f.PrevFun.Selection.Text()
	case "html": f.Result, _ = f.PrevFun.Selection.Html()
	case "outerHTML": f.Result, _ = goquery.OuterHtml(f.PrevFun.Selection)
	case "style": f.Result, _ = f.PrevFun.Selection.Attr("style")
	case "href": f.Result, _ = f.PrevFun.Selection.Attr("href")
	case "src": f.Result, _ = f.PrevFun.Selection.Attr("src")
	case "calc":
		v, _ := arithmetic.Parse(f.PrevFun.Result)
		n, _ := arithmetic.ToFloat(v)
		prec := 2
		if len(f.Params) > 0 {
			i64, _ := strconv.ParseInt(f.Params[0], 10, 32)
			prec = int(i64)
		}
		f.Result = strconv.FormatFloat(n, 'g', prec, 64)
	case "expand":
		rx := regexp.MustCompile(f.Params[0])
		src := f.PrevFun.Result
		dst := []byte{}
		m := rx.FindStringSubmatchIndex(src)
		s := rx.ExpandString(dst, f.Params[1], src, m)
		f.Result = string(s)
	case "match":
		rx := regexp.MustCompile(f.Params[0])
		rs := rx.FindAllStringSubmatch(f.PrevFun.Result, -1)
		if len(rs) > 0 && len(rs[0]) > 1 {
			f.Result = rs[0][1]
		}
	}
	if f.NextFun != nil {
		return f.NextFun.Invoke()
	} else {
		return f.Result
	}
}


func ParseFun(n *Node, s string) *Fun {
	fun := new(Fun)
	fun.Node = n
	fun.Raw = s

	sa := rx_funName.FindAllString(s, -1)
	fun.Name = sa[0]
	ls := s[len(sa[0]):]
	ps := []string{}
	p, pl := parseParams(ls)
	for k := range p {
		ps = append(ps, k)
	}
	if len(ps) > 0 {
		fun.Params = ps
	}
	ls = ls[pl+1:]
	if len(ls) > 0 {
		ls = ls[1:]
		fun.Append(ls)
	}

	return fun
}