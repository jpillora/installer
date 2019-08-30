package opts

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
)

//Parse with os.Args
func (n *node) Parse() ParsedOpts {
	return n.ParseArgs(os.Args)
}

//ParseArgs with the provided arguments
func (n *node) ParseArgs(args []string) ParsedOpts {
	//shell-completion?
	if cl := os.Getenv("COMP_LINE"); n.complete && cl != "" {
		args := strings.Split(cl, " ")
		n.parse(args) //ignore error
		if ok := n.doCompletion(); !ok {
			os.Exit(1)
		}
		os.Exit(0)
	}
	//use built state to perform parse
	if err := n.parse(args); err != nil {
		//expected exit (0) print message as-is
		if ee, ok := err.(exitError); ok {
			fmt.Fprint(os.Stderr, string(ee))
			os.Exit(0)
		}
		//unexpected exit (1) print message to programmer
		if ae, ok := err.(authorError); ok {
			fmt.Fprintf(os.Stderr, "opts usage error: %s\n", ae)
			os.Exit(1)
		}
		//unexpected exit (1) embed message in help to user
		n.err = err
		fmt.Fprintf(os.Stderr, n.Help())
		os.Exit(1)
	}
	//success
	return n
}

//parse validates and initialises all internal items
//and then passes the args through, setting them items required
func (n *node) parse(args []string) error {
	//return the stored error
	if n.err != nil {
		return n.err
	}
	//root node? take program from the arg list (assumes os.Args format)
	if n.parent == nil {
		prog := ""
		if len(args) > 0 {
			prog = args[0]
			args = args[1:]
		}
		//find default name for root-node
		if n.item.name == "" {
			if exe, err := os.Executable(); err == nil && exe != "" {
				//TODO: use filepath.EvalSymlinks first?
				_, n.item.name = path.Split(exe)
			} else if prog != "" {
				_, n.item.name = path.Split(prog)
			}
			//looks like weve been go-run, use package name?
			if n.item.name == "main" {
				if pkgPath := n.item.val.Type().PkgPath(); pkgPath != "" {
					_, n.item.name = path.Split(pkgPath)
				}
			}
		}
	}
	//add this node and its fields (recurses if has sub-commands)
	if err := n.addStructFields(defaultGroup, n.item.val); err != nil {
		return err
	}
	//add user provided flagsets, will error if there is a naming collision
	if err := n.addFlagsets(); err != nil {
		return err
	}
	//add help, version, etc flags
	if err := n.addInternalFlags(); err != nil {
		return err
	}
	//find defaults from config's package
	n.setPkgDefaults()
	//add shortnames where possible
	for _, item := range n.flags() {
		if item.shortName == "" && len(item.name) >= 2 {
			if s := item.name[0:1]; !n.flagNames[s] {
				item.shortName = s
				n.flagNames[s] = true
			}
		}
	}
	//create a new flagset, and link each item
	flagset := flag.NewFlagSet(n.item.name, flag.ContinueOnError)
	flagset.SetOutput(ioutil.Discard)
	for _, item := range n.flags() {
		flagset.Var(item, item.name, "")
		if sn := item.shortName; sn != "" {
			flagset.Var(item, sn, "")
		}
	}
	if err := flagset.Parse(args); err != nil {
		//insert flag errors into help text
		n.err = err
		n.internalOpts.Help = true
	}
	//handle help, version, install/uninstall
	if n.internalOpts.Help {
		return exitError(n.Help())
	} else if n.internalOpts.Version {
		return exitError(n.version)
	} else if n.internalOpts.Install {
		return n.manageCompletion(false)
	} else if n.internalOpts.Uninstall {
		return n.manageCompletion(true)
	}
	//first round of defaults, applying env variables where necesseary
	for _, item := range n.flags() {
		k := item.envName
		if item.set() || k == "" {
			continue
		}
		v := os.Getenv(k)
		if v == "" {
			continue
		}
		err := item.Set(v)
		if err != nil {
			return fmt.Errorf("flag '%s' cannot set invalid env var (%s): %s", item.name, k, err)
		}
	}
	//second round, unmarshal directly into the struct, overwrites envs and flags
	if c := n.internalOpts.ConfigPath; c != "" {
		b, err := ioutil.ReadFile(c)
		if err == nil {
			v := n.val.Addr().Interface() //*struct
			err = json.Unmarshal(b, v)
			if err != nil {
				return fmt.Errorf("Invalid config file: %s", err)
			}
		}
	}
	//get remaining args after extracting flags
	remaining := flagset.Args()
	i := 0
	for {
		if len(n.args) == i {
			break
		}
		item := n.args[i]
		if len(remaining) == 0 && !item.set() && !item.slice {
			return fmt.Errorf("argument '%s' is missing", item.name)
		}
		if len(remaining) == 0 {
			break
		}
		s := remaining[0]
		if err := item.Set(s); err != nil {
			return fmt.Errorf("argument '%s' is invalid: %s", item.name, err)
		}
		remaining = remaining[1:]
		//use next arg?
		if !item.slice {
			i++
		}
	}
	//check min
	for _, item := range n.args {
		if item.slice && item.sets < item.min {
			return fmt.Errorf("argument '%s' has too few args (%d/%d)", item.name, item.sets, item.min)
		}
		if item.slice && item.max != 0 && item.sets > item.max {
			return fmt.Errorf("argument '%s' has too many args (%d/%d)", item.name, item.sets, item.max)
		}
	}
	//use command? next arg can optionally match command
	if len(n.cmds) > 0 && len(remaining) > 0 {
		a := remaining[0]
		//matching command, use it
		if sub, exists := n.cmds[a]; exists {
			//store matched command
			n.cmd = sub
			//user wants command name to be set on their struct?
			if n.cmdname != nil {
				*n.cmdname = a
			}
			//tail recurse! if only...
			return sub.parse(remaining[1:])
		}
	}
	//we *should* have consumed all args at this point.
	//this prevents:  ./foo --bar 42 -z 21 ping --pong 7
	//where --pong 7 is ignored
	if len(remaining) != 0 {
		return fmt.Errorf("Unexpected arguments: %s", strings.Join(remaining, " "))
	}
	return nil
}

func (n *node) addStructFields(group string, sv reflect.Value) error {
	if sv.Kind() != reflect.Struct {
		return n.errorf("opts: %s should be a pointer to a struct (got %s)", sv.Type().Name(), sv.Kind())
	}
	for i := 0; i < sv.NumField(); i++ {
		sf := sv.Type().Field(i)
		val := sv.Field(i)
		if err := n.addStructField(group, sf, val); err != nil {
			return fmt.Errorf("field '%s' %s", sf.Name, err)
		}
	}
	return nil
}

func (n *node) addStructField(group string, sf reflect.StructField, val reflect.Value) error {
	kv := newKV(sf.Tag.Get("opts"))
	help := sf.Tag.Get("help")
	mode := sf.Tag.Get("type") //legacy versions of this package used "type"
	if m := sf.Tag.Get("mode"); m != "" {
		mode = m //allow "mode" to be used directly, undocumented!
	}
	if err := n.addKVField(kv, sf.Name, help, mode, group, val); err != nil {
		return err
	}
	if ks := kv.keys(); len(ks) > 0 {
		return fmt.Errorf("unused opts keys: %s", strings.Join(ks, ", "))
	}
	return nil
}

func (n *node) addKVField(kv *kv, fName, help, mode, group string, val reflect.Value) error {
	//ignore unaddressed/unexported fields
	if !val.CanSet() {
		return nil
	}
	//parse key-values
	//ignore `opts:"-"`
	if _, ok := kv.take("-"); ok {
		return nil
	}
	//get field name and mode
	name, _ := kv.take("name")
	if name == "" {
		//default to struct field name
		name = camel2dash(fName)
		//slice? use singular, usage of
		//Foos []string should be: --foo bar --foo bazz
		if val.Type().Kind() == reflect.Slice {
			name = getSingular(name)
		}
	}
	//new kv mode supercede legacy mode
	if t, ok := kv.take("mode"); ok {
		mode = t
	}
	//default opts mode from go type
	if mode == "" {
		switch val.Type().Kind() {
		case reflect.Struct:
			mode = "embedded"
		default:
			mode = "flag"
		}
	}
	//use the specified group
	if g, ok := kv.take("group"); ok {
		group = g
	}
	//special cases
	if mode == "embedded" {
		return n.addStructFields(group, val) //recurse!
	}
	if mode == "cmdname" {
		return n.setCmdName(val)
	}
	//new kv help defs supercede legacy defs
	if h, ok := kv.take("help"); ok {
		help = h
	}
	//inline sub-command
	if mode == "cmd" {
		return n.addInlineCmd(name, help, val)
	}
	//from this point, we must have a flag or an arg
	i, err := newItem(val)
	if err != nil {
		return err
	}
	i.mode = mode
	i.name = name
	i.help = help
	//insert either as flag or as argument
	switch mode {
	case "flag":
		//set default text
		if d, ok := kv.take("default"); ok {
			i.defstr = d
		} else if !i.slice {
			v := val.Interface()
			t := val.Type()
			z := reflect.Zero(t)
			zero := reflect.DeepEqual(v, z.Interface())
			if !zero {
				i.defstr = fmt.Sprintf("%v", v)
			}
		}
		if e, ok := kv.take("env"); ok || n.useEnv {
			explicit := true
			if e == "" {
				explicit = false
				e = camel2const(i.name)
			}
			_, set := n.envNames[e]
			if set && explicit {
				return n.errorf("env name '%s' already in use", e)
			}
			if !set {
				n.envNames[e] = true
				i.envName = e
				i.useEnv = true
			}
		}
		//cannot have duplicates
		if n.flagNames[name] {
			return n.errorf("flag '%s' already exists", name)
		}
		//flags can also set short names
		if short, ok := kv.take("short"); ok {
			if len(short) != 1 {
				return n.errorf("short name '%s' on flag '%s' must be a single character", short, name)
			}
			if n.flagNames[short] {
				return n.errorf("short name '%s' on flag '%s' already exists", short, name)
			}
			n.flagNames[short] = true
			i.shortName = short
		}
		//add to this command's flags
		n.flagNames[name] = true
		g := n.flagGroup(group)
		g.flags = append(g.flags, i)
	case "arg":
		//minimum number of items
		if i.slice {
			if m, ok := kv.take("min"); ok {
				min, err := strconv.Atoi(m)
				if err != nil {
					return n.errorf("min not an integer")
				}
				i.min = min
			}
			if m, ok := kv.take("max"); ok {
				max, err := strconv.Atoi(m)
				if err != nil {
					return n.errorf("max not an integer")
				}
				i.max = max
			}
		}
		//validations
		if group != "" {
			return n.errorf("args cannot be placed into a group")
		}
		if len(n.cmds) > 0 {
			return n.errorf("args and commands cannot be used together")
		}
		for _, item := range n.args {
			if item.slice {
				return n.errorf("cannot come after arg list '%s'", item.name)
			}
		}
		//add to this command's arguments
		n.args = append(n.args, i)
	default:
		return fmt.Errorf("invalid opts mode '%s'", mode)
	}
	return nil
}

func (n *node) setCmdName(val reflect.Value) error {
	if n.cmdname != nil {
		return n.errorf("cmdname set twice")
	} else if val.Type().Kind() != reflect.String {
		return n.errorf("cmdname type must be string")
	} else if !val.CanAddr() {
		return n.errorf("cannot address cmdname string")
	}
	n.cmdname = val.Addr().Interface().(*string)
	return nil
}

func (n *node) addInlineCmd(name, help string, val reflect.Value) error {
	vt := val.Type()
	if vt.Kind() == reflect.Ptr {
		vt = vt.Elem()
	}
	if vt.Kind() != reflect.Struct {
		return errors.New("inline commands 'type=cmd' must be structs")
	} else if !val.CanAddr() {
		return errors.New("cannot address inline command")
	}
	//if nil ptr, auto-create new struct
	if val.Kind() == reflect.Ptr && val.IsNil() {
		val.Set(reflect.New(vt))
	}
	//ready!
	if _, ok := n.cmds[name]; ok {
		return n.errorf("command already exists: %s", name)
	}
	sub := newNode(val)
	sub.Name(name)
	sub.help = help
	sub.Summary(help)
	sub.parent = n
	n.cmds[name] = sub
	return nil
}

func (n *node) addInternalFlags() error {
	type internal struct{ name, help, group string }
	g := reflect.ValueOf(&n.internalOpts).Elem()
	flags := []internal{}
	if n.version != "" {
		flags = append(flags,
			internal{name: "Version", help: "display version"},
		)
	}
	flags = append(flags,
		internal{name: "Help", help: "display help"},
	)
	if n.complete {
		s := "shell"
		if bs := path.Base(os.Getenv("SHELL")); bs == "bash" || bs == "fish" || bs == "zsh" {
			s = bs
		}
		flags = append(flags,
			internal{name: "Install", help: "install " + s + "-completion", group: "Completion"},
			internal{name: "Uninstall", help: "uninstall " + s + "-completion", group: "Completion"},
		)
	}
	if n.userCfgPath {
		flags = append(flags,
			internal{name: "ConfigPath", help: "path to a JSON file"},
		)
	}
	for _, i := range flags {
		sf, _ := g.Type().FieldByName(i.name)
		val := g.FieldByName(i.name)
		if err := n.addKVField(nil, sf.Name, i.help, "flag", i.group, val); err != nil {
			return fmt.Errorf("error adding internal flag: %s: %s", i.name, err)
		}
	}
	return nil
}

func (n *node) addFlagsets() error {
	//add provided flag sets
	for _, fs := range n.flagsets {
		var err error
		//add all flags in each set
		fs.VisitAll(func(f *flag.Flag) {
			//convert into item
			val := reflect.ValueOf(f.Value)
			i, er := newItem(val)
			if er != nil {
				err = n.errorf("imported flag '%s': %s", f.Name, er)
				return
			}
			i.name = f.Name
			i.defstr = f.DefValue
			i.help = f.Usage
			//cannot have duplicates
			if n.flagNames[i.name] {
				err = n.errorf("imported flag '%s' already exists", i.name)
				return
			}
			//ready!
			g := n.flagGroup("")
			g.flags = append(g.flags, i)
			n.flagNames[i.name] = true
			//convert f into a black hole
			f.Value = noopValue
		})
		//fail with last error
		if err != nil {
			return err
		}
		fs.Init(fs.Name(), flag.ContinueOnError)
		fs.SetOutput(ioutil.Discard)
		fs.Parse([]string{}) //ensure this flagset returns Parsed() => true
	}
	return nil
}

func (n *node) setPkgDefaults() {
	//attempt to infer package name, repo, author
	configStruct := n.item.val.Type()
	pkgPath := configStruct.PkgPath()
	parts := strings.Split(pkgPath, "/")
	if len(parts) >= 3 {
		if n.authorInfer && n.author == "" {
			n.author = parts[1]
		}
		if n.repoInfer && n.repo == "" {
			switch parts[0] {
			case "github.com", "bitbucket.org":
				n.repo = "https://" + strings.Join(parts[0:3], "/")
			}
		}
	}
}
