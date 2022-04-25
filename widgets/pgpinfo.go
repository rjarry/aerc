package widgets

import (
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

		x := ctx.Printf(0, 0, warningStyle, "*")
		x += ctx.Printf(x, 0, defaultStyle,
			" Signed with unknown key (%8X); authenticity unknown",
			p.details.SignedByKeyId)
	} else if p.details.SignatureValidity != models.Valid {
		x := ctx.Printf(0, 0, errorStyle, "Invalid signature!")
		x += ctx.Printf(x, 0, errorStyle,
			" This message may have been tampered with! (%s)",
			p.details.SignatureError)
	} else {
		x := ctx.Printf(0, 0, validStyle, "✓ Authentic ")
		x += ctx.Printf(x, 0, defaultStyle,
			"Signature from %s (%8X)",
			p.details.SignedBy, p.details.SignedByKeyId)
	}
}

func (p *PGPInfo) DrawEncryption(ctx *ui.Context, y int) {
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	x := ctx.Printf(0, y, validStyle, "✓ Encrypted ")
	x += ctx.Printf(x, y, defaultStyle,
		"To %s (%8X) ", p.details.DecryptedWith, p.details.DecryptedWithKeyId)
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
