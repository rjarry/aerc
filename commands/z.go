package commands

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

type Zoxide struct {
	Target string `opt:"..." default:"~" metavar:"<folder> | <query>..."`
}

func ZoxideAdd(arg string) error {
	zargs := []string{"add", arg}
	cmd := exec.Command("zoxide", zargs...)
	err := cmd.Run()
	return err
}

func ZoxideQuery(args []string) (string, error) {
	zargs := append([]string{"query"}, args[1:]...)
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

func (Zoxide) Complete(args []string) []string {
	return ChangeDirectory{}.Complete(args)
}

// Execute calls zoxide add and query and delegates actually changing the
// directory to ChangeDirectory
func (z Zoxide) Execute(args []string) error {
	switch z.Target {
	case "-", "~":
		if previousDir != "" {
			err := ZoxideAdd(previousDir)
			if err != nil {
				return err
			}
		}
		return ChangeDirectory{}.Execute(args)
	default:
		_, err := os.Stat(z.Target)
		if err != nil {
			// not a file, assume zoxide query
			res, err := ZoxideQuery(args)
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
			err := ZoxideAdd(z.Target)
			if err != nil {
				return err
			}
			return ChangeDirectory{}.Execute(args)
		}

	}
}
