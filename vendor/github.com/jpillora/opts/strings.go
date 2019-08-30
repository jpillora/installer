package opts

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func camel2const(s string) string {
	b := strings.Builder{}
	var c rune
	start := 0
	end := 0
	for end, c = range s {
		if c >= 'A' && c <= 'Z' {
			//uppercase all prior letters and add an underscore
			if start < end {
				b.WriteString(strings.ToTitle(s[start:end] + "_"))
				start = end
			}
		}
	}
	//write remaining string
	b.WriteString(strings.ToTitle(s[start : end+1]))
	return strings.ReplaceAll(b.String(), "-", "_")
}

func nletters(r rune, n int) string {
	str := make([]rune, n)
	for i := range str {
		str[i] = r
	}
	return string(str)
}

func constrain(str string, maxWidth int) string {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		words := strings.Split(line, " ")
		width := 0
		for i, w := range words {
			remain := maxWidth - width
			wordWidth := len(w) + 1 //+space
			width += wordWidth
			overflow := width > maxWidth
			fits := width-maxWidth > remain
			if overflow && fits {
				width = wordWidth
				w = "\n" + w
			}
			words[i] = w
		}
		lines[i] = strings.Join(words, " ")
	}
	return strings.Join(lines, "\n")
}

//borrowed from https://github.com/huandu/xstrings/blob/master/convert.go#L77
func camel2dash(str string) string {
	if len(str) == 0 {
		return ""
	}
	buf := &bytes.Buffer{}
	var prev, r0, r1 rune
	var size int
	r0 = '-'
	for len(str) > 0 {
		prev = r0
		r0, size = utf8.DecodeRuneInString(str)
		str = str[size:]
		switch {
		case r0 == utf8.RuneError:
			buf.WriteByte(byte(str[0]))
		case unicode.IsUpper(r0):
			if prev != '-' {
				buf.WriteRune('-')
			}
			buf.WriteRune(unicode.ToLower(r0))
			if len(str) == 0 {
				break
			}
			r0, size = utf8.DecodeRuneInString(str)
			str = str[size:]
			if !unicode.IsUpper(r0) {
				buf.WriteRune(r0)
				break
			}
			// find next non-upper-case character and insert `_` properly.
			// it's designed to convert `HTTPServer` to `http_server`.
			// if there are more than 2 adjacent upper case characters in a word,
			// treat them as an abbreviation plus a normal word.
			for len(str) > 0 {
				r1 = r0
				r0, size = utf8.DecodeRuneInString(str)
				str = str[size:]
				if r0 == utf8.RuneError {
					buf.WriteRune(unicode.ToLower(r1))
					buf.WriteByte(byte(str[0]))
					break
				}
				if !unicode.IsUpper(r0) {
					if r0 == '-' || r0 == ' ' || r0 == '_' {
						r0 = '-'
						buf.WriteRune(unicode.ToLower(r1))
					} else {
						buf.WriteRune('-')
						buf.WriteRune(unicode.ToLower(r1))
						buf.WriteRune(r0)
					}
					break
				}
				buf.WriteRune(unicode.ToLower(r1))
			}
			if len(str) == 0 || r0 == '-' {
				buf.WriteRune(unicode.ToLower(r0))
				break
			}
		default:
			if r0 == ' ' || r0 == '_' {
				r0 = '-'
			}
			buf.WriteRune(r0)
		}
	}
	return buf.String()
}

func camel2title(str string) string {
	dash := camel2dash(str)
	title := []rune(dash)
	for i, r := range title {
		if r == '-' {
			r = ' '
		}
		if i == 0 {
			r = unicode.ToUpper(r)
		}
		title[i] = r
	}
	return string(title)
}

//borrowed from https://raw.githubusercontent.com/jinzhu/inflection/master/inflections.go
var getSingular = func() func(str string) string {
	type inflection struct {
		regexp  *regexp.Regexp
		replace string
	}
	// Regular is a regexp find replace inflection
	type Regular struct {
		find    string
		replace string
	}
	// Irregular is a hard replace inflection,
	// containing both singular and plural forms
	type Irregular struct {
		singular string
		plural   string
	}
	var singularInflections = []Regular{
		{"s$", ""},
		{"(ss)$", "${1}"},
		{"(n)ews$", "${1}ews"},
		{"([ti])a$", "${1}um"},
		{"((a)naly|(b)a|(d)iagno|(p)arenthe|(p)rogno|(s)ynop|(t)he)(sis|ses)$", "${1}sis"},
		{"(^analy)(sis|ses)$", "${1}sis"},
		{"([^f])ves$", "${1}fe"},
		{"(hive)s$", "${1}"},
		{"(tive)s$", "${1}"},
		{"([lr])ves$", "${1}f"},
		{"([^aeiouy]|qu)ies$", "${1}y"},
		{"(s)eries$", "${1}eries"},
		{"(m)ovies$", "${1}ovie"},
		{"(c)ookies$", "${1}ookie"},
		{"(x|ch|ss|sh)es$", "${1}"},
		{"^(m|l)ice$", "${1}ouse"},
		{"(bus)(es)?$", "${1}"},
		{"(o)es$", "${1}"},
		{"(shoe)s$", "${1}"},
		{"(cris|test)(is|es)$", "${1}is"},
		{"^(a)x[ie]s$", "${1}xis"},
		{"(octop|vir)(us|i)$", "${1}us"},
		{"(alias|status)(es)?$", "${1}"},
		{"^(ox)en", "${1}"},
		{"(vert|ind)ices$", "${1}ex"},
		{"(matr)ices$", "${1}ix"},
		{"(quiz)zes$", "${1}"},
		{"(database)s$", "${1}"},
	}
	var irregularInflections = []Irregular{
		{"person", "people"},
		{"man", "men"},
		{"child", "children"},
		{"sex", "sexes"},
		{"move", "moves"},
		{"mombie", "mombies"},
	}
	var uncountableInflections = []string{"equipment", "information", "rice", "money", "species", "series", "fish", "sheep", "jeans", "police"}
	var compiledSingularMaps []inflection
	compiledSingularMaps = []inflection{}
	for _, uncountable := range uncountableInflections {
		inf := inflection{
			regexp:  regexp.MustCompile("^(?i)(" + uncountable + ")$"),
			replace: "${1}",
		}
		compiledSingularMaps = append(compiledSingularMaps, inf)
	}
	for _, value := range irregularInflections {
		infs := []inflection{
			inflection{regexp: regexp.MustCompile(strings.ToUpper(value.plural) + "$"), replace: strings.ToUpper(value.singular)},
			inflection{regexp: regexp.MustCompile(strings.Title(value.plural) + "$"), replace: strings.Title(value.singular)},
			inflection{regexp: regexp.MustCompile(value.plural + "$"), replace: value.singular},
		}
		compiledSingularMaps = append(compiledSingularMaps, infs...)
	}
	for i := len(singularInflections) - 1; i >= 0; i-- {
		value := singularInflections[i]
		infs := []inflection{
			inflection{regexp: regexp.MustCompile(strings.ToUpper(value.find)), replace: strings.ToUpper(value.replace)},
			inflection{regexp: regexp.MustCompile(value.find), replace: value.replace},
			inflection{regexp: regexp.MustCompile("(?i)" + value.find), replace: value.replace},
		}
		compiledSingularMaps = append(compiledSingularMaps, infs...)
	}
	return func(str string) string {
		for _, inflection := range compiledSingularMaps {
			if inflection.regexp.MatchString(str) {
				return inflection.regexp.ReplaceAllString(str, inflection.replace)
			}
		}
		return str
	}
}()

type kv struct {
	m map[string]string
}

func (kv *kv) keys() []string {
	if kv == nil {
		return nil
	}
	ks := []string{}
	for k := range kv.m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func (kv *kv) take(k string) (string, bool) {
	if kv == nil {
		return "", false
	}
	v, ok := kv.m[k]
	if ok {
		delete(kv.m, k)
	}
	return v, ok
}

func newKV(s string) *kv {
	m := map[string]string{}
	key := ""
	keying := true
	sb := strings.Builder{}
	commit := func() {
		s := sb.String()
		if key == "" && s == "" {
			return
		} else if key == "" {
			m[s] = ""
		} else {
			m[key] = s
			key = ""
		}
		sb.Reset()
	}
	for _, r := range s {
		//key done
		if keying && sb.Len() == 0 && r == ' ' {
			continue //drop leading spaces
		}
		if keying && r == '=' {
			key = sb.String()
			sb.Reset()
			keying = false
			continue
		}
		//go to next
		if r == ',' {
			commit()
			keying = true
			continue
		}
		//write to builder
		sb.WriteRune(r)
	}
	//write last key=value
	commit()
	return &kv{m: m}
}
