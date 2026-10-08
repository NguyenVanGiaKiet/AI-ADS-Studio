import { useEffect, useState } from "react"

import {
  ACCENTS,
  API_BASE,
  fetchApiData,
  formatDate,
  OutputVideo,
  RemixTask,
} from "./shared"

export default function DashboardPage({
  onNavigate,
}: {
  onNavigate: (tab: "create" | "videos" | "projects") => void
}) {
  const [videos, setVideos] = useState<OutputVideo[]>([])
  const [tasks, setTasks] = useState<RemixTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  useEffect(() => {
    let active = true
    Promise.all([
      fetchApiData<OutputVideo[]>("/api/videos"),
      fetchApiData<RemixTask[]>("/api/remix/tasks"),
    ])
      .then(([videoData, taskData]) => {
        if (!active) return
        setVideos(videoData)
        setTasks(taskData)
      })
      .catch((requestError: unknown) => {
        if (active) {
          setError(
            requestError instanceof Error
              ? requestError.message
              : "Không thể tải dữ liệu tổng quan.",
          )
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const activeTasks = tasks.filter(
    (task) => task.status === "pending" || task.status === "processing",
  ).length
  const totalDuration = videos.reduce((total, video) => total + video.duration, 0)
  const latestVideos = videos.slice(0, 4)

  return (
    <div className="space-y-8 text-white">
      <section
        className="relative overflow-hidden rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-8 md:p-10"
        style={{ boxShadow: "12px 12px 0 #FFE600, 24px 24px 0 #FF3AF2" }}
      >
        <div className="pointer-events-none absolute inset-0 pattern-mesh opacity-40" aria-hidden="true" />
        <div className="relative z-10 flex flex-col justify-between gap-6 md:flex-row md:items-center">
          <div>
            <div className="mb-2 text-sm font-black uppercase tracking-widest text-[#00F5D4]">
              🎬 AI ADS STUDIO
            </div>
            <h1 className="font-['Unbounded'] text-3xl font-black leading-tight md:text-5xl">
              Tạo video quảng cáo
              <br />
              <span className="gradient-text">từ video sản phẩm</span>
            </h1>
            <p className="mt-3 max-w-xl text-sm leading-relaxed text-white/70">
              Tải video nguồn, chọn cách dựng và tạo lời thoại tiếng Việt với
              các giọng Piper chạy cục bộ. Theo dõi và tải video đã xuất ngay
              trong ứng dụng.
            </p>
          </div>
          <button
            type="button"
            onClick={() => onNavigate("create")}
            className="shrink-0 rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-7 py-4 text-sm font-black uppercase tracking-widest text-white transition hover:scale-105"
          >
            + Remix video
          </button>
        </div>
      </section>

      {error && (
        <div className="rounded-2xl border-2 border-[#FF6B35] bg-[#2D1B4E] p-4 text-sm text-[#FFB08E]">
          Không thể tải số liệu từ backend: {error}. Hãy kiểm tra backend Go
          rồi tải lại trang.
        </div>
      )}

      <section>
        <h2 className="mb-4 font-['Unbounded'] text-lg font-black uppercase tracking-wide">
          Tổng quan dữ liệu thực tế
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {[
            {
              label: "Video đã tạo",
              value: loading ? "…" : error ? "—" : videos.length.toLocaleString("vi-VN"),
              note: "Tổng số video đầu ra",
              icon: "🎞️",
            },
            {
              label: "Tác vụ đang chạy",
              value: loading ? "…" : error ? "—" : activeTasks.toLocaleString("vi-VN"),
              note: "Đang chờ hoặc đang xử lý",
              icon: "⚙️",
            },
            {
              label: "Tổng thời lượng",
              value: loading ? "…" : error ? "—" : `${totalDuration.toLocaleString("vi-VN")} giây`,
              note: "Cộng từ video đã xuất",
              icon: "⏱️",
            },
          ].map((stat, index) => (
            <div
              key={stat.label}
              className="rounded-3xl border-4 bg-[#2D1B4E] p-5"
              style={{
                borderColor: ACCENTS[index],
                boxShadow: `6px 6px 0 ${ACCENTS[(index + 1) % ACCENTS.length]}`,
              }}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-black uppercase tracking-widest text-white/60">
                  {stat.label}
                </span>
                <span className="text-2xl" aria-hidden="true">{stat.icon}</span>
              </div>
              <div className="mt-4 break-words font-['Unbounded'] text-2xl font-black" style={{ color: ACCENTS[index] }}>
                {stat.value}
              </div>
              <p className="mt-2 text-xs text-white/45">{stat.note}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="overflow-hidden rounded-3xl border-4 border-[#7B2FFF] bg-[#2D1B4E]">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b-2 border-dashed border-[#7B2FFF]/50 px-5 py-4">
          <h2 className="font-['Unbounded'] text-sm font-black uppercase tracking-wide">
            Video mới tạo
          </h2>
          <button
            type="button"
            onClick={() => onNavigate("videos")}
            className="text-xs font-black uppercase tracking-widest text-[#00F5D4] hover:text-[#FFE600]"
          >
            Mở thư viện →
          </button>
        </div>
        {error ? (
          <p className="p-8 text-center text-sm text-white/50">
            Không thể hiển thị video mới tạo khi chưa tải được dữ liệu.
          </p>
        ) : latestVideos.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-sm font-bold text-white/70">
              {loading ? "Đang tải video…" : "Chưa có video đầu ra."}
            </p>
            {!loading && (
              <button
                type="button"
                onClick={() => onNavigate("create")}
                className="mt-4 rounded-full border-2 border-[#FFE600] px-5 py-2 text-xs font-black uppercase tracking-widest text-[#FFE600] transition hover:bg-[#FFE600] hover:text-[#0D0D1A]"
              >
                Bắt đầu remix
              </button>
            )}
          </div>
        ) : (
          <div className="divide-y divide-dashed divide-[#7B2FFF]/30">
            {latestVideos.map((video) => (
              <div key={video.id} className="flex flex-wrap items-center justify-between gap-2 px-5 py-4">
                <div className="min-w-0">
                  <p className="truncate text-sm font-bold text-white">
                    {video.title || video.filename}
                  </p>
                  <p className="mt-1 text-xs text-white/45">
                    {video.duration}s · {formatDate(video.createdAt)}
                  </p>
                </div>
                <a
                  href={`${API_BASE}${video.url}`}
                  download
                  className="rounded-full border border-[#00F5D4] px-4 py-2 text-xs font-black text-[#00F5D4] hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
                >
                  Tải xuống
                </a>
              </div>
            ))}
          </div>
        )}
      </section>

      <button
        type="button"
        onClick={() => onNavigate("projects")}
        className="text-xs font-bold text-white/45 underline decoration-dashed underline-offset-4 hover:text-white"
      >
        Xem lịch sử tác vụ remix
      </button>
    </div>
  )
}
