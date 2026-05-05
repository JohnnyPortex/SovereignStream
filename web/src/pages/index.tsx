import { useEffect, useMemo, useState } from 'react'
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

  useEffect(() => {
    const fetchData = async () => {
      const [cfgRes, statusRes, urlRes] = await Promise.all([
        fetch('/api/config'),
        fetch('/api/live-status'),
        fetch('/api/stream-url'),
      ])

      if (cfgRes.ok) {
        setConfig(await cfgRes.json())
      }
      if (statusRes.ok) {
        setStatus(await statusRes.json())
      }
      if (urlRes.ok) {
        const data = (await urlRes.json()) as { url: string }
        setStreamBaseUrl(data.url)
      }
    }

    void fetchData()
    const interval = window.setInterval(fetchData, 4000)

    return () => {
      window.clearInterval(interval)
    }
  }, [])

  const accentStyle = useMemo(
    () => ({
      '--accent': config.accentColor || '#7c3aed',
    }) as React.CSSProperties,
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

          {streamBaseUrl ? (
            <VideoPlayer streamBaseUrl={streamBaseUrl} />
          ) : (
            <div className="rounded-md border border-dashed border-zinc-700 p-6 text-center text-sm text-zinc-400">
              Aguardando URL pública do túnel...
            </div>
          )}
        </section>

        <P2PManager roomId="default" enabled={status.online} />
        <Chat roomId="default" />
      </div>
    </main>
  )
}
