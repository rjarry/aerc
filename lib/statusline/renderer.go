package statusline

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

type renderParams struct {
	width int
	sep   string
	acct  *accountState
	fldr  *folderState
}

type renderFunc func(r renderParams) string

func newRenderer(renderFormat, textMode string) renderFunc {
	var texter Texter
	switch strings.ToLower(textMode) {
	case "icon":
		texter = &icon{}
	default:
		texter = &text{}
	}

	return renderer(texter, renderFormat)
}

func renderer(texter Texter, renderFormat string) renderFunc {
	var leftFmt, rightFmt string
	if idx := strings.Index(renderFormat, "%>"); idx < 0 {
		leftFmt = renderFormat
	} else {
		leftFmt, rightFmt = renderFormat[:idx], strings.Replace(renderFormat[idx:], "%>", "", 1)
	}

	return func(r renderParams) string {
		lfmtStr, largs, err := parseStatuslineFormat(leftFmt, texter, r)
		if err != nil {
			return err.Error()
		}
		rfmtStr, rargs, err := parseStatuslineFormat(rightFmt, texter, r)
		if err != nil {
			return err.Error()
		}
		leftText, rightText := fmt.Sprintf(lfmtStr, largs...), fmt.Sprintf(rfmtStr, rargs...)
		return runewidth.FillRight(leftText, r.width-len(rightText)-1) + rightText
	}
}

func connectionInfo(acct *accountState, texter Texter) (conn string) {
	if acct.ConnActivity != "" {
		conn += acct.ConnActivity
	} else {
		if acct.Connected {
			conn += texter.Connected()
		} else {
			conn += texter.Disconnected()
		}
	}
	return
}

func contentInfo(acct *accountState, fldr *folderState, texter Texter) []string {
	var status []string
	if fldr.FilterActivity != "" {
		status = append(status, fldr.FilterActivity)
	} else if fldr.Filter != "" {
		status = append(status, texter.FormatFilter(fldr.Filter))
	}
	if fldr.Search != "" {
		status = append(status, texter.FormatSearch(fldr.Search))
	}
	return status
}

func trayInfo(acct *accountState, fldr *folderState, texter Texter) []string {
	var tray []string
	if fldr.Sorting {
		tray = append(tray, texter.Sorting())
	}
	if fldr.Threading {
		tray = append(tray, texter.Threading())
	}
	if acct.Passthrough {
		tray = append(tray, texter.Passthrough())
	}
	return tray
}

func parseStatuslineFormat(format string, texter Texter, r renderParams) (string, []interface{}, error) {
	retval := make([]byte, 0, len(format))
	var args []interface{}
	mute := false

	var c rune
	for i, ni := 0, 0; i < len(format); {
		ni = strings.IndexByte(format[i:], '%')
		if ni < 0 {
			ni = len(format)
			retval = append(retval, []byte(format[i:ni])...)
			break
		}
		ni += i + 1
		// Check for fmt flags
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		if c == '+' || c == '-' || c == '#' || c == ' ' || c == '0' {
			ni++
		}

		// Check for precision and width
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		for unicode.IsDigit(c) {
			ni++
			c = rune(format[ni])
		}
		if c == '.' {
			ni++
			c = rune(format[ni])
			for unicode.IsDigit(c) {
				ni++
				c = rune(format[ni])
			}
		}

		retval = append(retval, []byte(format[i:ni])...)
		// Get final format verb
		if ni == len(format) {
			goto handle_end_error
		}
		c = rune(format[ni])
		switch c {
		case '%':
			retval = append(retval, '%')
		case 'a':
			retval = append(retval, 's')
			args = append(args, r.acct.Name)
		case 'c':
			retval = append(retval, 's')
			args = append(args, connectionInfo(r.acct, texter))
		case 'd':
			retval = append(retval, 's')
			args = append(args, r.fldr.Name)
		case 'm':
			mute = true
		case 'S':
			var status []string
			if conn := connectionInfo(r.acct, texter); conn != "" {
				status = append(status, conn)
			}

			if r.acct.Connected {
				status = append(status, contentInfo(r.acct, r.fldr, texter)...)
			}
			retval = append(retval, 's')
			args = append(args, strings.Join(status, r.sep))
		case 'T':
			var tray []string
			if r.acct.Connected {
				tray = trayInfo(r.acct, r.fldr, texter)
			}
			retval = append(retval, 's')
			args = append(args, strings.Join(tray, r.sep))
		case 'p':
			path, err := os.Getwd()
			if err == nil {
				home, _ := os.UserHomeDir()
				if strings.HasPrefix(path, home) {
					path = strings.Replace(path, home, "~", 1)
				}
				retval = append(retval, 's')
				args = append(args, path)
			}
		default:
			// Just ignore it and print as is
			// so %k in index format becomes %%k to Printf
			retval = append(retval, '%')
			retval = append(retval, byte(c))
		}
		i = ni + 1
	}

	if mute {
		return "", nil, nil
	}

	return string(retval), args, nil

handle_end_error:
	return "", nil,
		errors.New("reached end of string while parsing statusline format")
}
