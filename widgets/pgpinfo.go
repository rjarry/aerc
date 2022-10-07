package widgets

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/gdamore/tcell/v2"
)

type PGPInfo struct {
	details  *models.MessageDetails
	uiConfig *config.UIConfig
}

func NewPGPInfo(details *models.MessageDetails, uiConfig *config.UIConfig) *PGPInfo {
	return &PGPInfo{details: details, uiConfig: uiConfig}
}

func (p *PGPInfo) DrawSignature(ctx *ui.Context) {
	errorStyle := p.uiConfig.GetStyle(config.STYLE_ERROR)
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	var icon string
	var indicatorStyle, textstyle tcell.Style
	textstyle = defaultStyle
	var indicatorText, messageText string
	// TODO: Nicer prompt for TOFU, fetch from keyserver, etc
	switch p.details.SignatureValidity {
	case models.UnknownEntity:
		icon = p.uiConfig.IconUnknown
		indicatorStyle = warningStyle
		indicatorText = "Unknown"
		messageText = fmt.Sprintf("Signed with unknown key (%8X); authenticity unknown", p.details.SignedByKeyId)
	case models.Valid:
		icon = p.uiConfig.IconSigned
		if p.details.IsEncrypted && p.uiConfig.IconSignedEncrypted != "" {
			icon = p.uiConfig.IconSignedEncrypted
		}
		indicatorStyle = validStyle
		indicatorText = "Authentic"
		messageText = fmt.Sprintf("Signature from %s (%8X)", p.details.SignedBy, p.details.SignedByKeyId)
	default:
		icon = p.uiConfig.IconInvalid
		indicatorStyle = errorStyle
		indicatorText = "Invalid signature!"
		messageText = fmt.Sprintf("This message may have been tampered with! (%s)", p.details.SignatureError)
	}

	x := ctx.Printf(0, 0, indicatorStyle, "%s %s ", icon, indicatorText)
	ctx.Printf(x, 0, textstyle, messageText)
}

func (p *PGPInfo) DrawEncryption(ctx *ui.Context, y int) {
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	// if a sign-encrypt combination icon is set, use that
	icon := p.uiConfig.IconEncrypted
	if p.details.IsSigned && p.details.SignatureValidity == models.Valid && p.uiConfig.IconSignedEncrypted != "" {
		icon = strings.Repeat(" ", utf8.RuneCountInString(p.uiConfig.IconSignedEncrypted))
	}

	x := ctx.Printf(0, y, validStyle, "%s Encrypted", icon)
	x += ctx.Printf(x+1, y, defaultStyle, "To %s (%8X) ", p.details.DecryptedWith, p.details.DecryptedWithKeyId)
	if !p.details.IsSigned {
		ctx.Printf(x, y, warningStyle, "(message not signed!)")
	}
}

func (p *PGPInfo) Draw(ctx *ui.Context) {
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)

	switch {
	case p.details == nil && p.uiConfig.IconUnencrypted != "":
		x := ctx.Printf(0, 0, warningStyle, "%s ", p.uiConfig.IconUnencrypted)
		ctx.Printf(x, 0, defaultStyle, "message unencrypted and unsigned")
	case p.details.IsSigned && p.details.IsEncrypted:
		p.DrawSignature(ctx)
		p.DrawEncryption(ctx, 1)
	case p.details.IsSigned:
		p.DrawSignature(ctx)
	case p.details.IsEncrypted:
		p.DrawEncryption(ctx, 0)
	}
}

func (p *PGPInfo) Invalidate() {
	ui.Invalidate()
}
