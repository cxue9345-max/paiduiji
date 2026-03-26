const statusEl = document.getElementById('status');
const listEl = document.getElementById('list');
const roomidEl = document.getElementById('roomid');
const uidEl = document.getElementById('uid');
const cookieEl = document.getElementById('cookie');
const saveBtn = document.getElementById('save');

function setStatus(text){ statusEl.textContent = `状态：${text}`; }

function addLine(text, meta=''){
  const row = document.createElement('div');
  row.className = 'item';
  row.innerHTML = `<div>${text}</div><div class="meta">${meta}</div>`;
  listEl.prepend(row);
  if (listEl.children.length > 120) listEl.removeChild(listEl.lastChild);
}

async function loadConfig(){
  const res = await fetch('http://127.0.0.1:23334/api/config');
  const cfg = await res.json();
  roomidEl.value = cfg.roomid || '';
  uidEl.value = cfg.uid || '';
  cookieEl.value = cfg.cookie || '';
}

saveBtn.onclick = async () => {
  const payload = {
    roomid: Number(roomidEl.value || 0),
    uid: Number(uidEl.value || 0),
    cookie: cookieEl.value || ''
  };
  await fetch('http://127.0.0.1:23334/api/config', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify(payload)
  });
  setStatus('配置已保存，等待重连');
};

function connect(){
  const ws = new WebSocket('ws://127.0.0.1:23333/danmu/sub');
  ws.onopen = () => setStatus('已连接本地弹幕流');
  ws.onclose = () => {
    setStatus('连接断开，2秒后重连');
    setTimeout(connect, 2000);
  };
  ws.onerror = () => setStatus('连接异常');
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data);
      if (data.type === 'PDJ_STATUS') {
        addLine(`[状态] ${data.status}`, JSON.stringify(data));
        return;
      }
      const cmd = data.cmd || '';
      if (cmd.includes('DANMU_MSG')) {
        const info = data.info || [];
        const uname = (info[2] && info[2][1]) || '未知用户';
        const text = info[1] || '';
        addLine(`${uname}: ${text}`, cmd);
        return;
      }
      addLine(`[事件] ${cmd || 'unknown'}`, JSON.stringify(data).slice(0, 180));
    } catch {
      addLine('[原始消息]', ev.data.slice(0, 200));
    }
  };
}

loadConfig().finally(connect);
