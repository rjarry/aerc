package statusline

type folderState struct {
	Search         string
	Filter         string
	FilterActivity string

	Threading string
}

func (fs *folderState) State() []string {
	var line []string

	if fs.FilterActivity != "" {
		line = append(line, fs.FilterActivity)
	} else {
		if fs.Filter != "" {
			line = append(line, fs.Filter)
		}
	}
	if fs.Search != "" {
		line = append(line, fs.Search)
	}
	if fs.Threading != "" {
		line = append(line, fs.Threading)
	}
	return line
}
