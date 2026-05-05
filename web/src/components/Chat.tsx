import { useEffect, useState } from 'react'
import Pusher from 'pusher-js'

type Message = {
  sender: string
  message: string
}

type ChatProps = {
  roomId: string
  apiBaseUrl: string
}

export function Chat({ roomId, apiBaseUrl }: ChatProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [newMessage, setNewMessage] = useState('')

  useEffect(() => {
    const key = import.meta.env.VITE_PUSHER_KEY
    const cluster = import.meta.env.VITE_PUSHER_CLUSTER

    if (!key || !cluster) {
      return
    }

    const pusher = new Pusher(key, { cluster })
    const channel = pusher.subscribe(`chat-${roomId}`)

    const onMessage = (payload: Message) => {
      setMessages((prev) => [...prev.slice(-49), payload])
    }

    channel.bind('new-message', onMessage)

    return () => {
      channel.unbind('new-message', onMessage)
      pusher.unsubscribe(`chat-${roomId}`)
      pusher.disconnect()
    }
  }, [roomId])

  async function sendMessage() {
    const message = newMessage.trim()
    if (!message) {
      return
    }

    if (!apiBaseUrl) {
      return
    }
    await fetch(`${apiBaseUrl}/api/chat/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ roomId, sender: 'viewer', message }),
    })

    setNewMessage('')
  }

  return (
    <section className="rounded-xl border border-zinc-700 bg-zinc-900 p-4">
      <h2 className="mb-3 text-lg font-semibold text-white">Chat</h2>
      <div className="mb-3 h-52 space-y-2 overflow-y-auto rounded-md bg-zinc-950 p-3">
        {messages.length === 0 ? (
          <p className="text-sm text-zinc-400">No messages yet.</p>
        ) : (
          messages.map((msg, idx) => (
            <p key={`${msg.sender}-${idx}`} className="text-sm text-zinc-200">
              <span className="font-semibold text-violet-300">{msg.sender}: </span>
              {msg.message}
            </p>
          ))
        )}
      </div>

      <div className="flex gap-2">
        <input
          value={newMessage}
          onChange={(e) => setNewMessage(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              void sendMessage()
            }
          }}
          className="flex-1 rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm text-zinc-100"
          placeholder="Type your message"
        />
        <button
          onClick={() => void sendMessage()}
          className="rounded-md bg-violet-600 px-3 py-2 text-sm font-medium text-white"
        >
          Send
        </button>
      </div>
    </section>
  )
}
