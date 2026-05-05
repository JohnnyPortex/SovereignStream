# SovereignStream

SovereignStream is a zero-infrastructure livestream MVP where the streamer runs the media pipeline locally and viewers watch through a static cloud frontend.

## MVP Architecture

- **Local App (`server/`)**: Go service that receives RTMP from OBS, transcodes to HLS with FFmpeg, starts Cloudflare Tunnel, stores configuration in SQLite, and exposes a local control panel.
- **Cloud Frontend (`web/`)**: React + Tailwind app hosted on Vercel with HLS playback, P2P helper logic via Simple-Peer, and chat via Pusher channels.
- **Signaling**: Pusher free tier for live status updates, WebRTC signaling, and chat events.

Flow:

1. OBS publishes RTMP to `rtmp://localhost:1935/live/<streamKey>`.
2. Go server starts FFmpeg and produces `hls/index.m3u8`.
3. Cloudflare Tunnel exposes local HTTP API/HLS endpoints over HTTPS.
4. Frontend reads live URL/config and plays stream with `hls.js`.
5. Viewers exchange WebRTC signaling and chat through Pusher channels.

## Repository Layout

- `server/main.go`: server entrypoint and HTTP API routes
- `server/internal/rtmp`: RTMP listener (joy4)
- `server/internal/transcoder`: FFmpeg process manager and HLS file serving
- `server/internal/tunnel`: Cloudflare Tunnel process manager
- `server/internal/config`: SQLite-backed stream configuration store
- `server/internal/signaling`: Pusher publisher for signaling/chat/status
- `server/internal/dashboard`: local web panel (`localhost:3000`)
- `server/Dockerfile`: container image for local app
- `server/.goreleaser.yaml`: Linux/Windows release build config
- `web/`: Vite React frontend for Vercel
- `docker-compose.yml`: optional local container workflow

## Prerequisites

- FFmpeg installed on the machine that runs the local app
- `cloudflared` installed and available in `$PATH`
- OBS Studio
- Pusher account (free tier)
- Node.js 20+
- Go 1.22+

## Local Development

### 1) Start backend

```bash
cd server
cp .env.example .env
# Optional: export values from .env into your shell
go mod tidy
go run ./main.go
```

Default endpoints:

- RTMP ingest: `:1935`
- HTTP API + HLS: `:8080`
- Local dashboard: `:3000`

### 2) Publish from OBS

Set OBS custom server:

- Server: `rtmp://localhost:1935/live`
- Stream key: any key (for example, `main`)

### 3) Start frontend

```bash
cd web
cp .env.example .env
npm install
npm run dev
```

## Environment Variables

### Backend (`server/.env.example`)

- `SOVEREIGNSTREAM_HTTP_ADDR` (default `:8080`)
- `SOVEREIGNSTREAM_RTMP_ADDR` (default `:1935`)
- `SOVEREIGNSTREAM_DASHBOARD_ADDR` (default `:3000`)
- `SOVEREIGNSTREAM_DB_PATH` (default `./sovereignstream.db`)
- `SOVEREIGNSTREAM_HLS_DIR` (default `./hls`)

### Frontend (`web/.env.example`)

- `VITE_PUSHER_KEY`
- `VITE_PUSHER_CLUSTER`
- `VITE_BACKEND_BASE_URL` (optional fixed backend URL; tunnel URL can still arrive through Pusher `live-status`)

## Dashboard Features (`localhost:3000`)

- Live status (ONLINE/OFFLINE)
- Current public tunnel URL
- Branding config editor (`title`, `accentColor`, `bannerUrl`)
- Chat sender for room `default`

## API Surface (Go app)

- `GET /api/config`
- `PUT /api/config`
- `GET /api/live-status`
- `GET /api/stream-url`
- `POST /api/webrtc/signal`
- `POST /api/chat/send`
- `GET /healthz`
- `GET /hls/index.m3u8` and segment files

## Build and Packaging

### Docker Compose

```bash
docker compose up --build
```

### GoReleaser

```bash
cd server
goreleaser release --snapshot --clean
```

Generates Linux and Windows artifacts.

## Deploy Frontend to Vercel

Use this one-click deploy button (replace `YOUR_GITHUB_REPO_URL`):

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=YOUR_GITHUB_REPO_URL&root-directory=web)

After deploy, configure frontend environment variables in Vercel project settings.

## Cost Profile (MVP)

- Vercel Hobby: free static hosting
- Cloudflare Tunnel: free quick tunnel
- Pusher free tier: signaling/chat for small audiences
- Video transcoding and egress processing run on streamer hardware

## Stress Test Checklist

- Run one OBS publisher + 5 to 10 browser viewers
- Track upload usage on streamer machine
- Validate end-to-end latency target: 5 to 10 seconds for MVP
- Confirm tunnel stability for at least 30 minutes

## Notes

- Current backend assumes local FFmpeg and cloudflared binaries are installed.
- For production hardening, add auth for API routes and richer tunnel lifecycle handling.
