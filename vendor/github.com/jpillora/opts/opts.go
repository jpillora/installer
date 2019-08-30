package opts

import (
	"flag"
	"reflect"
)

//Opts is a single configuration command instance. It represents a node
//in a tree of commands. Use the AddCommand method to add subcommands (child nodes)
//to this command instance.
type Opts interface {
	//Name of the command. For the root command, Name defaults to the executable's
	//base name. For subcommands, Name defaults to the package name, unless its the
	//main package, then it defaults to the struct name.
	Name(name string) Opts
	//Version of the command. Commonly set using a package main variable at compile
	//time using ldflags (for example, go build -ldflags -X main.version=42).
	Version(version string) Opts
	//ConfigPath is a path to a JSON file to use as defaults. This is useful in
	//global paths like /etc/my-prog.json. For a user-specified path. Use the
	//UserConfigPath method.
	ConfigPath(path string) Opts
	//UserConfigPath is the same as ConfigPath however an extra flag (--config-path)
	//is added to this Opts instance to give the user control of the filepath.
	//Configuration unmarshalling occurs after flag parsing.
	UserConfigPath() Opts
	//UseEnv enables the default environment variables on all fields. This is
	//equivalent to adding the opts tag "env" on all flag fields.
	UseEnv() Opts
	//Complete enables auto-completion for this command. When enabled, two extra
	//flags are added (--install and --uninstall) which can be used to install
	//a dynamic shell (bash, zsh, fish) completion for this command. Internally,
	//this adds a stub file which runs the Go binary to auto-complete its own
	//command-line interface. Note, the absolute path returned from os.Executable()
	//is used to reference to the Go binary.
	Complete() Opts
	//EmbedFlagSet embeds the given pkg/flag.FlagSet into
	//this Opts instance. Placing the flags defined in the FlagSet
	//along side the configuration struct flags.
	EmbedFlagSet(*flag.FlagSet) Opts
	//EmbedGlobalFlagSet embeds the global pkg/flag.CommandLine
	//FlagSet variable into this Opts instance.
	EmbedGlobalFlagSet() Opts

	//Summary adds a short sentence below the usage text
	Summary(summary string) Opts
	//Repo sets the source repository of the program and is displayed
	//at the bottom of the help text.
	Repo(repo string) Opts
	//Author sets the author of the program and is displayed
	//at the bottom of the help text.
	Author(author string) Opts
	//PkgRepo automatically sets Repo using the struct's package path.
	//This does not work for types defined in the main package.
	PkgRepo() Opts
	//PkgAuthor automatically sets Author using the struct's package path.
	//This does not work for types defined in the main package.
	PkgAuthor() Opts
	//DocSet replaces an existing template.
	DocSet(id, template string) Opts
	//DocBefore inserts a new template before an existing template.
	DocBefore(existingID, newID, template string) Opts
	//DocAfter inserts a new template after an existing template.
	DocAfter(existingID, newID, template string) Opts
	//DisablePadAll removes the padding from the help text.
	DisablePadAll() Opts
	//SetPadWidth alters the padding to specific number of spaces.
	//By default, pad width is 2.
	SetPadWidth(padding int) Opts
	//SetLineWidth alters the maximum number of characters in a
	//line (excluding padding). By default, line width is 96.
	SetLineWidth(width int) Opts

	//AddCommand adds another Opts instance as a subcommand.
	AddCommand(Opts) Opts
	//Parse uses os.Args to parse the current flags and args.
	Parse() ParsedOpts
	//ParseArgs uses a given set of args to to parse the
	//current flags and args. Assumes the executed program is
	//the first arg.
	ParseArgs(args []string) ParsedOpts
}

type ParsedOpts interface {
	//Help returns the final help text
	Help() string
	//IsRunnable returns whether the matched command has a Run method
	IsRunnable() bool
	//Run assumes the matched command is runnable and executes its Run method.
	//The target Run method must be 'Run() error' or 'Run()'
	Run() error
	//RunFatal assumes the matched command is runnable and executes its Run method.
	//However, any error will be printed, followed by an exit(1).
	RunFatal()
}

//New creates a new Opts instance using the given configuration
//struct pointer.
func New(config interface{}) Opts {
	return newNode(reflect.ValueOf(config))
}

//Parse is shorthand for
//  opts.New(config).Parse()
func Parse(config interface{}) ParsedOpts {
	return New(config).Parse()
}

//Setter is any type which can be set from a string.
//This includes flag.Value.
type Setter interface {
	Set(string) error
}
