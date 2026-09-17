import json
from urllib.parse import quote
import re
import traceback

from supybot import callbacks, httpserver, ircmsgs, world
from supybot.ircutils import bold, italic, mircColor, underline


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
    URL = "https://lists.sr.ht/{list[owner][canonicalName]}/{list[name]}"
    CHANS = {
        "#public-inbox": "##rjarry",
        "#aerc-devel": "#aerc",
    }

    def announce_patch(self, patchset):
        subject = self.SUBJECT.format(**patchset)
        url = self.URL.format(**patchset)
        if not url.startswith("https://lists.sr.ht/~rjarry/"):
            raise ValueError("unknown list")
        url += "/patches/{id}".format(**patchset)
        channel = f"#{patchset['list']['name']}"
        channel = self.CHANS.get(channel, channel)
        try:
            submitter = patchset["submitter"]["canonicalName"]
        except KeyError:
            try:
                submitter = patchset["submitter"]["name"]
            except KeyError:
                submitter = patchset["submitter"]["address"]
        msg = f"{mircColor('received', 'light gray')} {bold(subject)}"
        msg += f" from {italic(submitter)}: {underline(url)}"
        self.plugin.announce(channel, msg)

    def announce_apply(self, mail):
        channel = f"#{mail['list']['name']}"
        channel = self.CHANS.get(channel, channel)
        root = mail["thread"]["root"]
        subject = re.sub(r"\s+", " ", root["subject"])
        if not re.match(r"^\[(RFC )?PATCH", subject):
            return
        url = self.URL.format(**mail) + quote(f"/{root['messageID']}")
        try:
            submitter = root["sender"]["canonicalName"]
        except KeyError:
            try:
                submitter = root["sender"]["name"]
            except KeyError:
                submitter = root["sender"]["address"]
        msg = f"{bold(mircColor('applied', 'green'))} {bold(subject)}"
        msg += f" from {italic(submitter)}: {underline(url)}"
        self.plugin.announce(channel, msg)

    def doPost(self, handler, path, form=None):
        if hasattr(form, "decode"):
            form = form.decode("utf-8")
        print(f"POST {path} {form}")
        try:
            body = json.loads(form)
            hook = body["data"]["webhook"]
            if hook["event"] == "PATCHSET_RECEIVED":
                self.announce_patch(hook["patchset"])
                handler.send_response(200)
                handler.end_headers()
                handler.wfile.write(b"")
                return

            if hook["event"] == "EMAIL_RECEIVED":
                if (hook["email"]["patchset_update"] == ["APPLIED"] or
                    appliedByProtonUser(hook["email"])):
                    self.announce_apply(hook["email"])
                handler.send_response(200)
                handler.end_headers()
                handler.wfile.write(b"")
                return

            raise ValueError(f"unsupported webhook: {hook}")

        except Exception as e:
            traceback.print_exception(e)
            handler.send_response(400)
            handler.end_headers()
            handler.wfile.write(b"Bad request\n")

    def log_message(self, format, *args):
        pass

    def appliedByProtonUser(email):
        # Workaround for maintainers using Proton, that strips mail headers
        root = email["thread"]["root"]
        if not "canonicalName" in root["sender"]:
            return False
        if root["sender"]["canonicalName"] != "~simartin":
            return False
        subject = re.sub(r"\s+", " ", root["subject"])
        return subject.startswith("Applied: [PATCH aerc")

Class = Sourcehut
