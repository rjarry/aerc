package hooks

type HookType interface {
	Cmd() string
	Env() []string
}
