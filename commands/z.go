package commands

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
)

type Zoxide struct {
	Args []string `opt:"..." required:"false" metavar:"<query>..." complete:"CompleteFolder"`
}

func ZoxideAdd(arg string) error {
	zargs := []string{"add", arg}
	cmd := exec.Command("zoxide", zargs...)
	err := cmd.Run()
	return err
}

func ZoxideQuery(args []string) (string, error) {
	zargs := append([]string{"query"}, args...)
	cmd := exec.Command("zoxide", zargs...)
	res, err := cmd.Output()
	return strings.TrimSuffix(string(res), "\n"), err
}

func init() {
	_, err := exec.LookPath("zoxide")
	if err == nil {
		register(Zoxide{})
	}
}

func (Zoxide) Aliases() []string {
	return []string{"z"}
}

func (*Zoxide) CompleteFolder(arg string) []string {
	return CompletePath(arg, true)
}

// Execute calls zoxide add and query and delegates actually changing the
// directory to ChangeDirectory
func (z Zoxide) Execute(args []string) error {
	if len(z.Args) == 0 {
		z.Args = []string{"~"}
	}
	if len(z.Args) == 1 && (z.Args[0] == "~" || z.Args[0] == "-") {
		if previousDir != "" {
			err := ZoxideAdd(previousDir)
			if err != nil {
				return err
			}
		}
		return ChangeDirectory{Target: z.Args[0]}.Execute(args)
	} else {
		target := xdg.ExpandHome(z.Args[0])
		_, err := os.Stat(target)
		if err != nil || len(z.Args) > 1 {
			// not a file, assume zoxide query
			res, err := ZoxideQuery(z.Args)
			if err != nil {
				return errors.New("zoxide: no match found")
			} else {
				err := ZoxideAdd(res)
				if err != nil {
					return err
				}
				cd := ChangeDirectory{Target: res}
				return cd.Execute([]string{"z", res})
			}

		} else {
			err := ZoxideAdd(target)
			if err != nil {
				return err
			}
			return ChangeDirectory{Target: target}.Execute(args)
		}

	}
}
