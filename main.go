package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/mattn/go-isatty"
	"github.com/xo/terminfo"

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
	libui "git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

func getCommands(selected libui.Drawable) []*commands.Commands {
	switch selected.(type) {
	case *widgets.AccountView:
		return []*commands.Commands{
			account.AccountCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Composer:
		return []*commands.Commands{
			compose.ComposeCommands,
			commands.GlobalCommands,
		}
	case *widgets.MessageViewer:
		return []*commands.Commands{
			msgview.MessageViewCommands,
			msg.MessageCommands,
			commands.GlobalCommands,
		}
	case *widgets.Terminal:
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
//	:q  --> :quit
//	:ar --> :archive
//	:im --> :import-mbox
func expandAbbreviations(cmd []string, sets []*commands.Commands) []string {
	if len(cmd) == 0 {
		return cmd
	}
	name := strings.TrimLeft(cmd[0], ":")
	candidate := ""
	for _, set := range sets {
		if set.ByName(name) != nil {
			// Direct match, return it directly.
			return cmd
		}
		// Check for partial matches.
		for _, n := range set.Names() {
			if !strings.HasPrefix(n, name) {
				continue
			}
			if candidate != "" {
				// We have more than one command partially
				// matching the input. We can't expand such an
				// abbreviation, so return the command as is so
				// it can raise an error later.
				return cmd
			}
			// We have a partial match.
			candidate = n
		}
	}
	// As we are here, we could have a command name matching our partial
	// name in `cmd`. In that case we replace the name in `cmd` with the
	// full name, otherwise we simply return `cmd` as is.
	if candidate != "" {
		cmd[0] = candidate
	}
	return cmd
}

func execCommand(
	aerc *widgets.Aerc, ui *libui.UI, cmd []string,
	acct *config.AccountConfig, msg *models.MessageInfo,
) error {
	cmds := getCommands(aerc.SelectedTabContent())
	cmd = expandAbbreviations(cmd, cmds)
	for i, set := range cmds {
		err := set.ExecuteCommand(aerc, cmd, acct, msg)
		if err != nil {
			if errors.As(err, new(commands.NoSuchCommand)) {
				if i == len(cmds)-1 {
					return err
				}
				continue
			}
			if errors.As(err, new(commands.ErrorExit)) {
				ui.Exit()
				return nil
			}
			return err
		}
		break
	}
	return nil
}

func getCompletions(aerc *widgets.Aerc, cmd string) ([]string, string) {
	if options, prefix, ok := commands.GetTemplateCompletion(aerc, cmd); ok {
		return options, prefix
	}
	var completions []string
	var prefix string
	for _, set := range getCommands(aerc.SelectedTabContent()) {
		options, s := set.GetCompletions(aerc, cmd)
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

func usage(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, "usage: aerc [-v] [-a <account-name[,account-name>] [mailto:...]")
	os.Exit(1)
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

func main() {
	defer log.PanicHandler()
	opts, optind, err := getopt.Getopts(os.Args, "va:")
	if err != nil {
		usage("error: " + err.Error())
		return
	}
	log.BuildInfo = buildInfo()
	var accts []string
	for _, opt := range opts {
		if opt.Option == 'v' {
			fmt.Println("aerc " + log.BuildInfo)
			return
		}
		if opt.Option == 'a' {
			accts = strings.Split(opt.Value, ",")
		}
	}
	retryExec := false
	args := os.Args[optind:]
	if len(args) > 0 {
		err := ipc.ConnectAndExec(args)
		if err == nil {
			return // other aerc instance takes over
		}
		fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
		// continue with setting up a new aerc instance and retry after init
		retryExec = true
	}

	err = config.LoadConfigFromFile(nil, accts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1) //nolint:gocritic // PanicHandler does not need to run as it's not a panic
	}

	log.Infof("Starting up version %s", log.BuildInfo)

	var (
		aerc *widgets.Aerc
		ui   *libui.UI
	)

	deferLoop := make(chan struct{})

	c := crypto.New()
	err = c.Init()
	if err != nil {
		log.Warnf("failed to initialise crypto interface: %v", err)
	}
	defer c.Close()

	aerc = widgets.NewAerc(c, func(
		cmd []string, acct *config.AccountConfig,
		msg *models.MessageInfo,
	) error {
		return execCommand(aerc, ui, cmd, acct, msg)
	}, func(cmd string) ([]string, string) {
		return getCompletions(aerc, cmd)
	}, &commands.CmdHistory, deferLoop)

	ui, err = libui.Initialize(aerc)
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

	as, err := ipc.StartServer(aerc)
	if err != nil {
		log.Warnf("Failed to start Unix server: %v", err)
	} else {
		defer as.Close()
	}

	// set the aerc version so that we can use it in the template funcs
	templates.SetVersion(Version)

	if retryExec {
		// retry execution
		err := ipc.ConnectAndExec(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to communicate to aerc: %v\n", err)
			err = aerc.CloseBackends()
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
			aerc.PushError(msg)
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
			aerc.HandleMessage(msg)
		case callback := <-libui.Callbacks:
			callback()
		case <-libui.Redraw:
			ui.Render()
		case <-ui.Quit:
			err = aerc.CloseBackends()
			if err != nil {
				log.Warnf("failed to close backends: %v", err)
			}
			break loop
		}
	}
}
