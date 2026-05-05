import { useEffect, useMemo, useState, type CSSProperties } from 'react'
import Pusher from 'pusher-js'
import { Chat } from '../components/Chat'
import { CustomBanner } from '../components/CustomBanner'
import { P2PManager } from '../components/P2PManager'
import { VideoPlayer } from '../components/VideoPlayer'
import type { LiveStatus, StreamConfig } from '../types/config'

const defaultConfig: StreamConfig = {
  title: 'SovereignStream Live',
  accentColor: '#7c3aed',
  bannerUrl: '',
}

export function HomePage() {
  const [config, setConfig] = useState<StreamConfig>(defaultConfig)
  const [status, setStatus] = useState<LiveStatus>({ online: false, streamKey: '' })
  const [streamBaseUrl, setStreamBaseUrl] = useState('')
  const envApiBase = import.meta.env.VITE_BACKEND_BASE_URL || ''
  const pusherKey = import.meta.env.VITE_PUSHER_KEY
  const pusherCluster = import.meta.env.VITE_PUSHER_CLUSTER
  const effectiveBaseUrl = envApiBase || streamBaseUrl

  useEffect(() => {
    const apiBase = envApiBase || streamBaseUrl
    if (!apiBase) {
      return
    }

    const fetchData = async () => {
      const [cfgRes, statusRes, urlRes] = await Promise.all([
        fetch(`${apiBase}/api/config`),
        fetch(`${apiBase}/api/live-status`),
        fetch(`${apiBase}/api/stream-url`),
      ])

      if (cfgRes.ok) {
        setConfig(await cfgRes.json())
      }
      if (statusRes.ok) {
        setStatus(await statusRes.json())
      }
      if (urlRes.ok) {
        const data = (await urlRes.json()) as { url: string }
        if (!envApiBase) {
          setStreamBaseUrl((prev) => data.url || prev)
        }
      }
    }

    void fetchData()
    const interval = globalThis.setInterval(fetchData, 4000)

    return () => {
      globalThis.clearInterval(interval)
    }
  }, [envApiBase, streamBaseUrl])

  useEffect(() => {
    if (!pusherKey || !pusherCluster) {
      return
    }
    const pusher = new Pusher(pusherKey, { cluster: pusherCluster })
    const channel = pusher.subscribe('live-status')
    const onLiveStatus = (payload: { online: boolean; streamUrl?: string }) => {
      setStatus((prev) => ({ ...prev, online: payload.online }))
      if (payload.streamUrl && !envApiBase) {
        setStreamBaseUrl(payload.streamUrl)
      }
    }
    channel.bind('state-updated', onLiveStatus)
    return () => {
      channel.unbind('state-updated', onLiveStatus)
      pusher.unsubscribe('live-status')
      pusher.disconnect()
    }
  }, [pusherCluster, pusherKey])

  const accentStyle = useMemo(
    () =>
      ({
        '--accent': config.accentColor || '#7c3aed',
      }) as CSSProperties,
    [config.accentColor],
  )

  return (
    <main style={accentStyle} className="min-h-screen bg-zinc-950 p-4 text-zinc-100 md:p-8">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
        <CustomBanner config={config} />

        <section className="rounded-xl border border-zinc-700 bg-zinc-900 p-4">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-xl font-semibold text-white">Stream</h2>
            <span
              className={`rounded-full px-3 py-1 text-xs font-semibold ${
                status.online ? 'bg-emerald-600 text-white' : 'bg-zinc-700 text-zinc-200'
              }`}
            >
              {status.online ? 'ONLINE' : 'OFFLINE'}
            </span>
          </div>

          {effectiveBaseUrl ? (
            <VideoPlayer streamBaseUrl={effectiveBaseUrl} />
          ) : (
            <div className="rounded-md border border-dashed border-zinc-700 p-6 text-center text-sm text-zinc-400">
              Waiting for public tunnel URL...
            </div>
          )}
        </section>

        <P2PManager roomId="default" enabled={status.online} apiBaseUrl={effectiveBaseUrl} />
        <Chat roomId="default" apiBaseUrl={effectiveBaseUrl} />
      </div>
    </main>
  )
}
