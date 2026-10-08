import { useCallback, useEffect, useState } from "react"

import { API_BASE, fetchApiData, VoiceOption } from "./shared"

interface HealthResponse {
  status: string
  service: string
}

export default function SettingsPage() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [voices, setVoices] = useState<VoiceOption[]>([])
  const [healthError, setHealthError] = useState("")
  const [voiceError, setVoiceError] = useState("")
  const [loading, setLoading] = useState(true)

  const loadStatus = useCallback(async () => {
    setLoading(true)
    setHealthError("")
    setVoiceError("")
    const [healthResult, voiceResult] = await Promise.allSettled([
      fetch(`${API_BASE}/api/health`).then(async (response) => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        return (await response.json()) as HealthResponse
      }),
      fetchApiData<VoiceOption[]>("/api/tts/voices"),
    ])
    if (healthResult.status === "fulfilled") {
      setHealth(healthResult.value)
    } else {
      setHealth(null)
      setHealthError(
        healthResult.reason instanceof Error
          ? healthResult.reason.message
          : "Không thể kết nối backend.",
      )
    }
    if (voiceResult.status === "fulfilled") {
      setVoices(voiceResult.value)
    } else {
      setVoices([])
      setVoiceError(
        voiceResult.reason instanceof Error
          ? voiceResult.reason.message
          : "Không thể tải danh sách giọng đọc.",
      )
    }
    setLoading(false)
  }, [])

  useEffect(() => {
    void loadStatus()
  }, [loadStatus])

  return (
    <div className="space-y-8 text-white">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#7B2FFF]">
            ⚙️ Tình trạng dịch vụ
          </div>
          <h1 className="font-['Unbounded'] text-3xl font-black">
            Thiết lập <span className="gradient-text">hệ thống</span>
          </h1>
          <p className="mt-2 text-sm text-white/50">
            Kiểm tra backend và các giọng Piper hiện được hệ thống nhận diện.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void loadStatus()}
          disabled={loading}
          className="rounded-full border-2 border-[#00F5D4] px-5 py-2.5 text-xs font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A] disabled:opacity-50"
        >
          {loading ? "Đang kiểm tra…" : "↻ Kiểm tra lại"}
        </button>
      </header>

      <section className="grid gap-5 md:grid-cols-2">
        <article className="rounded-3xl border-4 border-[#00F5D4] bg-[#2D1B4E] p-6">
          <div className="flex items-center justify-between gap-3">
            <h2 className="font-['Unbounded'] text-sm font-black uppercase text-[#00F5D4]">
              Backend Go
            </h2>
            <span className={`rounded-full border px-3 py-1 text-xs font-black ${health ? "border-[#00F5D4] text-[#00F5D4]" : "border-[#FF6B35] text-[#FF6B35]"}`}>
              {loading ? "ĐANG KIỂM TRA" : health ? "ĐANG KẾT NỐI" : "MẤT KẾT NỐI"}
            </span>
          </div>
          <p className="mt-4 text-sm text-white/70">
            {health?.service ?? "AI ADS Studio Backend"}
          </p>
          <p className="mt-2 break-all text-xs text-white/40">
            Endpoint: {API_BASE}/api/health
          </p>
          {healthError && <p className="mt-3 text-xs text-[#FFB08E]">{healthError}</p>}
        </article>

        <article className="rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-6">
          <h2 className="font-['Unbounded'] text-sm font-black uppercase text-[#FF3AF2]">
            Tạo lời thoại và giọng đọc
          </h2>
          <ul className="mt-4 space-y-3 text-sm leading-relaxed text-white/70">
            <li><strong className="text-white">Piper TTS:</strong> tạo giọng tiếng Việt cục bộ, không gọi API TTS trả phí.</li>
            <li><strong className="text-white">Groq:</strong> dùng để sinh kịch bản quảng cáo và cần cấu hình GROQ_API_KEY ở backend.</li>
            <li><strong className="text-white">FFmpeg:</strong> cần có để dựng video và xử lý thời lượng âm thanh.</li>
          </ul>
          <p className="mt-4 border-t border-dashed border-white/15 pt-3 text-xs leading-relaxed text-white/40">
            Cấu hình nằm ở backend/.env; khóa API không được nhập hoặc hiển thị
            trong giao diện này.
          </p>
        </article>
      </section>

      <section className="rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E] p-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-['Unbounded'] text-sm font-black uppercase text-[#FFE600]">
            Giọng Piper khả dụng
          </h2>
          <span className="text-xs font-bold text-white/45">
            {loading ? "Đang tải…" : `${voices.length} giọng`}
          </span>
        </div>
        {voiceError && (
          <p className="mt-4 rounded-xl border border-[#FF6B35]/60 bg-[#FF6B35]/10 p-3 text-sm text-[#FFB08E]">
            Không tải được giọng đọc: {voiceError}
          </p>
        )}
        {!loading && !voiceError && voices.length === 0 && (
          <p className="mt-4 text-sm text-white/50">Backend chưa phát hiện giọng đọc nào.</p>
        )}
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          {voices.map((voice) => (
            <div key={voice.id} className="rounded-2xl border border-[#FFE600]/30 bg-[#0D0D1A]/40 p-4">
              <h3 className="text-sm font-black text-white">{voice.name}</h3>
              <p className="mt-1 text-xs leading-relaxed text-white/50">{voice.description}</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
