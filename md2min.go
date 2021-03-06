package md2min

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	//	"regexp"
	"encoding/xml"
	"text/template"

	"github.com/russross/blackfriday"
)

type getID func() []byte

func h3Factory(mode string) (getID, getID) {
	var id uint64
	tmp := []byte("id-" + mode + "-")
	buf := make([][]byte, 0)
	return func() []byte {
			id++
			buf = append(buf, strconv.AppendUint(tmp, id, 10))
			return buf[id-1]
		}, func() []byte {
			return buf[id-1]
		}
}

type title []byte

func (t *title) init() {
	*t = make([]byte, 0)
}

func (t *title) addMore(bts []byte) {
	if *t == nil {
		*t = make([]byte, 0)
	}
	if len(bts) > 0 {
		*t = append(*t, bts...)
	}
}

func (t *title) has() bool {
	if len(*t) > 0 {
		return true
	}
	return false
}

func (t *title) reset() {
	if *t == nil {
		*t = make([]byte, 0)
	}
	*t = []byte(*t)[0:0]
}

func (t *title) String() string {
	return string(*t)
}

type a struct {
	Href   string `xml:"href,attr"`
	Title3 string `xml:",chardata"`
}

type li struct {
	//	Class string `xml:"class,attr"`
	A a `xml:"a"`
}

type listMenu struct {
	XMLName xml.Name `xml:"ul"`
	Lis     []li     `xml:"li"`
}

func (l *listMenu) init() {
	if l.Lis == nil {
		l.Lis = make([]li, 0)
	}
}

// MdContent composes the structure of the parsed markdown document.
type MdContent struct {
	level                 string
	title3                title
	listMenu              listMenu
	ListMenu              string
	Content               string
	ContentStyle          string
	MenuStyle             string
	MenuWrapStyle         string
	ScrollBar             string
	MenuLogo, ContentLogo string
}

func (md *MdContent) init(level string) *MdContent {
	md.level = level
	md.title3.init()
	md.listMenu.init()
	if md.level == "none" {
		md.ContentStyle = "margin: 0 auto;"
		md.MenuStyle = "background: #ffffff; overflow: hidden;"
		md.ContentLogo = `<div class="logo">Generated by <a href="https://github.com/fairlyblank/md2min">md2min</a></div>`
	} else {
		md.ContentStyle = "float: right;"
		md.MenuStyle = "background: #ffffff; overflow-x: hidden; overflow-y: scroll; width: 200px; height: 550px;"
		md.MenuWrapStyle = "width: 200px;"
		md.MenuLogo = `<div class="logo" style="font: 10px Helvetica, arial, freesans, clean, sans-serif; display: block; width: 200px; position: relative; top: 625px; right: 10px;">Generated by <a href="https://github.com/fairlyblank/md2min">md2min</a></div>`
		md.ScrollBar = `::-webkit-scrollbar {	width: 4px;	height: 8px; } ::-webkit-scrollbar-track-piece {	background-color: #ffffff;	border-radius: 4px; } ::-webkit-scrollbar-thumb {	background-color: #cfcfcf;	border-radius: 4px; }`
	}
	return md
}

func (md *MdContent) addToUl(id []byte) {
	a := &a{string(append([]byte{'#'}, id...)), strings.Trim(md.title3.String(), " \t\n")}
	md.listMenu.Lis = append(md.listMenu.Lis, li{*a})
	md.title3.reset()
}

func (md *MdContent) fillContentXML(output []byte) error {
	mode := md.level
	buf := bytes.NewBuffer(output)
	d := xml.NewDecoder(buf)
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	fdh := false
	getNewID, getLastID := h3Factory(mode)
	contBuf := bytes.NewBuffer(make([]byte, 0, len(output)*2))
	for token, err := d.RawToken(); err != io.EOF; token, err = d.RawToken() {
		if err != nil {
			return err
		}
		switch t := token.(type) {
		case xml.StartElement:
			if mode != "none" && t.Name.Local == mode {
				fdh = true
				md.title3.reset()
				t.Attr = append(t.Attr, xml.Attr{Name: xml.Name{Space: "", Local: "id"}, Value: string(getNewID())})
			}
			contBuf.WriteString("<")
			contBuf.WriteString(html.EscapeString(t.Name.Local))
			for _, a := range t.Attr {
				contBuf.WriteString(fmt.Sprintf(" %s=\"%s\"", a.Name.Local, a.Value))
			}
			contBuf.WriteString(">")
		case xml.EndElement:
			if mode != "none" && t.Name.Local == mode {
				fdh = false
				md.addToUl(getLastID())
			}
			contBuf.WriteString(fmt.Sprintf("</%s>", html.EscapeString(t.Name.Local)))
		case xml.CharData:
			if fdh {
				md.title3.addMore(t)
			}
			contBuf.WriteString(html.EscapeString(string(t)))
		case xml.ProcInst:
			contBuf.WriteString(fmt.Sprintf("<?%s %s>", t.Target, t.Inst))
		case xml.Directive:
			contBuf.WriteString(fmt.Sprintf("<!%s>", t))
		case xml.Comment:
			contBuf.WriteString(fmt.Sprintf("<!--%s-->", t))
		default:
			contBuf.WriteString("INVALID TOKEN")
		}
	}
	md.Content = contBuf.String()

	if mode != "none" {
		listMenu, err := xml.MarshalIndent(md.listMenu, "", "  ")
		if err != nil {
			return err
		}
		md.ListMenu = string(listMenu)
	}
	return nil
}

/*
func (md *mdContent) fillContentREG(output []byte) error {
	reg := regexp.MustCompile(`<h1>(.*)</h1>`)
	found := reg.FindSubmatch(output)
	if len(found) >= 2 {
		md.Title1 = found[1]
	}
//	fmt.Printf("%s\n", md.Title1)

	reg = regexp.MustCompile(`<h2>(.*)</h2>`)
	found = reg.FindSubmatch(output)
	if len(found) >= 2 {
		md.Title2 = found[1]
	}
//	fmt.Printf("%s\n", md.Title2)

	reg = regexp.MustCompile(`<h3>(.*)(</h3>)`)
	ind := reg.FindSubmatchIndex(output)
	fmt.Println(ind)
	dst := []byte{}
	dst = reg.Expand(dst, []byte("$1"), output, ind)
	fmt.Println(dst, string(dst))
//	output = reg.ReplaceAllFunc(output, func (old []byte) []byte {
//		fmt.Printf("%s, %s\n", string(old), reg.SubexpNames()[1])

//		return old
//	})

	return nil
}
*/

func (md *MdContent) fillContent(output []byte) error {
	err := md.fillContentXML(output)
	if err != nil {
		return err
	}

	/*	err := md.fillContentREG(output)
		if err != nil {
			return err
		}
	*/
	return nil
}

// New sets the level of navigation elements
func New(level string) *MdContent {
	md := new(MdContent).init(level)
	return md
}

// Parse is the workhorse that actually reads in the input bytes, applies template to the content and writes it to io.Writer wr.
func (md *MdContent) Parse(input []byte, wr io.Writer) error {
	output := blackfriday.MarkdownBasic(input)

	err := md.fillContent(output)
	if err != nil {
		return err
	}

	tmpl, err := template.New("tmpl").Parse(templContent)
	if err != nil {
		return err
	}
	err = tmpl.Execute(wr, md)
	if err != nil {
		return err
	}

	return nil
}
