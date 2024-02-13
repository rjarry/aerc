package patch

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/pama"
	"git.sr.ht/~rjarry/aerc/lib/pama/models"
)

type Apply struct {
	Cmd      string `opt:"-c"`
	Worktree string `opt:"-w"`
	Tag      string `opt:"tag" required:"true" complete:"CompleteTag"`
}

func init() {
	register(Apply{})
}

func (Apply) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (Apply) Aliases() []string {
	return []string{"apply"}
}

func (*Apply) CompleteTag(arg string) []string {
	patches, err := pama.New().CurrentPatches()
	if err != nil {
		log.Errorf("failed to current patches for completion: %v", err)
		patches = nil
	}

	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}

	uids, err := acct.MarkedMessages()
	if err != nil {
		return nil
	}
	if len(uids) == 0 {
		msg, err := acct.SelectedMessage()
		if err == nil {
			uids = append(uids, msg.Uid)
		}
	}

	store := acct.Store()
	if store == nil {
		return nil
	}

	var subjects []string
	for _, uid := range uids {
		if msg, ok := store.Messages[uid]; !ok || msg == nil || msg.Envelope == nil {
			continue
		} else {
			subjects = append(subjects, msg.Envelope.Subject)
		}
	}
	return proposePatchName(patches, subjects)
}

func (a Apply) Execute(args []string) error {
	patch := a.Tag
	worktree := a.Worktree
	applyCmd := a.Cmd

	m := pama.New()
	p, err := m.CurrentProject()
	if err != nil {
		return err
	}
	log.Tracef("Current project: %v", p)

	if worktree != "" {
		p, err = m.CreateWorktree(p, worktree, patch)
		if err != nil {
			return err
		}
		err = m.SwitchProject(p.Name)
		if err != nil {
			log.Warnf("could not switch to worktree project: %v", err)
		}
	}

	if models.Commits(p.Commits).HasTag(patch) {
		return fmt.Errorf("Patch name '%s' already exists.", patch)
	}

	if !m.Clean(p) {
		return fmt.Errorf("Aborting... There are unstaged changes in " +
			"your repository.")
	}

	commit, err := m.Head(p)
	if err != nil {
		return err
	}
	log.Tracef("HEAD commit before: %s", commit)

	if applyCmd != "" {
		rootFmt := "%r"
		if strings.Contains(applyCmd, rootFmt) {
			applyCmd = strings.ReplaceAll(applyCmd, rootFmt, p.Root)
		}
		log.Infof("use custom apply command: %s", applyCmd)
	} else {
		applyCmd, err = m.ApplyCmd(p)
		if err != nil {
			return err
		}
	}

	msgData := collectMessageData()

	// apply patches with the pipe cmd
	pipe := msg.Pipe{
		Background: false,
		Full:       true,
		Part:       false,
		Command:    applyCmd,
	}
	return pipe.Run(func() {
		p, err = m.ApplyUpdate(p, patch, commit, msgData)
		if err != nil {
			log.Errorf("Failed to save patch data: %v", err)
		}
	})
}

// collectMessageData returns a map where the key is the message id and the
// value the subject of the marked messages
func collectMessageData() map[string]string {
	acct := app.SelectedAccount()
	if acct == nil {
		return nil
	}

	uids, err := commands.MarkedOrSelected(acct)
	if err != nil {
		log.Errorf("error occurred: %v", err)
		return nil
	}

	store := acct.Store()
	if store == nil {
		return nil
	}

	kv := make(map[string]string)
	for _, uid := range uids {
		msginfo, ok := store.Messages[uid]
		if !ok || msginfo == nil {
			continue
		}
		id, err := msginfo.MsgId()
		if err != nil {
			continue
		}
		if msginfo.Envelope == nil {
			continue
		}

		kv[id] = msginfo.Envelope.Subject
	}

	return kv
}

func proposePatchName(patches, subjects []string) []string {
	parse := func(s string) (string, string, bool) {
		var tag strings.Builder
		var version string
		var i, j int

		i = strings.Index(s, "[")
		if i < 0 {
			goto noPatch
		}
		s = s[i+1:]

		j = strings.Index(s, "]")
		if j < 0 {
			goto noPatch
		}
		for _, elem := range strings.Fields(s[:j]) {
			vers := strings.ToLower(elem)
			if !strings.HasPrefix(vers, "v") {
				continue
			}
			isVersion := true
			for _, r := range vers[1:] {
				if !unicode.IsDigit(r) {
					isVersion = false
					break
				}
			}
			if isVersion {
				version = vers
				break
			}
		}
		s = strings.TrimSpace(s[j+1:])

		for _, r := range s {
			if unicode.IsSpace(r) || r == ':' {
				break
			}
			_, err := tag.WriteRune(r)
			if err != nil {
				continue
			}
		}
		return tag.String(), version, true
	noPatch:
		return "", "", false
	}

	summary := make(map[string]struct{})

	var results []string
	for _, s := range subjects {
		tag, version, isPatch := parse(s)
		if tag == "" || !isPatch {
			continue
		}
		if version == "" {
			version = "v1"
		}
		result := fmt.Sprintf("%s_%s", tag, version)
		result = strings.ReplaceAll(result, " ", "")

		collision := false
		for _, name := range patches {
			if name == result {
				collision = true
			}
		}
		if collision {
			continue
		}

		_, ok := summary[result]
		if ok {
			continue
		}
		results = append(results, result)
		summary[result] = struct{}{}
	}

	sort.Strings(results)
	return results
}
