package config

import (
	"os"
)

func EditorCmds() []string {
	return []string{
		Compose.Editor,
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
		"vi",
		"nano",
	}
}

func PagerCmds() []string {
	return []string{
		Viewer.Pager,
		os.Getenv("PAGER"),
		"less -Rc",
	}
}
