package main

// panelHTML — панель управления, которая показывается в окне приложения на ПК
// (через WebView2). Здесь видны статус, QR-код, адрес, принятые файлы и настройки.
const panelHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>IpDrop</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body {
    margin: 0; font: 14px/1.45 -apple-system, system-ui, "Segoe UI", Roboto, sans-serif;
    color: #e7ecf3; background: radial-gradient(1200px 600px at 50% -10%, #16233c 0%, #0b0f17 60%);
    min-height: 100vh; padding: 22px 20px 30px;
  }
  .wrap { max-width: 760px; margin: 0 auto; }
  header { display: flex; align-items: center; gap: 12px; margin-bottom: 22px; }
  .logo { width: 40px; height: 40px; border-radius: 11px; background: linear-gradient(160deg,#5b97ff,#3a6fe0);
          display: flex; align-items: center; justify-content: center; box-shadow: 0 6px 18px rgba(76,141,255,.35); }
  .logo svg { width: 22px; height: 22px; }
  h1 { font-size: 20px; margin: 0; letter-spacing: .2px; }
  .pill { margin-left: auto; display: inline-flex; align-items: center; gap: 7px; font-size: 13px;
          color: #9fe3b4; background: rgba(52,199,89,.12); border: 1px solid rgba(52,199,89,.3);
          padding: 6px 12px; border-radius: 999px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: #34c759; box-shadow: 0 0 0 0 rgba(52,199,89,.6);
         animation: pulse 2s infinite; }
  @keyframes pulse { 0%{box-shadow:0 0 0 0 rgba(52,199,89,.5)} 70%{box-shadow:0 0 0 8px rgba(52,199,89,0)} 100%{box-shadow:0 0 0 0 rgba(52,199,89,0)} }

  .grid { display: grid; grid-template-columns: 240px 1fr; gap: 16px; }
  @media (max-width: 640px){ .grid { grid-template-columns: 1fr; } }
  .card { background: #121a28; border: 1px solid #1f2a3a; border-radius: 18px; padding: 18px; }
  .card h2 { font-size: 13px; text-transform: uppercase; letter-spacing: .6px; color: #8093a8; margin: 0 0 14px; font-weight: 600; }

  .qr { background: #fff; border-radius: 14px; padding: 12px; width: 100%; aspect-ratio: 1; display:flex; }
  .qr img { width: 100%; height: 100%; image-rendering: pixelated; }
  .addr { display: flex; align-items: center; gap: 8px; margin-top: 14px; background: #0d1422;
          border: 1px solid #1f2a3a; border-radius: 11px; padding: 9px 10px; }
  .addr code { font-size: 13px; color: #cfe0f5; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
  .copy { cursor: pointer; border: none; background: #213049; color: #cfe0f5; border-radius: 8px;
          padding: 7px 10px; font-size: 12px; white-space: nowrap; transition: .15s; }
  .copy:hover { background: #2b3e5d; }
  .copy.ok { background: #1f7a3d; color: #fff; }
  .qrhint { font-size: 12px; color: #7e8ca0; margin: 12px 2px 0; text-align: center; }

  .files { display: flex; flex-direction: column; gap: 9px; max-height: 420px; overflow: auto; }
  .file { display: flex; align-items: center; gap: 12px; background: #0e1623; border: 1px solid #1c2738;
          border-radius: 12px; padding: 9px 11px; animation: in .25s ease; }
  @keyframes in { from{opacity:0; transform: translateY(4px)} to{opacity:1; transform:none} }
  .thumb { width: 42px; height: 42px; border-radius: 9px; flex: 0 0 auto; object-fit: cover; background: #1b2738;
           display: flex; align-items: center; justify-content: center; font-size: 20px; }
  .file .meta { min-width: 0; flex: 1; }
  .file .nm { font-size: 13px; color: #e7ecf3; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .file .sub { font-size: 12px; color: #7e8ca0; margin-top: 2px; }
  .empty { color: #6b7a90; text-align: center; padding: 40px 10px; }
  .empty .big { font-size: 34px; opacity: .6; }

  .row { display: flex; align-items: center; gap: 12px; margin-top: 12px; }
  .row:first-of-type { margin-top: 0; }
  .row .label { color: #8093a8; font-size: 12px; }
  .row .val { margin-left: auto; color: #cfe0f5; font-size: 13px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; max-width: 60%; }
  .btn { cursor: pointer; border: 1px solid #2b3e5d; background: #18233a; color: #cfe0f5; border-radius: 9px;
         padding: 8px 12px; font-size: 13px; transition: .15s; }
  .btn:hover { background: #21304b; }

  /* переключатель */
  .switch { position: relative; width: 44px; height: 26px; margin-left: auto; }
  .switch input { display: none; }
  .slider { position: absolute; inset: 0; background: #2a3548; border-radius: 999px; transition: .2s; cursor: pointer; }
  .slider:before { content:""; position:absolute; width:20px; height:20px; left:3px; top:3px; background:#fff; border-radius:50%; transition:.2s; }
  .switch input:checked + .slider { background: #34c759; }
  .switch input:checked + .slider:before { transform: translateX(18px); }

  .foot { margin-top: 16px; }
</style>
</head>
<body>
<div class="wrap">
  <header>
    <div class="logo">
      <svg viewBox="0 0 24 24" fill="none"><path d="M12 3v12m0 0l-5-5m5 5l5-5" stroke="#fff" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"/><path d="M5 20h14" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/></svg>
    </div>
    <h1>IpDrop</h1>
    <span class="pill"><span class="dot"></span> В сети</span>
  </header>

  <div class="grid">
    <!-- Подключение -->
    <div class="card">
      <h2>Подключить телефон</h2>
      <div class="qr"><img src="/qr.png" alt="QR"></div>
      <div class="addr">
        <code id="addr">…</code>
        <button class="copy" id="copy">Копировать</button>
      </div>
      <p class="qrhint">Наведите «Камеру» iPhone на QR-код</p>
    </div>

    <!-- Принятые файлы -->
    <div class="card">
      <h2>Принятые файлы</h2>
      <div class="files" id="files">
        <div class="empty"><div class="big">📭</div>Пока ничего не принято</div>
      </div>
      <div class="foot">
        <div class="row"><span class="label">Папка</span><span class="val" id="dir">…</span><button class="btn" id="openFolder">Открыть</button></div>
        <div class="row"><span class="label">Автозапуск при входе в Windows</span>
          <label class="switch"><input type="checkbox" id="autostart"><span class="slider"></span></label>
        </div>
      </div>
    </div>
  </div>
</div>

<script>
  const $ = s => document.querySelector(s);

  async function loadInfo() {
    try {
      const i = await (await fetch('/api/info')).json();
      $('#addr').textContent = i.url;
      $('#dir').textContent = i.dir;
      $('#autostart').checked = !!i.autostart;
    } catch (e) {}
  }

  $('#copy').onclick = async () => {
    try { await navigator.clipboard.writeText($('#addr').textContent); } catch(e){}
    const b = $('#copy'); b.textContent = 'Скопировано'; b.classList.add('ok');
    setTimeout(() => { b.textContent = 'Копировать'; b.classList.remove('ok'); }, 1400);
  };

  $('#openFolder').onclick = () => fetch('/api/open-folder', { method: 'POST' });

  $('#autostart').onchange = async (e) => {
    const r = await fetch('/api/autostart', {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ enable: e.target.checked })
    });
    const j = await r.json(); e.target.checked = !!j.enabled;
  };

  function fmtSize(n) {
    if (n < 1024) return n + ' Б';
    const u = ['КБ','МБ','ГБ','ТБ']; let i = -1;
    do { n /= 1024; i++; } while (n >= 1024 && i < u.length-1);
    return n.toFixed(1) + ' ' + u[i];
  }
  function fmtTime(ms) {
    const d = new Date(ms), now = Date.now(), diff = (now - ms)/1000;
    if (diff < 60) return 'только что';
    if (diff < 3600) return Math.floor(diff/60) + ' мин назад';
    return d.toLocaleString('ru-RU', { hour:'2-digit', minute:'2-digit', day:'2-digit', month:'2-digit' });
  }
  function escapeHtml(s){return s.replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}

  let lastKey = '';
  async function loadFiles() {
    let files = [];
    try { files = await (await fetch('/api/files')).json(); } catch (e) { return; }
    const key = files.map(f => f.name + f.mtime).join('|');
    if (key === lastKey) return; // не перерисовываем без изменений
    lastKey = key;

    const box = $('#files');
    if (!files.length) {
      box.innerHTML = '<div class="empty"><div class="big">📭</div>Пока ничего не принято</div>';
      return;
    }
    box.innerHTML = files.map(f => {
      const thumb = f.image
        ? '<img class="thumb" src="/api/file?name=' + encodeURIComponent(f.name) + '" alt="">'
        : '<div class="thumb">📄</div>';
      return '<div class="file">' + thumb +
        '<div class="meta"><div class="nm">' + escapeHtml(f.name) + '</div>' +
        '<div class="sub">' + fmtSize(f.size) + ' · ' + fmtTime(f.mtime) + '</div></div></div>';
    }).join('');
  }

  loadInfo();
  loadFiles();
  setInterval(loadFiles, 1500);
</script>
</body>
</html>`
