import type { StreamConfig } from '../types/config'

type CustomBannerProps = {
  config: StreamConfig
}

export function CustomBanner({ config }: CustomBannerProps) {
  return (
    <header className="space-y-3 rounded-xl border border-zinc-700 bg-zinc-900 p-4">
      {config.bannerUrl ? (
        <img src={config.bannerUrl} alt="Stream banner" className="h-44 w-full rounded-md object-cover" />
      ) : null}
      <div>
        <h1 className="text-3xl font-bold text-white">{config.title || 'SovereignStream Live'}</h1>
        <p className="text-sm text-zinc-300">Independent livestream with P2P distribution</p>
      </div>
    </header>
  )
}
