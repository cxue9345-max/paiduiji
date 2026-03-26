import tkinter as tk
from tkinter import scrolledtext
import asyncio
import websockets
import json
import threading

# 定义WebSocket服务端类
class WebSocketServer:
    def __init__(self):
        self.websocket = None
        self.connected = False
        self.lock = threading.Lock()
    
    async def handler(self, websocket, path):
        self.websocket = websocket
        self.connected = True
        try:
            async for message in websocket:
                print(f"Received message: {message}")
        except websockets.ConnectionClosed:
            self.connected = False
            print("Connection closed")
    
    def start(self):
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        start_server = websockets.serve(self.handler, "127.0.0.1", 23223)
        loop.run_until_complete(start_server)
        loop.run_forever()
    
    async def send_message(self, message):
        if self.connected and self.websocket:
            await self.websocket.send(message)
        else:
            print("Not connected")

# 创建主窗口
class WebSocketApp:
    def __init__(self, root):
        self.root = root
        self.root.title("WebSocket Server")
        
        self.server = WebSocketServer()
        server_thread = threading.Thread(target=self.server.start)
        server_thread.start()
        
        # 创建控件
        self.text_display = scrolledtext.ScrolledText(root, wrap=tk.WORD, height=6, width=40)
        self.text_display.pack(pady=5)
        
        self.entry = tk.Entry(root, width=40)
        self.entry.bind("<Return>", self.on_enter_key)
        self.entry.pack(pady=5)
        
        self.button1 = tk.Button(root, text="发送 你好", command=self.send_hello)
        self.button1.pack(side=tk.LEFT, padx=5)
        
        self.button2 = tk.Button(root, text="检测连接", command=self.check_connection)
        self.button2.pack(side=tk.LEFT, padx=5)
        
        self.button3 = tk.Button(root, text="发送 JSON", command=self.send_json)
        self.button3.pack(side=tk.LEFT, padx=5)
    
    def on_enter_key(self, event):
        message = self.entry.get()
        asyncio.run(self.send_message(message))
        self.entry.delete(0, tk.END)
    
    def send_hello(self):
        asyncio.run(self.send_message("你好"))
    
    def check_connection(self):
        if self.server.connected:
            self.text_display.insert(tk.END, "连接正常\n")
        else:
            self.text_display.insert(tk.END, "未连接\n")
    
    def send_json(self):
        message = json.dumps({"msg": "hello"})
        asyncio.run(self.send_message(message))
    
    async def send_message(self, message):
        await self.server.send_message(message)
        self.text_display.insert(tk.END, f"发送: {message}\n")

# 运行应用
if __name__ == "__main__":
    root = tk.Tk()
    app = WebSocketApp(root)
    root.mainloop()
