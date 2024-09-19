package app

import (
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/auth"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/mattn/go-runewidth"
)

type AuthInfo struct {
	authdetails *auth.Details
	showInfo    bool
	uiConfig    *config.UIConfig
}

func NewAuthInfo(auth *auth.Details, showInfo bool, uiConfig *config.UIConfig) *AuthInfo {
	return &AuthInfo{authdetails: auth, showInfo: showInfo, uiConfig: uiConfig}
}

func (a *AuthInfo) Draw(ctx *ui.Context) {
	defaultStyle := a.uiConfig.GetStyle(config.STYLE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)
	var text string
	switch {
	case a.authdetails == nil:
		text = "(no header)"
		ctx.Printf(0, 0, defaultStyle, "%s", text)
	case a.authdetails.Err != nil:
		style := a.uiConfig.GetStyle(config.STYLE_ERROR)
		text = a.authdetails.Err.Error()
		ctx.Printf(0, 0, style, "%s", text)
	default:
		checkBounds := func(x int) bool {
			return x < ctx.Width()
		}
		setResult := func(result auth.Result) (string, vaxis.Style) {
			switch result {
			case auth.ResultNone:
				return "none", defaultStyle
			case auth.ResultNeutral:
				return "neutral", a.uiConfig.GetStyle(config.STYLE_WARNING)
			case auth.ResultPolicy:
				return "policy", a.uiConfig.GetStyle(config.STYLE_WARNING)
			case auth.ResultPass:
				return "✓", a.uiConfig.GetStyle(config.STYLE_SUCCESS)
			case auth.ResultFail:
				return "✗", a.uiConfig.GetStyle(config.STYLE_ERROR)
			default:
				return string(result), a.uiConfig.GetStyle(config.STYLE_ERROR)
			}
		}
		x := 1
		for i := 0; i < len(a.authdetails.Results); i++ {
			if checkBounds(x) {
				text, style := setResult(a.authdetails.Results[i])
				if i > 0 {
					text = " " + text
				}
				x += ctx.Printf(x, 0, style, "%s", text)
			}
		}
		if a.showInfo {
			infoText := ""
			for i := 0; i < len(a.authdetails.Infos); i++ {
				if i > 0 {
					infoText += ","
				}
				infoText += a.authdetails.Infos[i]
				if reason := a.authdetails.Reasons[i]; reason != "" {
					infoText += reason
				}
			}
			if checkBounds(x) && infoText != "" {
				if trunc := ctx.Width() - x - 3; trunc > 0 {
					text = runewidth.Truncate(infoText, trunc, "…")
					ctx.Printf(x, 0, defaultStyle, " (%s)", text)
				}
			}
		}
	}
}

func (a *AuthInfo) Invalidate() {
	ui.Invalidate()
}
