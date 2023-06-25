from supybot import conf, registry
try:
    from supybot.i18n import PluginInternationalization
    _ = PluginInternationalization('Sourcehut')
except:
    _ = lambda x: x


def configure(advanced):
    from supybot.questions import expect, anything, something, yn
    conf.registerPlugin('Sourcehut', True)


Sourcehut = conf.registerPlugin('Sourcehut')
