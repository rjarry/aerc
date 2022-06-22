package widgets

import (
	"strings"
	"unicode/utf8"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/models"
)

type PGPInfo struct {
	ui.Invalidatable
	details  *models.MessageDetails
	uiConfig config.UIConfig
}

func NewPGPInfo(details *models.MessageDetails, uiConfig config.UIConfig) *PGPInfo {
	return &PGPInfo{details: details, uiConfig: uiConfig}
}

func (p *PGPInfo) DrawSignature(ctx *ui.Context) {
	errorStyle := p.uiConfig.GetStyle(config.STYLE_ERROR)
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	// TODO: Nicer prompt for TOFU, fetch from keyserver, etc
	if p.details.SignatureValidity == models.UnknownEntity ||
		p.details.SignedBy == "" {

		x := ctx.Printf(0, 0, warningStyle, "%s unknown", p.uiConfig.IconUnknown)
		x += ctx.Printf(x, 0, defaultStyle,
			" Signed with unknown key (%8X); authenticity unknown",
			p.details.SignedByKeyId)
	} else if p.details.SignatureValidity != models.Valid {
		x := ctx.Printf(0, 0, errorStyle, "%s Invalid signature!", p.uiConfig.IconInvalid)
		x += ctx.Printf(x, 0, errorStyle,
			" This message may have been tampered with! (%s)",
			p.details.SignatureError)
	} else {
		icon := p.uiConfig.IconSigned
		if p.details.IsEncrypted {
			icon = p.uiConfig.IconSignedEncrypted
		}
		x := ctx.Printf(0, 0, validStyle, "%s Authentic ", icon)
		x += ctx.Printf(x, 0, defaultStyle,
			"Signature from %s (%8X)",
			p.details.SignedBy, p.details.SignedByKeyId)
	}
}

func (p *PGPInfo) DrawEncryption(ctx *ui.Context, y int) {
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	icon := p.uiConfig.IconEncrypted
	if p.details.IsSigned && p.details.SignatureValidity == models.Valid {
		icon = strings.Repeat(" ", utf8.RuneCountInString(p.uiConfig.IconSignedEncrypted))
	}

	x := ctx.Printf(0, y, validStyle, "%s Encrypted ", icon)
	x += ctx.Printf(x, y, defaultStyle,
		"To %s (%8X) ", p.details.DecryptedWith, p.details.DecryptedWithKeyId)
	if !p.details.IsSigned {
		x += ctx.Printf(x, y, warningStyle,
			"(message not signed!)")
	}
}

func (p *PGPInfo) Draw(ctx *ui.Context) {
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', defaultStyle)
	if p.details.IsSigned && p.details.IsEncrypted {
		p.DrawSignature(ctx)
		p.DrawEncryption(ctx, 1)
	} else if p.details.IsSigned {
		p.DrawSignature(ctx)
	} else if p.details.IsEncrypted {
		p.DrawEncryption(ctx, 0)
	}
}

func (p *PGPInfo) Invalidate() {
	p.DoInvalidate(p)
}
