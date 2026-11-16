#!/usr/bin/env python3
import json
import re
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DATE = re.compile(r"\b(\d{4}-\d{2}-\d{2})\b")


def answer(req):
    history = req.get("history") or []
    last = history[-1] if history else {"role": "user", "content": ""}
    tools = {t["name"] for t in req.get("tools") or []}

    if last["role"] == "tool":
        result = json.loads(last["content"]).get("result", {})
        if not result.get("ok", False):
            return {"text": "Не получилось: " + str(result.get("error", "ошибка инструмента"))}
        if result.get("pending_action"):
            return {"text": "Подготовил заявку: %s\nПодтвердите её в карточке." % result["pending_action"]["summary"]}
        return {"text": "Готово."}

    text = last["content"]
    low = text.lower()
    date = DATE.search(text)
    if "отгул" in low and date and "create_absence_request" in tools:
        return {"tool_call": {"name": "create_absence_request", "args": {"date_from": date.group(1)}}}
    if "прогресс" in low and "get_my_progress" in tools:
        return {"tool_call": {"name": "get_my_progress", "args": {}}}
    if req.get("kb"):
        return {"text": "По базе знаний: " + req["kb"][0]["content"]}
    return {"text": "В базе знаний ничего не нашёл. Лучше уточнить у наставника."}


class Handler(BaseHTTPRequestHandler):
    def _read(self):
        return json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")

    def do_POST(self):
        if self.path == "/chat":
            body = json.dumps(answer(self._read()), ensure_ascii=False).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        elif self.path == "/chat/stream":
            resp = answer(self._read())
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            for word in (resp.get("text") or "").split(" "):
                self.wfile.write(("data: %s\n\n" % json.dumps({"token": word + " "}, ensure_ascii=False)).encode())
            done = dict(resp, done=True)
            self.wfile.write(("data: %s\n\n" % json.dumps(done, ensure_ascii=False)).encode())
        else:
            self.send_error(404)

    def log_message(self, fmt, *args):
        sys.stderr.write("fake-ml: " + fmt % args + "\n")


if __name__ == "__main__":
    ThreadingHTTPServer(("", 8000), Handler).serve_forever()
