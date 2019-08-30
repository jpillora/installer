package opts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/posener/complete"
	"github.com/posener/complete/cmd/install"
)

//NOTE: Currently completion internally uses posener/complete.
//in future this may change to another implementation.

//Completer represents a shell-completion implementation
//for a single field. By default, all fields will auto-complete
//files and directories. Use a field type which implements this
//Completer interface to override this behaviour.
type Completer interface {
	//Complete is given user's input and should
	//return a corresponding set of valid inputs.
	//Note: all result strings must be prefixed
	//with the user's input.
	Complete(user string) []string
}

//Complete enables shell-completion for this command and
//its subcommands
func (n *node) Complete() Opts {
	n.complete = true
	return n
}

func (n *node) manageCompletion(uninstall bool) error {
	msg := ""
	fn := install.Install
	if uninstall {
		fn = install.Uninstall
	}
	if err := fn(n.name); err != nil {
		w := err.(interface{ WrappedErrors() []error })
		for _, w := range w.WrappedErrors() {
			msg += strings.TrimPrefix(fmt.Sprintf("%s\n", w), "does ")
		}
	} else if uninstall {
		msg = "Uninstalled"
	} else {
		msg = "Installed"
	}
	return exitError(msg) //always exit
}

func (n *node) doCompletion() bool {
	return complete.New(n.name, n.nodeCompletion()).Complete()
}

func (n *node) nodeCompletion() complete.Command {
	//make a completion command for this node
	c := complete.Command{
		Sub:         complete.Commands{},
		Flags:       complete.Flags{},
		GlobalFlags: nil,
		Args:        nil,
	}
	//prepare flags
	for _, item := range n.flags() {
		//item's predictor
		var p complete.Predictor
		//choose a predictor
		if item.noarg {
			//disable
			p = complete.PredictNothing
		} else if item.completer != nil {
			//user completer
			p = &completerWrapper{
				compl: item.completer,
			}
		} else {
			//by default, predicts files and directories
			p = &completerWrapper{
				compl: &completerFS{},
			}
		}
		//add to completion flags set
		c.Flags["--"+item.name] = p
		if item.shortName != "" {
			c.Flags["-"+item.shortName] = p
		}
	}
	//prepare args
	if len(n.args) > 0 {
		c.Args = &completerWrapper{
			compl: &completerFS{},
		}
	}
	//prepare sub-commands
	for name, subn := range n.cmds {
		c.Sub[name] = subn.nodeCompletion() //recurse
	}
	return c
}

type completerWrapper struct {
	compl Completer
}

func (w *completerWrapper) Predict(args complete.Args) []string {
	user := args.Last
	results := w.compl.Complete(user)
	if os.Getenv("OPTS_DEBUG") == "1" {
		debugf("'%s' => %v", user, results)
	}
	return results
}

type completerFS struct{}

func (*completerFS) Complete(user string) []string {
	home := os.Getenv("HOME")
	if home != "" && strings.HasPrefix(user, "~/") {
		user = home + "/" + strings.TrimPrefix(user, "~/")
	}
	completed := []string{}
	matches, _ := filepath.Glob(user + "*")
	for _, m := range matches {
		if home != "" && strings.HasPrefix(m, home) {
			m = "~" + strings.TrimPrefix(m, home)
		}
		if !strings.HasPrefix(m, user) {
			continue
		}
		completed = append(completed, m)
	}
	return matches
}

func debugf(f string, a ...interface{}) {
	l, err := os.OpenFile("/tmp/opts.debug", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		fmt.Fprintf(l, f+"\n", a...)
		l.Close()
	}
}
