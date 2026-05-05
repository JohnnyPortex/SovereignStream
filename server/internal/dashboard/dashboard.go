package dashboard

import "net/http"

func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(pageHTML))
	})
	return mux
}

const pageHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>SovereignStream Local Control</title>
    <style>
      body { font-family: Arial, sans-serif; max-width: 920px; margin: 24px auto; padding: 0 16px; }
      .card { border: 1px solid #ddd; border-radius: 8px; padding: 16px; margin-bottom: 12px; }
      input, textarea { width: 100%; padding: 8px; margin: 6px 0 12px; box-sizing: border-box; }
      button { padding: 10px 14px; cursor: pointer; }
      .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
      .status { font-weight: 700; }
    </style>
  </head>
  <body>
    <h1>SovereignStream Local Panel</h1>
    <div class="card">
      <div class="status" id="status">Checking stream...</div>
      <div>Public URL: <a id="stream-url" href="#" target="_blank">-</a></div>
    </div>

    <div class="grid">
      <section class="card">
        <h2>Branding</h2>
        <label>Title</label>
        <input id="title" />
        <label>Accent Color</label>
        <input id="accentColor" placeholder="#7c3aed" />
        <label>Banner URL</label>
        <input id="bannerUrl" />
        <button id="save-config">Save Config</button>
      </section>

      <section class="card">
        <h2>Chat</h2>
        <label>Room ID</label>
        <input id="roomId" value="default" />
        <label>Message</label>
        <textarea id="chatMessage" rows="4"></textarea>
        <button id="send-chat">Send Message</button>
      </section>
    </div>

    <script>
      const $ = (id) => document.getElementById(id);

      async function refreshStatus() {
        const statusRes = await fetch('/api/live-status');
        const status = await statusRes.json();
        $('status').textContent = status.online ? 'LIVE' : 'OFFLINE';

        const tunnelRes = await fetch('/api/stream-url');
        const tunnel = await tunnelRes.json();
        const url = tunnel.url || '-';
        $('stream-url').textContent = url;
        $('stream-url').href = url;
      }

      async function loadConfig() {
        const res = await fetch('/api/config');
        const cfg = await res.json();
        $('title').value = cfg.title || '';
        $('accentColor').value = cfg.accentColor || '';
        $('bannerUrl').value = cfg.bannerUrl || '';
      }

      $('save-config').addEventListener('click', async () => {
        const payload = {
          title: $('title').value,
          accentColor: $('accentColor').value,
          bannerUrl: $('bannerUrl').value,
        };
        await fetch('/api/config', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        await loadConfig();
      });

      $('send-chat').addEventListener('click', async () => {
        const payload = {
          roomId: $('roomId').value,
          sender: 'streamer',
          message: $('chatMessage').value,
        };
        await fetch('/api/chat/send', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        });
        $('chatMessage').value = '';
      });

      loadConfig();
      refreshStatus();
      setInterval(refreshStatus, 3000);
    </script>
  </body>
</html>`
