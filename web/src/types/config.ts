export type StreamConfig = {
  title: string
  accentColor: string
  bannerUrl: string
  rtmpStreamKey?: string
  pusherAppId?: string
  pusherKey?: string
  pusherSecret?: string
  pusherCluster?: string
}

export type LiveStatus = {
  online: boolean
  streamKey: string
}
