#!/usr/bin/env python3
"""PDJ lightweight backend.

- HTTP API: GET/POST /api/config
- WS relay: ws://127.0.0.1:23333/danmu/sub
- Upstream: wss://broadcastlv.chat.bilibili.com/sub
"""

from __future__ import annotations

import asyncio
import json
import logging
import struct
import threading
import time
import zlib
from dataclasses import dataclass
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

import websockets
from websockets.server import WebSocketServerProtocol

ROOT = Path(__file__).resolve().parent
CONFIG_PATH = ROOT / "pdj" / "dograin" / "pdj_config.json"
LOCAL_WS_HOST = "127.0.0.1"
LOCAL_WS_PORT = 23333
UPSTREAM_WS_URL = "wss://broadcastlv.chat.bilibili.com/sub"

logging.basicConfig(level=logging.INFO, format="[%(asctime)s] %(levelname)s %(message)s")
LOG = logging.getLogger("pdj_server")


DEFAULT_CONFIG = {
    "roomid": 26714219,
    "uid": 0,
    "cookie": "",
}


@dataclass
class PDJConfig:
    roomid: int
    uid: int
    cookie: str


class ConfigStore:
    def __init__(self, path: Path):
        self.path = path
        self._lock = threading.Lock()
        self.path.parent.mkdir(parents=True, exist_ok=True)
        if not self.path.exists():
            self.save(DEFAULT_CONFIG)

    def load(self) -> dict[str, Any]:
        with self._lock:
            try:
                data = json.loads(self.path.read_text(encoding="utf-8"))
            except Exception:
                data = DEFAULT_CONFIG.copy()
                self.save(data)
            return {
                "roomid": int(data.get("roomid") or 0),
                "uid": int(data.get("uid") or 0),
                "cookie": str(data.get("cookie") or ""),
            }

    def save(self, data: dict[str, Any]) -> dict[str, Any]:
        normalized = {
            "roomid": int(data.get("roomid") or 0),
            "uid": int(data.get("uid") or 0),
            "cookie": str(data.get("cookie") or ""),
        }
        with self._lock:
            self.path.write_text(json.dumps(normalized, ensure_ascii=False, indent=2), encoding="utf-8")
        return normalized


CONFIG = ConfigStore(CONFIG_PATH)


class ConfigHandler(BaseHTTPRequestHandler):
    def _send_json(self, payload: dict[str, Any], status: HTTPStatus = HTTPStatus.OK):
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status.value)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()
        self.wfile.write(data)

    def do_OPTIONS(self):
        self._send_json({"ok": True})

    def do_GET(self):
        if self.path == "/api/config":
            self._send_json(CONFIG.load())
            return
        self._send_json({"error": "not found"}, HTTPStatus.NOT_FOUND)

    def do_POST(self):
        if self.path != "/api/config":
            self._send_json({"error": "not found"}, HTTPStatus.NOT_FOUND)
            return

        try:
            length = int(self.headers.get("Content-Length", "0"))
            body = self.rfile.read(length)
            payload = json.loads(body.decode("utf-8"))
            saved = CONFIG.save(payload)
            self._send_json({"ok": True, "config": saved})
            relay.notify_config_changed()
        except Exception as exc:
            self._send_json({"ok": False, "error": str(exc)}, HTTPStatus.BAD_REQUEST)


def encode_packet(payload: dict[str, Any], op: int = 7) -> bytes:
    body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
    packet_len = 16 + len(body)
    header = struct.pack(">IHHII", packet_len, 16, 1, op, 1)
    return header + body


def decode_packets(blob: bytes) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    offset = 0
    while offset + 16 <= len(blob):
        packet_len, header_len, ver, op, _seq = struct.unpack(">IHHII", blob[offset : offset + 16]
        )
        if packet_len < 16:
            break
        body = blob[offset + header_len : offset + packet_len]
        offset += packet_len

        if op == 3:
            popularity = struct.unpack(">I", body[:4])[0] if len(body) >= 4 else 0
            out.append({"type": "PDJ_STATUS", "status": "popularity", "popularity": popularity})
            continue

        if op != 5:
            continue

        if ver == 2:
            try:
                out.extend(decode_packets(zlib.decompress(body)))
            except Exception:
                LOG.warning("zlib decompress failed", exc_info=True)
            continue

        try:
            text = body.decode("utf-8", errors="ignore").strip("\x00")
            if not text:
                continue
            out.append(json.loads(text))
        except Exception:
            continue
    return out


class RelayService:
    def __init__(self):
        self.clients: set[WebSocketServerProtocol] = set()
        self._config_version = 0

    def notify_config_changed(self):
        self._config_version += 1

    async def broadcast_json(self, payload: dict[str, Any]):
        if not self.clients:
            return
        text = json.dumps(payload, ensure_ascii=False)
        dead: list[WebSocketServerProtocol] = []
        for cli in self.clients:
            try:
                await cli.send(text)
            except Exception:
                dead.append(cli)
        for d in dead:
            self.clients.discard(d)

    async def client_handler(self, websocket: WebSocketServerProtocol):
        if websocket.path != "/danmu/sub":
            await websocket.close(code=1008, reason="invalid path")
            return

        self.clients.add(websocket)
        await websocket.send(json.dumps({"type": "PDJ_STATUS", "status": "client_connected"}, ensure_ascii=False))
        try:
            async for _msg in websocket:
                pass
        finally:
            self.clients.discard(websocket)

    async def run_upstream(self):
        current_version = -1
        while True:
            cfg = CONFIG.load()
            current_version = self._config_version
            auth_body = {
                "uid": int(cfg["uid"] or 0),
                "roomid": int(cfg["roomid"] or 0),
                "protover": 2,
                "platform": "web",
                "type": 2,
                "clientver": "1.16.3",
            }
            headers = {}
            if cfg.get("cookie"):
                headers["Cookie"] = cfg["cookie"]

            try:
                await self.broadcast_json({"type": "PDJ_STATUS", "status": "connecting", "roomid": cfg["roomid"]})
                async with websockets.connect(UPSTREAM_WS_URL, extra_headers=headers, ping_interval=None) as up:
                    await up.send(encode_packet(auth_body, op=7))
                    await self.broadcast_json({"type": "PDJ_STATUS", "status": "connected", "roomid": cfg["roomid"]})

                    last_hb = time.time()
                    while True:
                        if self._config_version != current_version:
                            await self.broadcast_json({"type": "PDJ_STATUS", "status": "config_changed"})
                            break

                        if time.time() - last_hb > 30:
                            await up.send(encode_packet({}, op=2))
                            last_hb = time.time()

                        try:
                            raw = await asyncio.wait_for(up.recv(), timeout=1)
                        except asyncio.TimeoutError:
                            continue

                        if isinstance(raw, str):
                            continue

                        for item in decode_packets(raw):
                            await self.broadcast_json(item)
            except Exception as exc:
                await self.broadcast_json({"type": "PDJ_STATUS", "status": "upstream_error", "error": str(exc)})
                LOG.warning("upstream disconnected: %s", exc)
                await asyncio.sleep(2)


relay = RelayService()


def run_http_server():
    server = ThreadingHTTPServer(("127.0.0.1", 23334), ConfigHandler)
    LOG.info("HTTP API listening on http://127.0.0.1:23334")
    server.serve_forever()


async def run_ws_server():
    LOG.info("WS relay listening on ws://%s:%s/danmu/sub", LOCAL_WS_HOST, LOCAL_WS_PORT)
    async with websockets.serve(relay.client_handler, LOCAL_WS_HOST, LOCAL_WS_PORT, max_size=2**23):
        await relay.run_upstream()


def main():
    http_thread = threading.Thread(target=run_http_server, daemon=True)
    http_thread.start()
    asyncio.run(run_ws_server())


if __name__ == "__main__":
    main()
