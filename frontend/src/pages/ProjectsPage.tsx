import { useEffect, useState } from "react"

import {
  ACCENTS,
  fetchApiData,
  formatDate,
  RemixTask,
  StatusBadge,
} from "./shared"

export default function ProjectsPage({
  onNavigateToCreate,
}: {
  onNavigateToCreate: () => void
}) {
  const [tasks, setTasks] = useState<RemixTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const sortedTasks = [...tasks].sort(
    (left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime(),
  )

  useEffect(() => {
    let active = true
    fetchApiData<RemixTask[]>("/api/remix/tasks")
      .then((data) => {
        if (active) setTasks(data)
      })
      .catch((requestError: unknown) => {
        if (active) {
          setError(requestError instanceof Error ? requestError.message : "Không thể tải tác vụ.")
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  return (
    <div className="space-y-7 text-white">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#FF6B35]">
            ⚙️ Tiến trình xử lý
          </div>
          <h1 className="font-['Unbounded'] text-3xl font-black">
            <span className="gradient-text">Lịch sử tác vụ</span>
          </h1>
          <p className="mt-2 text-sm text-white/50">
            Theo dõi các lần remix video do backend hiện tại quản lý.
          </p>
        </div>
        <button
          type="button"
          onClick={onNavigateToCreate}
          className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] to-[#7B2FFF] px-6 py-3 text-xs font-black uppercase tracking-widest text-white transition hover:scale-105"
        >
          + Tạo tác vụ mới
        </button>
      </header>

      <div className="rounded-2xl border border-[#FFE600]/40 bg-[#FFE600]/5 p-4 text-xs leading-relaxed text-white/65">
        Danh sách này là lịch sử tác vụ trong bộ nhớ backend và có thể được làm
        mới khi khởi động lại server. Video đã xuất được lưu riêng trong thư
        viện <strong className="text-[#FFE600]">Video đã tạo</strong>.
      </div>

      {error && (
        <div className="rounded-2xl border-2 border-[#FF6B35] bg-[#2D1B4E] p-4 text-sm text-[#FFB08E]">
          Không tải được lịch sử tác vụ: {error}
        </div>
      )}

      {loading ? (
        <div className="rounded-3xl border-2 border-dashed border-[#7B2FFF]/50 bg-[#2D1B4E] p-10 text-center text-sm text-white/60">
          Đang tải tác vụ…
        </div>
      ) : error ? null : tasks.length === 0 ? (
        <div className="rounded-3xl border-2 border-dashed border-[#7B2FFF]/50 bg-[#2D1B4E] p-10 text-center">
          <div className="text-4xl" aria-hidden="true">🧩</div>
          <h2 className="mt-3 font-['Unbounded'] text-lg font-black">Chưa có tác vụ remix</h2>
          <p className="mt-2 text-sm text-white/50">
            Tạo tác vụ đầu tiên từ video nguồn của bạn.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {sortedTasks.map((task, index) => {
            const outputs = task.outputVideos?.length ?? 0
            return (
              <article
                key={task.id}
                className="rounded-3xl border-4 bg-[#2D1B4E] p-5"
                style={{
                  borderColor: ACCENTS[index % ACCENTS.length],
                  boxShadow: `6px 6px 0 ${ACCENTS[(index + 1) % ACCENTS.length]}`,
                }}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="break-all font-['Unbounded'] text-sm font-black text-white">
                      Tác vụ {task.id.slice(0, 8)}
                    </h2>
                    <p className="mt-1 text-xs text-white/45">
                      Bắt đầu: {formatDate(task.createdAt)}
                    </p>
                  </div>
                  <StatusBadge status={task.status} />
                </div>
                <p className="mt-4 text-sm text-white/75">
                  {task.error || task.message || "Chưa có thông tin tiến trình."}
                </p>
                <div className="mt-4 h-2 overflow-hidden rounded-full bg-[#0D0D1A]">
                  <div
                    className="h-full rounded-full bg-gradient-to-r from-[#FF3AF2] to-[#00F5D4] transition-all"
                    style={{ width: `${Math.min(100, Math.max(0, task.progress))}%` }}
                  />
                </div>
                <div className="mt-2 flex flex-wrap justify-between gap-2 text-xs text-white/50">
                  <span>{Math.min(100, Math.max(0, task.progress))}% · {outputs} video đầu ra</span>
                  <span>
                    Yêu cầu {task.request.outputCount} video · {task.request.duration}s
                    {task.request.replaceVoice ? " · Có thay giọng" : " · Giữ âm thanh"}
                  </span>
                </div>
              </article>
            )
          })}
        </div>
      )}
    </div>
  )
}
