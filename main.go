package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~rjarry/go-opt"
	"github.com/mattn/go-isatty"
	"github.com/xo/terminfo"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/commands/compose"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/commands/msgview"
	"git.sr.ht/~rjarry/aerc/commands/terminal"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/hooks"
	"git.sr.ht/~rjarry/aerc/lib/ipc"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func getCommands(selected ui.Drawable) []*commands.Commands {
	switch selected.(type) {
	case *app.AccountView:
		return []*commands.Commands{
			account.AccountCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *app.Composer:
		return []*commands.Commands{
			compose.ComposeCommands,
			commands.GlobalCommands,
		}
	case *app.MessageViewer:
		return []*commands.Commands{
			msgview.MessageViewCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *app.Terminal:
		return []*commands.Commands{
			terminal.TerminalCommands,
			commands.GlobalCommands,
		}
	default:
		return []*commands.Commands{commands.GlobalCommands}
	}
}

// Expand non-ambiguous command abbreviations.
//
//	q  --> quit
//	ar --> archive
//	im --> import-mbox
func expandAbbreviations(name string, sets []*commands.Commands) (string, commands.Command) {
	var cmd commands.Command
	candidate := name

	for _, set := range sets {
		cmd = set.ByName(name)
		if cmd != nil {
			// Direct match, return it directly.
			return name, cmd
		}
		// Check for partial matches.
		for _, n := range set.Names() {
			if !strings.HasPrefix(n, name) {
				continue
			}
			if cmd != nil {
				// We have more than one command partially
				// matching the input. We can't expand such an
				// abbreviation, so return the command as is so
				// it can raise an error later.
				return name, nil
			}
			// We have a partial match.
			candidate = n
			cmd = set.ByName(n)
		}
	}
	return candidate, cmd
}

func execCommand(
	cmdline string,
	acct *config.AccountConfig, msg *models.MessageInfo,
) error {
	cmdline, err := commands.ExpandTemplates(cmdline, acct, msg)
	if err != nil {
		return err
	}
	cmdline = strings.TrimLeft(cmdline, ":")
	name, rest, didCut := strings.Cut(cmdline, " ")
	cmds := getCommands(app.SelectedTabContent())
	name, cmd := expandAbbreviations(name, cmds)
	if cmd == nil {
		return commands.NoSuchCommand(name)
	}
	cmdline = name
	if didCut {
		cmdline += " " + rest
	}
	err = commands.ExecuteCommand(cmd, cmdline)
	if errors.As(err, new(commands.ErrorExit)) {
		ui.Exit()
		return nil
	}
	return err
}

func getCompletions(cmd string) ([]string, string) {
	if options, prefix, ok := commands.GetTemplateCompletion(cmd); ok {
		return options, prefix
	}
	var completions []string
	var prefix string
	for _, set := range getCommands(app.SelectedTabContent()) {
		options, s := set.GetCompletions(cmd)
		if s != "" {
			prefix = s
		}
		completions = append(completions, options...)
	}
	sort.Strings(completions)
	return completions, prefix
}

// set at build time
var (
	Version string
	Flags   string
)

func buildInfo() string {
	info := Version
	flags, _ := base64.StdEncoding.DecodeString(Flags)
	if strings.Contains(string(flags), "notmuch") {
		info += " +notmuch"
	}
	info += fmt.Sprintf(" (%s %s %s)",
		runtime.Version(), runtime.GOARCH, runtime.GOOS)
	return info
}

func setWindowTitle() {
	log.Tracef("Parsing terminfo")
	ti, err := terminfo.LoadFromEnv()
	if err != nil {
		log.Warnf("Cannot get terminfo: %v", err)
		return
	}

	if !ti.Has(terminfo.HasStatusLine) {
		log.Infof("Terminal does not have status line support")
		return
	}

	log.Debugf("Setting terminal title")
	buf := new(bytes.Buffer)
	ti.Fprintf(buf, terminfo.ToStatusLine)
	fmt.Fprint(buf, "aerc")
	ti.Fprintf(buf, terminfo.FromStatusLine)
	os.Stderr.Write(buf.Bytes())
}

type Opts struct {
	Help     bool     `opt:"-h" action:"ShowHelp"`
	Version  bool     `opt:"-v" action:"ShowVersion"`
	Accounts []string `opt:"-a" action:"ParseAccounts" metavar:"<account>"`
	Command  []string `opt:"..." required:"false" metavar:"mailto:<address> | mbox:<file> | :<command...>"`
}

func (o *Opts) ShowHelp(arg string) error {
	fmt.Println("Usage: " + opt.NewCmdSpec(os.Args[0], o).Usage())
	fmt.Print(`
Aerc is an email client for your terminal.

Options:

  -h                 Show this help message and exit.
  -v                 Print version information.
  -a <account>       Load only the named account, as opposed to all configured
                     accounts. It can also be a comma separated list of names.
                     This option may be specified multiple times. The account
                     order will be preserved.
  mailto:<address>   Open the composer with the address(es) in the To field.
                     If aerc is already running, the composer is started in
                     this instance, otherwise aerc will be started.
  mbox:<file>        Open the specified mbox file as a virtual temporary account.
  :<command...>      Run an aerc command as you would in Ex-Mode.
`)
	os.Exit(0)
	return nil
}

func (o *Opts) ShowVersion(arg string) error {
	fmt.Println(log.BuildInfo)
	os.Exit(0)
	return nil
}

func (o *Opts) ParseAccounts(arg string) error {
	o.Accounts = append(o.Accounts, strings.Split(arg, ",")...)
	return nil
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	defer log.PanicHandler()
	log.BuildInfo = buildInfo()

	var opts Opts

	args := opt.QuoteArgs(os.Args...)
	err := opt.ArgsToStruct(args, &opts)
	if err != nil {
		die("%s", err)
	}
	retryExec := false
	if len(opts.Command) > 0 {
		err := ipc.ConnectAndExec(opts.Command)
		if err == nil {
			return // other aerc instance takes over
		}
		fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
		// continue with setting up a new aerc instance and retry after init
		retryExec = true
	}

	err = config.LoadConfigFromFile(nil, opts.Accounts)
	if err != nil {
		die("failed to load config: %s", err)
	}

	log.Infof("Starting up version %s", log.BuildInfo)

	deferLoop := make(chan struct{})

	c := crypto.New()
	err = c.Init()
	if err != nil {
		log.Warnf("failed to initialise crypto interface: %v", err)
	}
	defer c.Close()

	app.Init(c, execCommand, getCompletions, &commands.CmdHistory, deferLoop)

	err = ui.Initialize(app.Drawable())
	if err != nil {
		panic(err)
	}
	defer ui.Close()
	log.UICleanup = func() {
		ui.Close()
	}
	close(deferLoop)

	if config.Ui.MouseEnabled {
		ui.EnableMouse()
	}

	as, err := ipc.StartServer(app.IPCHandler())
	if err != nil {
		log.Warnf("Failed to start Unix server: %v", err)
	} else {
		defer as.Close()
	}

	// set the aerc version so that we can use it in the template funcs
	templates.SetVersion(Version)

	if retryExec {
		// retry execution
		err := ipc.ConnectAndExec(opts.Command)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
			err = app.CloseBackends()
			if err != nil {
				log.Warnf("failed to close backends: %v", err)
			}
			return
		}
	}

	if isatty.IsTerminal(os.Stderr.Fd()) {
		setWindowTitle()
	}

	go func() {
		defer log.PanicHandler()
		err := hooks.RunHook(&hooks.AercStartup{Version: Version})
		if err != nil {
			msg := fmt.Sprintf("aerc-startup hook: %s", err)
			app.PushError(msg)
		}
	}()
	defer func(start time.Time) {
		err := hooks.RunHook(
			&hooks.AercShutdown{Lifetime: time.Since(start)},
		)
		if err != nil {
			log.Errorf("aerc-shutdown hook: %s", err)
		}
	}(time.Now())
loop:
	for {
		select {
		case event := <-ui.Events:
			ui.HandleEvent(event)
		case msg := <-types.WorkerMessages:
			app.HandleMessage(msg)
		case callback := <-ui.Callbacks:
			callback()
		case <-ui.Redraw:
			ui.Render()
		case <-ui.SuspendQueue:
			err = ui.Suspend()
			if err != nil {
				app.PushError(fmt.Sprintf("suspend: %s", err))
			}
		case <-ui.Quit:
			err = app.CloseBackends()
			if err != nil {
				log.Warnf("failed to close backends: %v", err)
			}
			break loop
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
