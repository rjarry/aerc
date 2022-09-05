package commands

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"git.sr.ht/~rjarry/aerc/widgets"
)

type Zoxide struct{}

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

func (Zoxide) Complete(aerc *widgets.Aerc, args []string) []string {
	return ChangeDirectory{}.Complete(aerc, args)
}

// Execute calls zoxide add and query and delegates actually changing the
// directory to ChangeDirectory
func (Zoxide) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) < 1 {
		return errors.New("Usage: z [directory or zoxide query]")
	}
	target := strings.Join(args[1:], " ")
	switch target {
	case "":
		return ChangeDirectory{}.Execute(aerc, args)
	case "-":
		if previousDir != "" {
			err := ZoxideAdd(previousDir)
			if err != nil {
				return err
			}
		}
		return ChangeDirectory{}.Execute(aerc, args)
	default:
		_, err := os.Stat(target)
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
				return ChangeDirectory{}.Execute(aerc, []string{"z", res})
			}

		} else {
			err := ZoxideAdd(target)
			if err != nil {
				return err
			}
			return ChangeDirectory{}.Execute(aerc, args)
		}

	}
}
