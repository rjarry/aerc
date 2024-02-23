package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"git.sr.ht/~rjarry/go-opt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/hooks"
	"git.sr.ht/~rjarry/aerc/lib/ipc"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/templates"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"

	_ "git.sr.ht/~rjarry/aerc/commands/account"
	_ "git.sr.ht/~rjarry/aerc/commands/compose"
	_ "git.sr.ht/~rjarry/aerc/commands/msg"
	_ "git.sr.ht/~rjarry/aerc/commands/msgview"
	_ "git.sr.ht/~rjarry/aerc/commands/patch"
)

func execCommand(
	cmdline string,
	acct *config.AccountConfig, msg *models.MessageInfo,
) error {
	cmdline, cmd, err := commands.ResolveCommand(cmdline, acct, msg)
	if err != nil {
		return err
	}
	err = commands.ExecuteCommand(cmd, cmdline)
	if errors.As(err, new(commands.ErrorExit)) {
		ui.Exit()
		return nil
	}
	return err
}

func getCompletions(cmdline string) ([]string, string) {
	// complete template terms
	if options, prefix, ok := commands.GetTemplateCompletion(cmdline); ok {
		sort.Strings(options)
		return options, prefix
	}

	args := opt.LexArgs(cmdline)

	if args.Count() < 2 && args.TrailingSpace() == "" {
		// complete command names
		var completions []string
		for _, name := range commands.ActiveCommandNames() {
			if strings.HasPrefix(name, cmdline) {
				completions = append(completions, name+" ")
			}
		}
		sort.Strings(completions)
		return completions, ""
	}

	// complete command arguments
	_, cmd, err := commands.ExpandAbbreviations(args.Arg(0))
	if err != nil {
		return nil, cmdline
	}
	return commands.GetCompletions(cmd, args)
}

// set at build time
var (
	Version string
	Date    string
)

func buildInfo() string {
	info := Version
	if soVersion, hasNotmuch := lib.NotmuchVersion(); hasNotmuch {
		info += fmt.Sprintf(" +notmuch-%s", soVersion)
	}
	info += fmt.Sprintf(" (%s %s %s %s)",
		runtime.Version(), runtime.GOARCH, runtime.GOOS, Date)
	return info
}

type Opts struct {
	Help         bool     `opt:"-h" action:"ShowHelp"`
	Version      bool     `opt:"-v" action:"ShowVersion"`
	Accounts     []string `opt:"-a" action:"ParseAccounts" metavar:"<account>"`
	ConfAerc     string   `opt:"--aerc-conf"`
	ConfAccounts string   `opt:"--accounts-conf"`
	ConfBinds    string   `opt:"--binds-conf"`
	NoIPC        bool     `opt:"--no-ipc"`
	Command      []string `opt:"..." required:"false" metavar:"mailto:<address> | mbox:<file> | :<command...>"`
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
  --aerc-conf        Path to configuration file to be used instead of the default.
  --accounts-conf    Path to configuration file to be used instead of the default.
  --binds-conf       Path to configuration file to be used instead of the default.
  --no-ipc           Run any commands in this aerc instance, and don't create a
                     socket for other aerc instances to communicate with this one.
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
	fmt.Println("aerc " + log.BuildInfo)
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

	err = config.LoadConfigFromFile(
		nil, opts.Accounts, opts.ConfAerc, opts.ConfBinds, opts.ConfAccounts,
	)
	if err != nil {
		die("%s", err)
	}

	noIPC := opts.NoIPC || config.General.DisableIPC

	if len(opts.Command) > 0 && !noIPC &&
		!(config.General.DisableIPCMailto && strings.HasPrefix(opts.Command[0], "mailto:")) &&
		!(config.General.DisableIPCMbox && strings.HasPrefix(opts.Command[0], "mbox:")) {

		response, err := ipc.ConnectAndExec(opts.Command)
		if err == nil {
			if response.Error != "" {
				fmt.Printf("response: %s\n", response.Error)
			}
			return // other aerc instance takes over
		}
		// continue with setting up a new aerc instance and retry after init
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

	startup, startupDone := context.WithCancel(context.Background())

	if !noIPC {
		as, err := ipc.StartServer(app.IPCHandler(), startup)
		if err != nil {
			log.Warnf("Failed to start Unix server: %v", err)
		} else {
			defer as.Close()
		}
	}

	// set the aerc version so that we can use it in the template funcs
	templates.SetVersion(Version)

	endStartup := func() {
		startupDone()
		if len(opts.Command) == 0 {
			return
		}
		// Retry execution. Since IPC has already failed, we know no
		// other aerc instance is running (or IPC was explicitly
		// disabled); run the command directly.
		err := app.Command(opts.Command)
		if err != nil {
			// no other aerc instance is running, so let
			// this one stay running but show the error
			errMsg := fmt.Sprintf("Startup command (%s) failed: %s\n",
				strings.Join(opts.Command, " "), err)
			log.Errorf(errMsg)
			app.PushError(errMsg)
		}
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
	var once sync.Once
loop:
	for {
		select {
		case event := <-ui.Events:
			ui.HandleEvent(event)
		case msg := <-types.WorkerMessages:
			app.HandleMessage(msg)
			// XXX: The app may not be 100% ready at this point.
			// The issue is that there is no real way to tell when
			// it will be ready. And in some cases, it may never be.
			// At least, we can be confident that accepting IPC
			// commands will not crash the whole process.
			once.Do(endStartup)
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
