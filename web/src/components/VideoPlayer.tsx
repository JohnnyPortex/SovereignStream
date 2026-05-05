import { useEffect, useRef } from 'react'
import Hls from 'hls.js'

type VideoPlayerProps = Readonly<{
  streamBaseUrl: string
}>

export function VideoPlayer({ streamBaseUrl }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video || !streamBaseUrl) {
      return
    }

    const source = `${streamBaseUrl.replace(/\/$/, '')}/hls/index.m3u8`

    const tryPlay = () => {
      void video.play().catch(() => undefined)
    }

    if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = source
      video.load()
      tryPlay()
      return
    }

    const hls = new Hls({
      lowLatencyMode: true,
      liveSyncDurationCount: 2,
      liveMaxLatencyDurationCount: 4,
      maxLiveSyncPlaybackRate: 1.5,
      maxBufferLength: 4,
      backBufferLength: 8,
      manifestLoadingTimeOut: 10000,
      levelLoadingTimeOut: 10000,
      fragLoadingTimeOut: 15000,
    })

    hls.on(Hls.Events.MANIFEST_PARSED, () => {
      tryPlay()
    })

    hls.on(Hls.Events.LEVEL_LOADED, (_event, data) => {
      if (data.details.live && hls.liveSyncPosition && video.currentTime > 0) {
        const drift = hls.latency ?? 0
        if (drift > 6) {
          video.currentTime = hls.liveSyncPosition
        }
      }
      if (video.paused) {
        tryPlay()
      }
    })

    hls.on(Hls.Events.ERROR, (_event, data) => {
      if (!data.fatal) {
        return
      }

      if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
        hls.startLoad()
        return
      }

      if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
        hls.recoverMediaError()
        return
      }

      hls.destroy()
    })

    hls.loadSource(source)
    hls.attachMedia(video)

    return () => {
      hls.destroy()
    }
  }, [streamBaseUrl])

  return (
    <div className="overflow-hidden rounded-lg border border-zinc-700 bg-black">
      <video ref={videoRef} controls autoPlay playsInline muted className="h-auto w-full" />
    </div>
  )
}
