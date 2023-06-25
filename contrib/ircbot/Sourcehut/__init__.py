"""
Sourcehut: Supybot plugin to receive Sourcehut webhooks
"""

import sys
import supybot

__version__ = "0.1"
__author__ = supybot.authors.unknown
__contributors__ = {}
__url__ = ''

from . import config
from . import plugin
from importlib import reload

reload(config)
reload(plugin)

Class = plugin.Class
configure = config.configure
