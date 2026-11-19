// Thin wrapper over the YouTube IFrame API. The video iframe is kept off-screen;
// the turntable is the only visible "player".
declare global {
  interface Window { YT: any; onYouTubeIframeAPIReady?: () => void }
}

let apiPromise: Promise<any> | null = null
function loadApi(): Promise<any> {
  if (window.YT?.Player) return Promise.resolve(window.YT)
  if (apiPromise) return apiPromise
  apiPromise = new Promise((resolve) => {
    const prev = window.onYouTubeIframeAPIReady
    window.onYouTubeIframeAPIReady = () => { prev?.(); resolve(window.YT) }
    const s = document.createElement('script')
    s.src = 'https://www.youtube.com/iframe_api'
    document.head.appendChild(s)
  })
  return apiPromise
}

export function useYouTubePlayer(elementId: string, opts: { onEnded?: () => void } = {}) {
  const ready = ref(false)
  const playing = ref(false)
  const buffering = ref(false)
  const error = ref<string | null>(null)
  const currentTime = ref(0)
  const duration = ref(0)
  const isLive = ref(false)
  let player: any = null
  let cuedId: string | null = null
  let timer: ReturnType<typeof setInterval> | undefined

  async function init(volume: number) {
    const YT = await loadApi()
    await new Promise<void>((resolve) => {
      player = new YT.Player(elementId, {
        width: 320, height: 180,
        playerVars: { controls: 0, disablekb: 1, playsinline: 1, rel: 0 },
        events: {
          onReady: () => { ready.value = true; player.setVolume(volume); resolve() },
          onStateChange: (e: any) => {
            const S = YT.PlayerState
            playing.value = e.data === S.PLAYING || e.data === S.BUFFERING
            buffering.value = e.data === S.BUFFERING
            if (e.data === S.PLAYING) {
              error.value = null
              isLive.value = !!player.getVideoData?.().isLive
              duration.value = isLive.value ? 0 : player.getDuration() || 0
            }
            if (e.data === S.ENDED) {
              if (opts.onEnded) opts.onEnded()
              else { player.seekTo(0); player.playVideo() } // loop
            }
          },
          onError: (e: any) => {
            playing.value = false
            error.value = [101, 150].includes(e.data) ? 'The owner does not allow this video to be played outside YouTube.' : 'This record cannot be played.'
          },
        },
      })
    })
    timer = setInterval(() => { if (player?.getCurrentTime) currentTime.value = player.getCurrentTime() }, 1000)
  }

  function load(videoId: string, autoplay: boolean) {
    if (!player) return
    error.value = null
    duration.value = 0
    currentTime.value = 0
    isLive.value = false
    cuedId = videoId
    autoplay ? player.loadVideoById(videoId) : player.cueVideoById(videoId)
  }
  function play() { if (player && cuedId) player.playVideo() }
  function pause() { player?.pauseVideo() }
  function setVolume(v: number) { player?.setVolume(v) }
  function mute() { player?.mute?.() }
  function unMute() { player?.unMute?.() }
  function seekTo(seconds: number) {
    if (!player) return
    player.seekTo(seconds, true)
    currentTime.value = seconds
  }
  function getDuration() { return isLive.value ? 0 : player?.getDuration?.() || 0 }

  onBeforeUnmount(() => { clearInterval(timer); player?.destroy?.() })

  return { ready, playing, buffering, error, currentTime, duration, isLive, init, load, play, pause, setVolume, mute, unMute, seekTo, getDuration }
}
