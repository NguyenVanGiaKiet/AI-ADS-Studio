import { useEffect, useState } from 'react';
import { ACCENTS, API_BASE, fetchApiData, formatDate, formatFileSize, OutputVideo } from './shared';

function getVideoDayKey(createdAt: string) {
  const date = new Date(createdAt);
  if (Number.isNaN(date.getTime())) return 'unknown';
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function getVideoDayLabel(dayKey: string) {
  if (dayKey === 'unknown') return 'Không rõ ngày tạo';
  const [year, month, day] = dayKey.split('-').map(Number);
  const date = new Date(year, month - 1, day);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (date.getTime() === today.getTime()) return 'Hôm nay';
  if (date.getTime() === yesterday.getTime()) return 'Hôm qua';
  return date.toLocaleDateString('vi-VN', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

export default function CreatedVideosPage({ onNavigateToCreate }: { onNavigateToCreate?: () => void }) {
  const [videos, setVideos] = useState<OutputVideo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedDay, setSelectedDay] = useState('all');
  const [isDownloading, setIsDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState('');

  async function fetchVideos() {
    setLoading(true);
    setError('');
    try {
      setVideos(await fetchApiData<OutputVideo[]>('/api/videos'));
    } catch (err) {
      console.error('Fetch videos error:', err);
      setError(err instanceof Error ? err.message : 'Không thể tải danh sách video.');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchVideos();
  }, []);

  async function downloadFilteredVideos() {
    if (filteredVideos.length === 0 || isDownloading) return;
    setIsDownloading(true);
    setDownloadError('');
    try {
      const response = await fetch(`${API_BASE}/api/videos/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ videoIds: filteredVideos.map(video => video.id) }),
      });
      if (!response.ok) {
        throw new Error((await response.text()) || `HTTP ${response.status}`);
      }
      const archiveUrl = URL.createObjectURL(await response.blob());
      const link = document.createElement('a');
      link.href = archiveUrl;
      link.download = 'ai-ads-studio-videos.zip';
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(archiveUrl), 1000);
    } catch (downloadFailure) {
      setDownloadError(
        downloadFailure instanceof Error
          ? downloadFailure.message
          : 'Không thể tải các video đang lọc.',
      );
    } finally {
      setIsDownloading(false);
    }
  }

  const availableDays = [...new Set(videos.map(video => getVideoDayKey(video.createdAt)))]
    .sort((left, right) => {
      if (left === 'unknown') return 1;
      if (right === 'unknown') return -1;
      return right.localeCompare(left);
    });
  const filteredVideos = videos.filter(video => {
    const matchesSearch =
      video.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      video.filename.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesSearch && (selectedDay === 'all' || getVideoDayKey(video.createdAt) === selectedDay);
  });
  const videosByDay = new Map<string, OutputVideo[]>();
  for (const video of filteredVideos) {
    const dayKey = getVideoDayKey(video.createdAt);
    const dayVideos = videosByDay.get(dayKey) ?? [];
    dayVideos.push(video);
    videosByDay.set(dayKey, dayVideos);
  }
  const sortedDayGroups = [...videosByDay.entries()].sort(([left], [right]) => {
    if (left === 'unknown') return 1;
    if (right === 'unknown') return -1;
    return right.localeCompare(left);
  });

  return (
    <div className="space-y-8 text-white pb-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#FFE600]">
            🎥 Video Library
          </div>
          <h1 className="font-['Unbounded'] text-3xl font-black text-shadow-lg md:text-4xl">
            Video <span className="gradient-text">Đã Tạo</span>
          </h1>
        </div>
        <div className="flex gap-2">
          {videos.length > 0 && (
            <button
              type="button"
              onClick={downloadFilteredVideos}
              disabled={filteredVideos.length === 0 || isDownloading || loading || Boolean(error)}
              className="rounded-full border-4 border-[#FFE600] bg-[#FFE600]/10 px-5 py-2.5 text-xs font-black uppercase tracking-widest text-[#FFE600] transition hover:bg-[#FFE600] hover:text-[#0D0D1A] disabled:cursor-not-allowed disabled:opacity-40"
            >
              {isDownloading
                ? 'Đang nén…'
                : `⬇ Tải video đang lọc (${filteredVideos.length})`}
            </button>
          )}
          <button
            type="button"
            onClick={fetchVideos}
            className="rounded-full border-4 border-[#00F5D4] bg-[#0D0D1A]/80 px-5 py-2.5 text-xs font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
          >
            🔄 Tải lại
          </button>
          {onNavigateToCreate && (
            <button
              type="button"
              onClick={onNavigateToCreate}
              className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-6 py-2.5 text-xs font-black uppercase tracking-widest text-white transition hover:scale-105"
            >
              + Remix Video Mới
            </button>
          )}
        </div>
      </div>
      {downloadError && (
        <p role="alert" className="-mt-6 text-xs text-[#FF8B75]">
          Không thể tải video: {downloadError}
        </p>
      )}

      {/* Filter / Search Bar */}
      <div className="flex flex-col gap-3 rounded-2xl border-4 border-[#7B2FFF]/60 bg-[#2D1B4E] p-3 sm:flex-row sm:items-center">
        <label className="flex min-w-0 flex-1 items-center gap-3">
        <span className="text-xl">🔍</span>
        <input
          type="text"
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
          placeholder="Tìm kiếm theo tên video hoặc file name..."
          className="min-w-0 flex-1 bg-transparent text-sm font-bold text-white placeholder-white/30 outline-none"
        />
        {searchTerm && (
          <button
            type="button"
            onClick={() => setSearchTerm('')}
            className="text-xs text-white/50 hover:text-white"
          >
            Xóa
          </button>
        )}
        </label>
        <label className="flex shrink-0 items-center gap-2 border-t border-white/10 pt-3 sm:border-l sm:border-t-0 sm:pl-3 sm:pt-0">
          <span className="text-xs font-black uppercase tracking-widest text-white/50">Ngày</span>
          <select
            value={selectedDay}
            onChange={event => setSelectedDay(event.target.value)}
            className="max-w-full rounded-xl border border-[#7B2FFF]/60 bg-[#0D0D1A] px-3 py-2 text-xs font-bold text-white outline-none focus:border-[#00F5D4]"
            aria-label="Lọc video theo ngày tạo"
          >
            <option value="all">Tất cả ngày</option>
            {availableDays.map(day => (
              <option key={day} value={day}>{getVideoDayLabel(day)}</option>
            ))}
          </select>
        </label>
      </div>

      {/* Loading State */}
      {loading && (
        <div className="rounded-3xl border-4 border-dashed border-[#00F5D4]/40 bg-[#2D1B4E]/60 p-12 text-center">
          <div className="mb-3 text-4xl animate-bounce">⏳</div>
          <p className="text-sm font-bold text-[#00F5D4]">Đang tải danh sách video đã tạo từ Server Go...</p>
        </div>
      )}

      {/* Error State */}
      {error && !loading && (
        <div className="rounded-3xl border-4 border-[#FF6B35] bg-[#2D1B4E] p-6 text-center">
          <p className="text-sm font-bold text-[#FF6B35]">{error}</p>
          <button
            type="button"
            onClick={fetchVideos}
            className="mt-4 rounded-full border-2 border-[#FFE600] bg-[#FFE600]/10 px-5 py-2 text-xs font-black uppercase tracking-widest text-[#FFE600]"
          >
            Thử lại
          </button>
        </div>
      )}

      {/* Empty State */}
      {!loading && !error && videos.length === 0 && (
        <div className="rounded-3xl border-4 border-dashed border-[#FF3AF2]/50 bg-[#2D1B4E] p-12 text-center">
          <div className="mb-4 text-5xl">🎬</div>
          <h2 className="font-['Unbounded'] text-lg font-black text-white">Chưa có video nào được tạo</h2>
          <p className="mt-2 text-xs text-white/60">Hãy chuyển sang tab Remix Video để tải video nguồn và tạo bộ video quảng cáo đầu tiên!</p>
          {onNavigateToCreate && (
            <button
              type="button"
              onClick={onNavigateToCreate}
              className="mt-6 rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-8 py-3 text-xs font-black uppercase tracking-widest text-white transition hover:scale-105"
            >
              + Tạo Video Ngay
            </button>
          )}
        </div>
      )}

      {!loading && !error && videos.length > 0 && filteredVideos.length === 0 && (
        <div className="rounded-3xl border-4 border-dashed border-[#7B2FFF]/50 bg-[#2D1B4E] p-10 text-center">
          <p className="text-sm font-bold text-white/70">
            {searchTerm
              ? `Không tìm thấy video phù hợp với “${searchTerm}”${selectedDay === 'all' ? '' : ` trong ngày ${getVideoDayLabel(selectedDay)}`}.`
              : `Không có video trong ngày ${getVideoDayLabel(selectedDay)}.`}
          </p>
          <button
            type="button"
            onClick={() => {
              setSearchTerm('');
              setSelectedDay('all');
            }}
            className="mt-4 rounded-full border-2 border-[#00F5D4] px-5 py-2 text-xs font-black uppercase tracking-widest text-[#00F5D4] hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
          >
            Xóa bộ lọc
          </button>
        </div>
      )}

      {sortedDayGroups.length > 0 && (
        <div className="space-y-8">
          {sortedDayGroups.map(([dayKey, dayVideos]) => (
            <section key={dayKey} className="space-y-4">
              <div className="flex items-center justify-between gap-3 border-b-2 border-dashed border-[#7B2FFF]/50 pb-3">
                <h2 className="font-['Unbounded'] text-sm font-black uppercase tracking-wide text-[#00F5D4]">
                  📅 {getVideoDayLabel(dayKey)}
                </h2>
                <span className="rounded-full border border-[#7B2FFF]/60 bg-[#2D1B4E] px-3 py-1 text-[11px] font-bold text-white/60">
                  {dayVideos.length} video
                </span>
              </div>
              <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {dayVideos.map((video, index) => {
                  const accent = ACCENTS[index % ACCENTS.length];
                  const videoSrc = `${API_BASE}${video.url}`;

                  return (
                    <div
                      key={video.id}
                      className="group relative flex flex-col justify-between overflow-hidden rounded-3xl border-4 bg-[#2D1B4E] p-4 transition-all duration-300 hover:scale-[1.02]"
                      style={{
                        borderColor: accent,
                        boxShadow: `6px 6px 0 ${ACCENTS[(index + 1) % ACCENTS.length]}`,
                      }}
                    >
                      <div>
                        <div className="relative mb-3 overflow-hidden rounded-2xl border-2 border-[#0D0D1A] bg-[#0D0D1A]">
                          <video
                            src={videoSrc}
                            controls
                            preload="metadata"
                            className="aspect-[9/16] w-full object-contain"
                          />
                        </div>
                        <h3 className="font-['Unbounded'] text-sm font-black text-white group-hover:text-[#00F5D4]">
                          {video.title || video.filename}
                        </h3>
                        <p className="mt-1 truncate text-xs font-bold text-white/60">
                          📄 {video.filename}
                        </p>
                        <div className="mt-3 flex items-center justify-between text-[11px] font-bold text-white/50">
                          <span>⏱ {video.duration}s</span>
                          <span>💾 {formatFileSize(video.size)}</span>
                        </div>
                        <div className="mt-1 text-[10px] text-white/40">
                          🕒 {formatDate(video.createdAt)}
                        </div>
                      </div>
                      <div className="mt-4 flex gap-2 border-t-2 border-dashed border-white/10 pt-3">
                        <a
                          href={videoSrc}
                          target="_blank"
                          rel="noreferrer"
                          download
                          className="flex-1 rounded-2xl border-2 border-[#00F5D4] bg-[#00F5D4]/10 py-2 text-center text-xs font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
                        >
                          Tải về ⬇
                        </a>
                        <a
                          href={videoSrc}
                          target="_blank"
                          rel="noreferrer"
                          className="rounded-2xl border-2 border-white/20 bg-[#0D0D1A]/60 px-4 py-2 text-xs font-black uppercase tracking-widest text-white/70 hover:border-white hover:text-white"
                        >
                          Xem ↗
                        </a>
                      </div>
                    </div>
                  );
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
