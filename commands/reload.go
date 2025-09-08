package commands

import (
	"os"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/ui"
)

type Reload struct {
	Binds bool   `opt:"-B" desc:"Reload binds.conf."`
	Conf  bool   `opt:"-C" desc:"Reload aerc.conf."`
	Style string `opt:"-s" complete:"CompleteStyle" desc:"Reload the specified styleset."`
}

func init() {
	Register(Reload{})
}

func (Reload) Description() string {
	return "Hot-reload configuration files."
}

func (r *Reload) CompleteStyle(s string) []string {
	var files []string
	for _, dir := range config.Ui().StyleSetDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			log.Debugf("could not read directory '%s': %v", dir,
				err)
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			files = append(files, e.Name())
		}
	}
	return FilterList(files, s, nil)
}

func (Reload) Context() CommandContext {
	return GLOBAL
}

func (Reload) Aliases() []string {
	return []string{"reload"}
}

func (r Reload) Execute(args []string) error {
	if !r.Binds && !r.Conf && r.Style == "" {
		r.Binds = true
		r.Conf = true
		r.Style = config.Ui().StyleSetName
	}

	reconfigure := false

	if r.Binds {
		f, err := config.ReloadBinds()
		if err != nil {
			return err
		}
		app.PushSuccess("Binds reloaded: " + f)
	}

	if r.Conf {
		f, err := config.ReloadConf()
		if err != nil {
			return err
		}
		app.PushSuccess("Conf reloaded: " + f)
		reconfigure = true
	}

	if r.Style != "" {
		config.Ui().ClearCache()
		config.Ui().StyleSetName = r.Style
		err := config.Ui().LoadStyle()
		if err != nil {
			return err
		}
		app.PushSuccess("Styleset: " + r.Style)
		reconfigure = true
	}

	if !reconfigure {
		return nil
	}

	// reload account views and message stores
	for _, name := range app.AccountNames() {

		// rebuild account view
		view, err := app.Account(name)
		if err != nil {
			continue
		}

		dirlist := view.Directories()
		if dirlist == nil {
			continue
		}

		wantTree := config.Ui().ForAccount(name).DirListTree
		dirlist = adjustDirlist(dirlist, wantTree)
		view.SetDirectories(dirlist)

		// now rebuild grid with correct dirlist
		view.Configure()

		// reconfigure the message stores
		for _, dir := range dirlist.List() {
			store, ok := dirlist.MsgStore(dir)
			if !ok {
				continue
			}
			uiConf := dirlist.UiConfig(dir)
			store.Configure(view.SortCriteria(uiConf))
		}
		ui.Invalidate()
	}

	// reload message viewers
	doTabs(func(tab *ui.Tab) {
		mv, ok := tab.Content.(*app.MessageViewer)
		if !ok {
			return
		}
		reloaded, err := app.NewMessageViewer(
			mv.SelectedAccount(),
			mv.MessageView(),
		)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		app.ReplaceTab(mv, reloaded, tab.Name, false)
	})

	// reload composers
	doTabs(func(tab *ui.Tab) {
		c, ok := tab.Content.(*app.Composer)
		if !ok {
			return
		}
		_ = c.SwitchAccount(c.Account())
	})

	return nil
}

func adjustDirlist(d app.DirectoryLister, wantTree bool) app.DirectoryLister {
	switch d := d.(type) {
	case *app.DirectoryList:
		if wantTree {
			log.Tracef("dirlist: build tree")
			tree := app.NewDirectoryTree(d)
			tree.SelectedMsgStore()
			return tree
		}
		log.Tracef("dirlist: already dirlist")
		return d
	case *app.DirectoryTree:
		if !wantTree {
			log.Tracef("dirtree: get dirlist")
			return d.DirectoryList
		}
		log.Tracef("dirtree: already tree")
		return d
	default:
		return d
	}
}

func doTabs(do func(*ui.Tab)) {
	var tabname string
	if t := app.SelectedTab(); t != nil {
		tabname = t.Name
	}
	for i := range app.TabNames() {
		tab := app.GetTab(i)
		if tab == nil {
			continue
		}
		do(tab)
	}
	if tabname != "" {
		app.SelectTab(tabname)
	}
}
