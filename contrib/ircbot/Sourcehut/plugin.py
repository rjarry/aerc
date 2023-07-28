import json

from supybot import ircmsgs, callbacks, httpserver, log, world
from supybot.ircutils import bold, italic, underline


class Sourcehut(callbacks.Plugin):
    """
    Supybot plugin to receive Sourcehut webhooks
    """

    def __init__(self, irc):
        super().__init__(irc)
        httpserver.hook("sourcehut", SourcehutServerCallback(self))

    def die(self):
        httpserver.unhook("sourcehut")
        super().die()

    def announce(self, channel, message):
        libera = world.getIrc("libera")
        if libera is None:
            print("error: no irc libera")
            return
        if channel not in libera.state.channels:
            print(f"error: not in {channel} channel")
            return
        libera.sendMsg(ircmsgs.notice(channel, message))


class SourcehutServerCallback(httpserver.SupyHTTPServerCallback):
    name = "Sourcehut"
    defaultResponse = "Bad request\n"

    def __init__(self, plugin: Sourcehut):
        super().__init__()
        self.plugin = plugin

    SUBJECT = "[PATCH {prefix} v{version}] {subject}"
    URL = "https://lists.sr.ht/{list[owner][canonicalName]}/{list[name]}/patches/{id}"
    CHANS = {
        "#public-inbox": "##rjarry",
        "#aerc-devel": "#aerc",
    }

    def doPost(self, handler, path, form=None):
        if hasattr(form, "decode"):
            form = form.decode("utf-8")
        print(f"POST {path} {form}")
        try:
            body = json.loads(form)
            hook = body["data"]["webhook"]
            if hook["event"] == "PATCHSET_RECEIVED":
                patchset = hook["patchset"]
                subject = self.SUBJECT.format(**patchset)
                url = self.URL.format(**patchset)
                if not url.startswith("https://lists.sr.ht/~rjarry/"):
                    raise ValueError("unknown list")
                channel = f"#{patchset['list']['name']}"
                channel = self.CHANS.get(channel, channel)
                try:
                    submitter = patchset["submitter"]["canonicalName"]
                except KeyError:
                    try:
                        submitter = patchset["submitter"]["name"]
                    except KeyError:
                        submitter = patchset["submitter"]["address"]
                msg = f"received {bold(subject)} from {italic(submitter)}: {underline(url)}"
                self.plugin.announce(channel, msg)
                handler.send_response(200)
                handler.end_headers()
                handler.wfile.write(b"")
                return

            raise ValueError("unsupported webhook: %r" % hook)

        except Exception as e:
            print("ERROR", e)
            handler.send_response(400)
            handler.end_headers()
            handler.wfile.write(b"Bad request\n")

    def log_message(self, format, *args):
        pass


Class = Sourcehut
