export const ACCENTS = [
  "#FF3AF2",
  "#00F5D4",
  "#FFE600",
  "#FF6B35",
  "#7B2FFF",
] as const

export const BORDER_COLORS = [
  "border-[#FF3AF2]",
  "border-[#00F5D4]",
  "border-[#FFE600]",
  "border-[#FF6B35]",
  "border-[#7B2FFF]",
] as const

export const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080"

export interface OutputVideo {
  id: string
  taskId: string
  title: string
  filename: string
  url: string
  duration: number
  size: number
  createdAt: string
}

export interface RemixTask {
  id: string
  status: "pending" | "processing" | "completed" | "failed" | string
  progress: number
  message: string
  request: {
    outputCount: number
    duration: number
    replaceVoice: boolean
    voice: string
    remixMode: string
  }
  outputVideos: OutputVideo[]
  error?: string
  createdAt: string
  updatedAt: string
}

export interface VoiceOption {
  id: string
  name: string
  description: string
}

export async function fetchApiData<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`)
  if (!response.ok) {
    throw new Error(`Yêu cầu thất bại (HTTP ${response.status})`)
  }
  const payload: { data?: T } = await response.json()
  if (payload.data === undefined) {
    throw new Error("Phản hồi từ máy chủ không có dữ liệu")
  }
  return payload.data
}

export function formatFileSize(size: number) {
  return size < 1024 * 1024
    ? `${(size / 1024).toFixed(0)} KB`
    : `${(size / (1024 * 1024)).toFixed(1)} MB`
}

export function formatDate(value: string) {
  if (!value) return "—"
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString("vi-VN")
}

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; text: string; label: string }> = {
    pending: {
      bg: "bg-[#7B2FFF]/20",
      text: "text-[#7B2FFF]",
      label: "⏳ Đang chờ",
    },
    processing: {
      bg: "bg-[#FF6B35]/20",
      text: "text-[#FF6B35]",
      label: "⚡ Đang xử lý",
    },
    completed: {
      bg: "bg-[#00F5D4]/20",
      text: "text-[#00F5D4]",
      label: "✓ Hoàn tất",
    },
    failed: {
      bg: "bg-red-500/20",
      text: "text-red-300",
      label: "✕ Thất bại",
    },
  }

  const s = map[status] ?? {
    bg: "bg-white/10",
    text: "text-white/60",
    label: status || "Không rõ",
  }

  return (
    <span
      className={`${s.bg} ${s.text} text-xs font-bold uppercase tracking-widest px-3 py-1 rounded-full border border-current`}
    >
      {s.label}
    </span>
  )
}
