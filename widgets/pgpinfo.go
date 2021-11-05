package widgets

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib/ui"

	"golang.org/x/crypto/openpgp"
	pgperrors "golang.org/x/crypto/openpgp/errors"
)

type PGPInfo struct {
	ui.Invalidatable
	details  *openpgp.MessageDetails
	uiConfig config.UIConfig
}

func NewPGPInfo(details *openpgp.MessageDetails, uiConfig config.UIConfig) *PGPInfo {
	return &PGPInfo{details: details, uiConfig: uiConfig}
}

func (p *PGPInfo) DrawSignature(ctx *ui.Context) {
	errorStyle := p.uiConfig.GetStyle(config.STYLE_ERROR)
	warningStyle := p.uiConfig.GetStyle(config.STYLE_WARNING)
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)

	// TODO: Nicer prompt for TOFU, fetch from keyserver, etc
	if errors.Is(p.details.SignatureError, pgperrors.ErrUnknownIssuer) ||
		p.details.SignedBy == nil {

		x := ctx.Printf(0, 0, warningStyle, "*")
		x += ctx.Printf(x, 0, defaultStyle,
			" Signed with unknown key (%8X); authenticity unknown",
			p.details.SignedByKeyId)
	} else if p.details.SignatureError != nil {
		x := ctx.Printf(0, 0, errorStyle, "Invalid signature!")
		x += ctx.Printf(x, 0, errorStyle,
			" This message may have been tampered with! (%s)",
			p.details.SignatureError.Error())
	} else {
		entity := p.details.SignedBy.Entity
		ident := entity.PrimaryIdentity()

		x := ctx.Printf(0, 0, validStyle, "✓ Authentic ")
		x += ctx.Printf(x, 0, defaultStyle,
			"Signature from %s (%8X)",
			ident.Name, p.details.SignedByKeyId)
	}
}

func (p *PGPInfo) DrawEncryption(ctx *ui.Context, y int) {
	validStyle := p.uiConfig.GetStyle(config.STYLE_SUCCESS)
	defaultStyle := p.uiConfig.GetStyle(config.STYLE_DEFAULT)
	entity := p.details.DecryptedWith.Entity
	ident := entity.PrimaryIdentity()

	x := ctx.Printf(0, y, validStyle, "✓ Encrypted ")
	x += ctx.Printf(x, y, defaultStyle,
		"To %s (%8X) ", ident.Name, p.details.DecryptedWith.PublicKey.KeyId)
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
