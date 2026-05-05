import { useEffect, useRef } from 'react'
import Pusher from 'pusher-js'
import Peer, { type SignalData } from 'simple-peer'

type P2PManagerProps = {
  roomId: string
  enabled: boolean
  apiBaseUrl: string
}

type SignalPayload = {
  from: string
  to?: string
  signal: SignalData
}

function randomPeerId() {
  return Math.random().toString(36).slice(2)
}

export function P2PManager({ roomId, enabled, apiBaseUrl }: P2PManagerProps) {
  const peerIdRef = useRef<string>(randomPeerId())
  const peerRef = useRef<Peer.Instance | null>(null)

  useEffect(() => {
    if (!enabled) {
      return
    }

    const key = import.meta.env.VITE_PUSHER_KEY
    const cluster = import.meta.env.VITE_PUSHER_CLUSTER

    if (!key || !cluster) {
      return
    }

    const pusher = new Pusher(key, { cluster })
    const channel = pusher.subscribe(`webrtc-${roomId}`)

    const isInitiator = location.hash.includes('seed=true')
    const peer = new Peer({ initiator: isInitiator, trickle: true })
    peerRef.current = peer

    peer.on('signal', async (signal: SignalData) => {
      const payload: SignalPayload = {
        from: peerIdRef.current,
        signal,
      }

      if (!apiBaseUrl) {
        return
      }
      await fetch(`${apiBaseUrl}/api/webrtc/signal`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ roomId, data: payload }),
      })
    })

    const onSignalEvent = (payload: SignalPayload) => {
      if (payload.from === peerIdRef.current || payload.to === peerIdRef.current) {
        return
      }
      peer.signal(payload.signal)
    }

    channel.bind('signal', onSignalEvent)

    return () => {
      channel.unbind('signal', onSignalEvent)
      pusher.unsubscribe(`webrtc-${roomId}`)
      pusher.disconnect()
      peer.destroy()
      peerRef.current = null
    }
  }, [apiBaseUrl, enabled, roomId])

  return null
}
