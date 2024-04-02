import email.header
import email.utils
import json
import mailbox
from urllib.parse import quote
from urllib.request import urlopen

from supybot import callbacks, httpserver, ircmsgs, log, world
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


def decode_header(header: str) -> str:
    if not header:
        return ""
    text = ""
    for chunk, encoding in email.header.decode_header(header):
        if isinstance(chunk, bytes):
            chunk = chunk.decode(encoding or "us-ascii")
        text += chunk
    return text


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
        refs = []
        for header in mail['references']:
            refs += header.split()
        for ref in refs:
            url = self.URL.format(**mail) + quote(f"/{ref}")
            print(f"GET {url}/raw")
            with urlopen(f"{url}/raw") as u:
                msg = mailbox.Message(u.read())
            subject = decode_header(msg["subject"])
            if not subject.startswith("[PATCH"):
                continue
            for name, addr in email.utils.getaddresses([decode_header(msg["from"])]):
                if name:
                    submitter = name
                else:
                    submitter = addr
                msg = f"{bold(mircColor('applied', 'green'))} {bold(subject)}"
                msg += f" from {italic(submitter)}: {underline(url)}"
                self.plugin.announce(channel, msg)
                return

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
                if hook["email"]["patchset_update"] == ["APPLIED"]:
                    self.announce_apply(hook["email"])
                handler.send_response(200)
                handler.end_headers()
                handler.wfile.write(b"")
                return

            raise ValueError(f"unsupported webhook: {hook}")

        except Exception as e:
            print("ERROR", e)
            handler.send_response(400)
            handler.end_headers()
            handler.wfile.write(b"Bad request\n")

    def log_message(self, format, *args):
        pass


Class = Sourcehut
