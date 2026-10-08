import { useEffect, useMemo, useState } from "react"

import {
  ACCENTS,
  fetchApiData,
  formatDate,
  formatFileSize,
  OutputVideo,
  RemixTask,
  StatusBadge,
} from "./shared"

export default function AnalyticsPage() {
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
          setError(requestError instanceof Error ? requestError.message : "Không thể tải số liệu.")
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const dailyCounts = useMemo(() => {
    const days = Array.from({ length: 7 }, (_, index) => {
      const date = new Date()
      date.setHours(0, 0, 0, 0)
      date.setDate(date.getDate() - (6 - index))
      return { date, count: 0 }
    })
    const countsByDate = new Map(days.map((day) => [day.date.toDateString(), day]))
    for (const video of videos) {
      const created = new Date(video.createdAt)
      if (Number.isNaN(created.getTime())) continue
      const day = countsByDate.get(
        new Date(created.getFullYear(), created.getMonth(), created.getDate()).toDateString(),
      )
      if (day) day.count++
    }
    return days
  }, [videos])

  const maxDailyCount = Math.max(1, ...dailyCounts.map((day) => day.count))
  const totalSize = videos.reduce((sum, video) => sum + video.size, 0)
  const totalDuration = videos.reduce((sum, video) => sum + video.duration, 0)
  const completedTasks = tasks.filter((task) => task.status === "completed").length
  const failedTasks = tasks.filter((task) => task.status === "failed").length
  const recentTasks = [...tasks].sort(
    (left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime(),
  )

  return (
    <div className="space-y-8 text-white">
      <header>
        <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#FF6B35]">
          📊 Thống kê cục bộ
        </div>
        <h1 className="font-['Unbounded'] text-3xl font-black md:text-4xl">
          Số liệu <span className="gradient-text">video đã tạo</span>
        </h1>
        <p className="mt-2 text-sm text-white/50">
          Thống kê lấy từ video đầu ra và tác vụ backend; không phải lượt xem,
          CTR hay hiệu quả quảng cáo trên các nền tảng.
        </p>
      </header>

      {error && (
        <div className="rounded-2xl border-2 border-[#FF6B35] bg-[#2D1B4E] p-4 text-sm text-[#FFB08E]">
          Không tải được số liệu: {error}
        </div>
      )}

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[
          ["Video đã tạo", loading ? "…" : error ? "—" : videos.length.toLocaleString("vi-VN"), "🎞️"],
          ["Tổng thời lượng", loading ? "…" : error ? "—" : `${totalDuration.toLocaleString("vi-VN")} giây`, "⏱️"],
          ["Dung lượng đầu ra", loading ? "…" : error ? "—" : formatFileSize(totalSize), "💾"],
          ["Tác vụ hoàn tất / lỗi", loading ? "…" : error ? "—" : `${completedTasks} / ${failedTasks}`, "⚙️"],
        ].map(([label, value, icon], index) => (
          <div
            key={label}
            className="rounded-3xl border-4 bg-[#2D1B4E] p-5"
            style={{ borderColor: ACCENTS[index], boxShadow: `5px 5px 0 ${ACCENTS[(index + 1) % ACCENTS.length]}` }}
          >
            <div className="flex justify-between text-xs font-black uppercase tracking-widest text-white/55">
              {label}<span className="text-xl" aria-hidden="true">{icon}</span>
            </div>
            <div className="mt-4 break-words font-['Unbounded'] text-2xl font-black" style={{ color: ACCENTS[index] }}>
              {value}
            </div>
          </div>
        ))}
      </section>

      <section className="rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-6">
        <h2 className="font-['Unbounded'] text-sm font-black uppercase tracking-wide text-[#FF3AF2]">
          Video đầu ra trong 7 ngày gần nhất
        </h2>
        {loading ? (
          <p className="mt-6 text-sm text-white/50">Đang tải số liệu…</p>
        ) : error ? (
          <p className="mt-6 text-sm text-white/50">Không thể lập biểu đồ khi chưa tải được dữ liệu.</p>
        ) : (
          <div className="mt-6 flex h-48 items-end gap-2">
            {dailyCounts.map((day, index) => (
              <div key={day.date.toISOString()} className="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-2">
                <span className="text-xs font-bold text-white/65">{day.count}</span>
                <div
                  className="w-full rounded-t-lg transition-all"
                  style={{
                    height: `${Math.max(day.count > 0 ? 8 : 0, (day.count / maxDailyCount) * 70)}%`,
                    background: ACCENTS[index % ACCENTS.length],
                  }}
                  title={`${day.count} video`}
                />
                <span className="text-[10px] text-white/45">
                  {day.date.toLocaleDateString("vi-VN", { weekday: "short" })}
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="overflow-hidden rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E]">
        <div className="border-b-2 border-dashed border-[#FFE600]/40 px-5 py-4">
          <h2 className="font-['Unbounded'] text-sm font-black uppercase tracking-wide text-[#FFE600]">
            Tác vụ gần đây
          </h2>
        </div>
        {loading ? (
          <p className="p-6 text-sm text-white/50">Đang tải tác vụ…</p>
        ) : error ? (
          <p className="p-6 text-sm text-white/50">Không thể tải danh sách tác vụ.</p>
        ) : tasks.length === 0 ? (
          <p className="p-6 text-sm text-white/50">Chưa có tác vụ để thống kê.</p>
        ) : (
          <div className="divide-y divide-dashed divide-[#FFE600]/20">
            {recentTasks.slice(0, 8).map((task) => (
              <div key={task.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
                <div>
                  <p className="break-all text-sm font-bold text-white">Tác vụ {task.id.slice(0, 8)}</p>
                  <p className="mt-1 text-xs text-white/45">
                    {formatDate(task.createdAt)} · {task.outputVideos?.length ?? 0} video đầu ra
                  </p>
                </div>
                <StatusBadge status={task.status} />
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
