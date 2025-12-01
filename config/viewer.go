package config

import (
	"regexp"
	"sync/atomic"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

type ViewerConfig struct {
	Pager            string     `ini:"pager" default:"less -Rc"`
	Alternatives     []string   `ini:"alternatives" default:"text/plain,text/html" delim:","`
	ShowHeaders      bool       `ini:"show-headers"`
	AlwaysShowMime   bool       `ini:"always-show-mime"`
	MaxMimeHeight    int        `ini:"max-mime-height" default:"0"`
	ParseHttpLinks   bool       `ini:"parse-http-links" default:"true"`
	HtmlInlineImages bool       `ini:"html-inline-images"`
	HeaderLayout     [][]string `ini:"header-layout" parse:"ParseLayout" default:"From|To,Cc|Bcc,Date,Subject"`
	KeyPassthrough   bool

	// private
	contextualViewers []*ViewerConfigContext
	contextualCounts  map[viewerContextType]int
	contextualCache   map[viewerContextKey]*ViewerConfig
}

type viewerContextType int

const (
	viewerContextSender viewerContextType = iota
	viewerContextFrom
	viewerContextSubject
)

type ViewerConfigContext struct {
	ContextType  viewerContextType
	Value        string
	Regex        *regexp.Regexp
	ViewerConfig *ViewerConfig
	Section      ini.Section
}

type viewerContextKey struct {
	ctxType viewerContextType
	value   string
}

var viewerConfig atomic.Pointer[ViewerConfig]

func Viewer() *ViewerConfig {
	return viewerConfig.Load()
}

var viewerContextualSectionRe = regexp.MustCompile(`^viewer:(sender|from|subject)([~=])(.+)$`)

func parseViewer(file *ini.File) (*ViewerConfig, error) {
	conf := &ViewerConfig{
		contextualCounts: make(map[viewerContextType]int),
		contextualCache:  make(map[viewerContextKey]*ViewerConfig),
	}
	if err := conf.parse(file.Section("viewer"), true); err != nil {
		return nil, err
	}

	for _, section := range file.Sections() {
		var err error
		groups := viewerContextualSectionRe.FindStringSubmatch(section.Name())
		if groups == nil {
			continue
		}
		ctx, separator, value := groups[1], groups[2], groups[3]

		viewerSubConfig := ViewerConfig{}
		if err = viewerSubConfig.parse(section, false); err != nil {
			return nil, err
		}
		contextualViewer := ViewerConfigContext{
			ViewerConfig: &viewerSubConfig,
			Section:      *section,
		}

		switch ctx {
		case "sender":
			contextualViewer.ContextType = viewerContextSender
		case "from":
			contextualViewer.ContextType = viewerContextFrom
		case "subject":
			contextualViewer.ContextType = viewerContextSubject
		}
		if separator == "=" {
			contextualViewer.Value = value
		} else {
			contextualViewer.Regex, err = regexp.Compile(value)
			if err != nil {
				return nil, err
			}
		}

		conf.contextualViewers = append(conf.contextualViewers, &contextualViewer)
		conf.contextualCounts[contextualViewer.ContextType]++
	}

	log.Debugf("aerc.conf: [viewer] %#v", conf)
	return conf, nil
}

func (config *ViewerConfig) parse(section *ini.Section, useDefaults bool) error {
	return MapToStruct(section, config, useDefaults)
}

func (v *ViewerConfig) ParseLayout(sec *ini.Section, key *ini.Key) ([][]string, error) {
	layout := parseLayout(key.String())
	return layout, nil
}

func (base *ViewerConfig) mergeContextual(
	contextType viewerContextType, matcher func(*ViewerConfigContext) bool,
) *ViewerConfig {
	for _, contextualViewer := range base.contextualViewers {
		if contextualViewer.ContextType != contextType {
			continue
		}
		if !matcher(contextualViewer) {
			continue
		}
		viewer := *base
		err := viewer.parse(&contextualViewer.Section, false)
		if err != nil {
			log.Warnf("merge viewer failed: %v", err)
		}
		viewer.contextualCache = make(map[viewerContextKey]*ViewerConfig)
		viewer.contextualCounts = base.contextualCounts
		viewer.contextualViewers = base.contextualViewers
		return &viewer
	}
	return base
}

func (base *ViewerConfig) contextual(
	ctxType viewerContextType, ctxKey string, matcher func(*ViewerConfigContext) bool,
) *ViewerConfig {
	if base.contextualCounts[ctxType] == 0 {
		// shortcut if no contextual viewer for that type
		return base
	}
	key := viewerContextKey{ctxType: ctxType, value: ctxKey}
	ctx, found := base.contextualCache[key]
	if !found {
		ctx = base.mergeContextual(ctxType, matcher)
		base.contextualCache[key] = ctx
	}
	return ctx
}

func makeMatcher(
	valueTarget string, regexTarget string,
) func(ctx *ViewerConfigContext) bool {
	return func(ctx *ViewerConfigContext) bool {
		if ctx.Value != "" && ctx.Value == valueTarget {
			return true
		}
		if ctx.Regex != nil && ctx.Regex.Match([]byte(regexTarget)) {
			return true
		}
		return false
	}
}

func (base *ViewerConfig) forAddresses(
	ctxType viewerContextType, addresses []*mail.Address,
) *ViewerConfig {
	for _, address := range addresses {
		base = base.contextual(
			ctxType, address.String(),
			makeMatcher(address.Address, address.String()),
		)
	}
	return base
}

func (base *ViewerConfig) ForEnvelope(envelope *models.Envelope) *ViewerConfig {
	if envelope == nil {
		return base
	}
	base = base.forAddresses(viewerContextSender, envelope.Sender)
	base = base.forAddresses(viewerContextFrom, envelope.From)
	return base.contextual(
		viewerContextSubject, envelope.Subject,
		makeMatcher(envelope.Subject, envelope.Subject),
	)
}
