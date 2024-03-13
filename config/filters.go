package config

import (
	"regexp"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"github.com/go-ini/ini"
)

type FilterType int

const (
	FILTER_MIMETYPE FilterType = iota
	FILTER_HEADER
	FILTER_HEADERS
	FILTER_FILENAME
)

type FilterConfig struct {
	Type    FilterType
	Filter  string
	Command string
	Header  string
	Regex   *regexp.Regexp
}

var Filters []*FilterConfig

func parseFilters(file *ini.File) error {
	filters, err := file.GetSection("filters")
	if err != nil {
		goto end
	}

	for _, key := range filters.Keys() {
		filter := FilterConfig{
			Command: key.Value(),
			Filter:  key.Name(),
		}

		switch {
		case strings.HasPrefix(filter.Filter, ".filename,~"):
			filter.Type = FILTER_FILENAME
			regex := filter.Filter[strings.Index(filter.Filter, "~")+1:]
			filter.Regex, err = regexp.Compile(regex)
			if err != nil {
				return err
			}
		case strings.HasPrefix(filter.Filter, ".filename,"):
			filter.Type = FILTER_FILENAME
			value := filter.Filter[strings.Index(filter.Filter, ",")+1:]
			filter.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			if err != nil {
				return err
			}
		case strings.Contains(filter.Filter, ",~"):
			filter.Type = FILTER_HEADER
			//nolint:gocritic // guarded by strings.Contains
			header := filter.Filter[:strings.Index(filter.Filter, ",")]
			regex := filter.Filter[strings.Index(filter.Filter, "~")+1:]
			filter.Header = strings.ToLower(header)
			filter.Regex, err = regexp.Compile(regex)
			if err != nil {
				return err
			}
		case strings.ContainsRune(filter.Filter, ','):
			filter.Type = FILTER_HEADER
			//nolint:gocritic // guarded by strings.Contains
			header := filter.Filter[:strings.Index(filter.Filter, ",")]
			value := filter.Filter[strings.Index(filter.Filter, ",")+1:]
			filter.Header = strings.ToLower(header)
			filter.Regex, err = regexp.Compile(regexp.QuoteMeta(value))
			if err != nil {
				return err
			}
		case filter.Filter == ".headers":
			filter.Type = FILTER_HEADERS
		default:
			filter.Type = FILTER_MIMETYPE
		}
		Filters = append(Filters, &filter)
	}

end:
	log.Debugf("aerc.conf: [filters] %#v", Filters)
	return nil
}
