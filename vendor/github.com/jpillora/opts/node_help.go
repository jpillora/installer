package opts

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
)

//data is only used for templating below
type data struct {
	datum        //data is also a datum
	FlagGroups   []*datumGroup
	Args         []*datum
	Cmds         []*datum
	Order        []string
	Parents      string
	Version      string
	Summary      string
	Repo, Author string
	ErrMsg       string
}

type datum struct {
	Name, Help, Pad string //Pad is Opt.padWidth many spaces
}

type datumGroup struct {
	Name  string
	Flags []*datum
}

//DefaultOrder defines which templates get rendered in which order.
//This list is referenced in the "help" template below.
var DefaultOrder = []string{
	"usage",
	"summary",
	"args",
	"flaggroups",
	"cmds",
	"author",
	"version",
	"repo",
	"errmsg",
}

func defaultOrder() []string {
	order := make([]string, len(DefaultOrder))
	copy(order, DefaultOrder)
	return order
}

//DefaultTemplates define a set of individual templates
//that get rendered in DefaultOrder. You can replace templates or insert templates before or after existing
//templates using the DocSet, DocBefore and DocAfter methods. For example, you can insert a string after the
//usage text with:
//
//  DocAfter("usage", "this is a string, and if it is very long, it will be wrapped")
//
//The entire help text is simply the "help" template listed below, which renders a set of these templates in
//the order defined above. All templates can be referenced using the keys in this map:
var DefaultTemplates = map[string]string{
	"help":          `{{ $root := . }}{{range $t := .Order}}{{ templ $t $root }}{{end}}`,
	"usage":         `Usage: {{.Name }} [options]{{template "usageargs" .}}{{template "usagecmd" .}}` + "\n",
	"usageargs":     `{{range .Args}} {{.Name}}{{end}}`,
	"usagecmd":      `{{if .Cmds}} <command>{{end}}`,
	"extradefault":  `{{if .}}default {{.}}{{end}}`,
	"extraenv":      `{{if .}}env {{.}}{{end}}`,
	"extramultiple": `{{if .}}allows multiple{{end}}`,
	"summary":       "{{if .Summary}}\n{{ .Summary }}\n{{end}}",
	"args":          `{{range .Args}}{{template "arg" .}}{{end}}`,
	"arg":           "{{if .Help}}\n{{.Help}}\n{{end}}",
	"flaggroups":    `{{ range $g := .FlagGroups}}{{template "flaggroup" $g}}{{end}}`,
	"flaggroup": "{{if .Flags}}\n{{if .Name}}{{.Name}} options{{else}}Options{{end}}:\n" +
		`{{ range $f := .Flags}}{{template "flag" $f}}{{end}}{{end}}`,
	"flag":    `{{.Name}}{{if .Help}}{{.Pad}}{{.Help}}{{end}}` + "\n",
	"cmds":    "{{if .Cmds}}\nCommands:\n" + `{{ range $sub := .Cmds}}{{template "cmd" $sub}}{{end}}{{end}}`,
	"cmd":     "· {{ .Name }}{{if .Help}}{{.Pad}}  {{ .Help }}{{end}}\n",
	"version": "{{if .Version}}\nVersion:\n{{.Pad}}{{.Version}}\n{{end}}",
	"repo":    "{{if .Repo}}\nRead more:\n{{.Pad}}{{.Repo}}\n{{end}}",
	"author":  "{{if .Author}}\nAuthor:\n{{.Pad}}{{.Author}}\n{{end}}",
	"errmsg":  "{{if .ErrMsg}}\nError:\n{{.Pad}}{{.ErrMsg}}\n{{end}}",
}

var trailingSpaces = regexp.MustCompile(`(?m)\ +$`)

//Help renders the help text as a string
func (o *node) Help() string {
	h, err := renderHelp(o)
	if err != nil {
		log.Fatalf("render help failed: %s", err)
	}
	return h
}

func renderHelp(o *node) (string, error) {
	var err error
	//add default templates
	for name, str := range DefaultTemplates {
		if _, ok := o.templates[name]; !ok {
			o.templates[name] = str
		}
	}
	//prepare templates
	t := template.New(o.name)
	t = t.Funcs(map[string]interface{}{
		//reimplementation of "template" except with dynamic name
		"templ": func(name string, data interface{}) (string, error) {
			b := &bytes.Buffer{}
			err = t.ExecuteTemplate(b, name, data)
			if err != nil {
				return "", err
			}
			return b.String(), nil
		},
	})
	//parse all templates and "define" themselves as nested templates
	for name, str := range o.templates {
		t, err = t.Parse(fmt.Sprintf(`{{define "%s"}}%s{{end}}`, name, str))
		if err != nil {
			return "", fmt.Errorf("template '%s': %s", name, err)
		}
	}
	//convert node into template data
	tf, err := convert(o)
	if err != nil {
		return "", fmt.Errorf("node convert: %s", err)
	}
	//execute all templates
	b := &bytes.Buffer{}
	err = t.ExecuteTemplate(b, "help", tf)
	if err != nil {
		return "", fmt.Errorf("template execute: %s", err)
	}
	out := b.String()
	if o.padAll {
		/*
			"foo
			bar"
			becomes
			"
			  foo
			  bar
			"
		*/
		lines := strings.Split(out, "\n")
		for i, l := range lines {
			lines[i] = tf.Pad + l
		}
		out = "\n" + strings.Join(lines, "\n") + "\n"
	}
	out = trailingSpaces.ReplaceAllString(out, "")
	return out, nil
}

func convert(o *node) (*data, error) {
	names := []string{}
	curr := o
	for curr != nil {
		names = append([]string{curr.name}, names...)
		curr = curr.parent
	}
	name := strings.Join(names, " ")
	args := make([]*datum, len(o.args))
	for i, arg := range o.args {
		//arguments are required
		n := "<" + arg.name + ">"
		//unless...
		if arg.slice {
			p := []string{arg.name, arg.name}
			for i, n := range p {
				if i < arg.min {
					//still required
					n = "<" + n + ">"
				} else {
					//optional!
					n = "[" + n + "]"
				}
				p[i] = n
			}
			n = strings.Join(p, " ") + " ..."
		}
		args[i] = &datum{
			Name: n,
			Help: constrain(arg.help, o.lineWidth),
		}
	}
	flagGroups := make([]*datumGroup, len(o.flagGroups))
	//initialise and calculate padding
	max := 0
	pad := nletters(' ', o.padWidth)
	for i, g := range o.flagGroups {
		dg := &datumGroup{
			Name:  g.name,
			Flags: make([]*datum, len(g.flags)),
		}
		flagGroups[i] = dg
		for i, item := range g.flags {
			to := &datum{Pad: pad}
			to.Name = "--" + item.name
			if item.shortName != "" {
				to.Name += ", -" + item.shortName
			}
			l := len(to.Name)
			//max shared across ALL groups
			if l > max {
				max = l
			}
			dg.Flags[i] = to
		}
	}
	//get item help, with optional default values and env names and
	//constrain to a specific line width
	extras := make([]*template.Template, 3)
	keys := []string{"default", "env", "multiple"}
	for i, k := range keys {
		t, err := template.New("").Parse(o.templates["extra"+k])
		if err != nil {
			return nil, fmt.Errorf("template extra%s: %s", k, err)
		}
		extras[i] = t
	}
	//calculate...
	padsInOption := o.padWidth
	optionNameWidth := max + padsInOption
	spaces := nletters(' ', optionNameWidth)
	helpWidth := o.lineWidth - optionNameWidth
	//go back and render each option using calculated values
	for i, dg := range flagGroups {
		for j, to := range dg.Flags {
			//pad all option names to be the same length
			to.Name += spaces[:max-len(to.Name)]
			//constrain help text
			item := o.flagGroups[i].flags[j]
			//render flag help string
			vals := []interface{}{item.defstr, item.envName, item.slice}
			outs := []string{}
			for i, v := range vals {
				b := strings.Builder{}
				if err := extras[i].Execute(&b, v); err != nil {
					return nil, err
				}
				if b.Len() > 0 {
					outs = append(outs, b.String())
				}
			}
			help := item.help
			extra := strings.Join(outs, ", ")
			if help == "" {
				help = extra
			} else if extra != "" {
				help += " (" + extra + ")"
			}
			help = constrain(help, helpWidth)
			//align each row after the flag
			lines := strings.Split(help, "\n")
			for i, l := range lines {
				if i > 0 {
					lines[i] = spaces + l
				}
			}
			to.Help = strings.Join(lines, "\n")
		}
	}
	//commands
	max = 0
	for _, s := range o.cmds {
		if l := len(s.name); l > max {
			max = l
		}
	}
	subs := make([]*datum, len(o.cmds))
	i := 0
	for _, s := range o.cmds {
		h := s.help
		if h == "" {
			h = s.summary
		}

		subs[i] = &datum{
			Name: s.name,
			Help: h,
			Pad:  nletters(' ', max-len(s.name)),
		}
		i++
	}
	//convert error to string
	err := ""
	if o.err != nil {
		err = o.err.Error()
	}
	return &data{
		datum: datum{
			Name: name,
			Help: o.help,
			Pad:  pad,
		},
		Args:       args,
		FlagGroups: flagGroups,
		Cmds:       subs,
		Order:      o.order,
		Version:    o.version,
		Summary:    constrain(o.summary, o.lineWidth),
		Repo:       o.repo,
		Author:     o.author,
		ErrMsg:     err,
	}, nil
}
