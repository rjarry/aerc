package widgets

import (
	"errors"

	"git.sr.ht/~sircmpwn/aerc/lib/ui"

	"github.com/gdamore/tcell"
	"golang.org/x/crypto/openpgp"
	pgperrors "golang.org/x/crypto/openpgp/errors"
)

type PGPInfo struct {
	ui.Invalidatable
	details *openpgp.MessageDetails
}

func NewPGPInfo(details *openpgp.MessageDetails) *PGPInfo {
	return &PGPInfo{details: details}
}

func (p *PGPInfo) DrawSignature(ctx *ui.Context, offs bool) {
	errorStyle := tcell.StyleDefault.Background(tcell.ColorRed).
		Foreground(tcell.ColorWhite).Bold(true)
	softErrorStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).
		Reverse(true).Bold(true)
	validStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	header := "Signature "
	if offs {
		header += " "
	}

	// TODO: Nicer prompt for TOFU, fetch from keyserver, etc
	if errors.Is(p.details.SignatureError, pgperrors.ErrUnknownIssuer) ||
		p.details.SignedBy == nil {

		x := ctx.Printf(0, 0, tcell.StyleDefault.Bold(true), "%s", header)
		x += ctx.Printf(x, 0, softErrorStyle, " Unknown ")
		x += ctx.Printf(x, 0, tcell.StyleDefault,
			" Signed with unknown key (%8X); authenticity unknown",
			p.details.SignedByKeyId)
	} else if p.details.SignatureError != nil {
		x := ctx.Printf(0, 0, tcell.StyleDefault.Bold(true), "%s", header)
		x += ctx.Printf(x, 0, errorStyle, " ✗ Invalid! ")
		x += ctx.Printf(x, 0, tcell.StyleDefault.
			Foreground(tcell.ColorRed).Bold(true),
			" This message may have been tampered with! (%s)",
			p.details.SignatureError.Error())
	} else {
		entity := p.details.SignedBy.Entity
		var ident *openpgp.Identity
		// TODO: Pick identity more intelligently
		for _, ident = range entity.Identities {
			break
		}
		ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', validStyle)
		x := ctx.Printf(0, 0, tcell.StyleDefault.Bold(true), "%s", header)
		x += ctx.Printf(x, 0, validStyle, "✓ Signed ")
		x += ctx.Printf(x, 0, tcell.StyleDefault,
			"by %s (%8X)", ident.Name, p.details.SignedByKeyId)
	}
}

func (p *PGPInfo) DrawEncryption(ctx *ui.Context, y int) {
	validStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	entity := p.details.DecryptedWith.Entity
	var ident *openpgp.Identity
	// TODO: Pick identity more intelligently
	for _, ident = range entity.Identities {
		break
	}

	x := ctx.Printf(0, y, tcell.StyleDefault.Bold(true), "Encryption ")
	x += ctx.Printf(x, y, validStyle, "✓ Encrypted ")
	x += ctx.Printf(x, y, tcell.StyleDefault,
		"for %s (%8X) ", ident.Name, p.details.DecryptedWith.PublicKey.KeyId)
}

func (p *PGPInfo) Draw(ctx *ui.Context) {
	ctx.Fill(0, 0, ctx.Width(), ctx.Height(), ' ', tcell.StyleDefault)
	if p.details.IsSigned && p.details.IsEncrypted {
		p.DrawSignature(ctx, true)
		p.DrawEncryption(ctx, 1)
	} else if p.details.IsSigned {
		p.DrawSignature(ctx, false)
	} else if p.details.IsEncrypted {
		p.DrawEncryption(ctx, 0)
	}
}

func (p *PGPInfo) Invalidate() {
	p.DoInvalidate(p)
}
