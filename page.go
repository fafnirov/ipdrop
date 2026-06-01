package main

// pageHTML — встроенная веб-страница приёма. Один файл, без внешних зависимостей,
// чтобы работала даже без интернета (только локальная сеть).
const pageHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<meta name="apple-mobile-web-app-title" content="IpDrop">
<meta name="theme-color" content="#0b0f17">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
<link rel="icon" type="image/png" href="/icon-192.png">
<link rel="manifest" href="/manifest.webmanifest">
<title>IpDrop — отправить на ПК</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
  body {
    margin: 0; min-height: 100vh; display: flex; align-items: center; justify-content: center;
    font: 16px/1.4 -apple-system, system-ui, "Segoe UI", Roboto, sans-serif;
    background: #0b0f17; color: #e7ecf3; padding: 24px;
  }
  .card {
    width: 100%; max-width: 440px; background: #131a26; border: 1px solid #1f2a3a;
    border-radius: 20px; padding: 28px 24px; box-shadow: 0 20px 50px rgba(0,0,0,.4);
  }
  h1 { margin: 0 0 4px; font-size: 22px; }
  .sub { margin: 0 0 22px; color: #93a1b5; font-size: 14px; }
  .drop {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    gap: 8px; text-align: center; padding: 38px 18px; border: 2px dashed #2c3a4f;
    border-radius: 16px; cursor: pointer; transition: .15s; background: #0f1623;
  }
  .drop.drag { border-color: #4c8dff; background: #15203a; }
  .drop .big { font-size: 17px; font-weight: 600; }
  .drop .hint { font-size: 13px; color: #7e8ca0; }
  .icon { font-size: 40px; }
  #list { margin-top: 16px; display: flex; flex-direction: column; gap: 10px; }
  .item { background: #0f1623; border: 1px solid #1f2a3a; border-radius: 12px; padding: 10px 12px; }
  .item .name { font-size: 14px; word-break: break-all; }
  .bar { height: 6px; border-radius: 6px; background: #1f2a3a; margin-top: 8px; overflow: hidden; }
  .bar > i { display: block; height: 100%; width: 0; background: #4c8dff; transition: width .15s; }
  .item.done .bar > i { background: #34c759; }
  .item .st { font-size: 12px; color: #93a1b5; margin-top: 6px; }
  .item.done .st { color: #34c759; }
  .item.err .st { color: #ff5b5b; }
  .tip { margin: 22px 0 0; font-size: 13px; color: #7e8ca0; text-align: center; }
</style>
</head>
<body>
  <div class="card">
    <h1>📥 IpDrop</h1>
    <p class="sub">Файлы сохранятся на компьютере. Выберите фото или документы ниже.</p>

    <label class="drop" id="drop">
      <input type="file" id="file" multiple hidden>
      <div class="icon">⬆️</div>
      <div class="big">Выбрать фото или файлы</div>
      <div class="hint">или перетащите сюда</div>
    </label>

    <div id="list"></div>

    <p class="tip">💡 Чтобы не открывать адрес каждый раз: нажмите
      «Поделиться» → «На экран Домой» — появится значок IpDrop.</p>
  </div>

<script>
  const input = document.getElementById('file');
  const drop  = document.getElementById('drop');
  const list  = document.getElementById('list');

  input.addEventListener('change', () => { if (input.files.length) send(input.files); });

  // Перетаскивание (для отправки с ноутбука/ПК).
  ['dragenter','dragover'].forEach(e => drop.addEventListener(e, ev => {
    ev.preventDefault(); drop.classList.add('drag');
  }));
  ['dragleave','drop'].forEach(e => drop.addEventListener(e, ev => {
    ev.preventDefault(); drop.classList.remove('drag');
  }));
  drop.addEventListener('drop', ev => { if (ev.dataTransfer.files.length) send(ev.dataTransfer.files); });

  function send(files) {
    const fd = new FormData();
    let names = [];
    for (const f of files) { fd.append('files', f, f.name); names.push(f.name); }

    const row = document.createElement('div');
    row.className = 'item';
    row.innerHTML =
      '<div class="name">' + names.map(escapeHtml).join(', ') + '</div>' +
      '<div class="bar"><i></i></div>' +
      '<div class="st">Отправка…</div>';
    list.prepend(row);
    const bar = row.querySelector('.bar > i');
    const st  = row.querySelector('.st');

    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/upload');
    xhr.upload.onprogress = e => {
      if (e.lengthComputable) {
        const p = Math.round(e.loaded / e.total * 100);
        bar.style.width = p + '%';
        st.textContent = 'Отправка… ' + p + '%';
      }
    };
    xhr.onload = () => {
      if (xhr.status === 200) {
        row.classList.add('done'); bar.style.width = '100%';
        st.textContent = '✓ Отправлено';
      } else {
        row.classList.add('err'); st.textContent = 'Ошибка: ' + xhr.statusText;
      }
    };
    xhr.onerror = () => { row.classList.add('err'); st.textContent = 'Ошибка соединения'; };
    xhr.send(fd);
    input.value = '';
  }

  function escapeHtml(s) {
    return s.replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  }
</script>
</body>
</html>`
