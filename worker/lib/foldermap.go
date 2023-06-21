package lib

import (
	"fmt"
	"io"

	"github.com/go-ini/ini"
)

func ParseFolderMap(r io.Reader) (map[string]string, []string, error) {
	cfg, err := ini.Load(r)
	if err != nil {
		return nil, nil, err
	}

	sec, err := cfg.GetSection("")
	if err != nil {
		return nil, nil, err
	}

	order := sec.KeyStrings()

	for _, k := range order {
		v, err := sec.GetKey(k)
		switch {
		case v.String() == "":
			return nil, nil, fmt.Errorf("no value for key '%s'", k)
		case err != nil:
			return nil, nil, err
		}
	}

	return sec.KeysHash(), order, nil
}
