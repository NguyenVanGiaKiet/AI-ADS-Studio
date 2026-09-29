import { useEffect, useState } from 'react';
import { ACCENTS } from './shared';

const API_BASE = 'http://localhost:8080';

interface OutputVideo {
  id: string;
  taskId: string;
  title: string;
  filename: string;
  url: string;
  duration: number;
  size: number;
  createdAt: string;
}

function formatFileSize(size: number) {
  return size < 1024 * 1024
    ? `${(size / 1024).toFixed(0)} KB`
    : `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(isoStr: string) {
  if (!isoStr) return '';
  const d = new Date(isoStr);
  return d.toLocaleString('vi-VN');
}

export default function CreatedVideosPage({ onNavigateToCreate }: { onNavigateToCreate?: () => void }) {
  const [videos, setVideos] = useState<OutputVideo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchTerm, setSearchTerm] = useState('');

  async function fetchVideos() {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${API_BASE}/api/videos`);
      if (!res.ok) {
        throw new Error(`HTTP error ${res.status}`);
      }
      const json = await res.json();
      setVideos(json.data || []);
    } catch (err) {
      console.error('Fetch videos error:', err);
      setError('Chưa kết nối được tới Go Backend hoặc chưa có video.');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchVideos();
  }, []);

  const filteredVideos = videos.filter(v =>
    v.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
    v.filename.toLowerCase().includes(searchTerm.toLowerCase())
  );

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

      {/* Filter / Search Bar */}
      <div className="flex items-center gap-3 rounded-2xl border-4 border-[#7B2FFF]/60 bg-[#2D1B4E] p-3">
        <span className="text-xl">🔍</span>
        <input
          type="text"
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
          placeholder="Tìm kiếm theo tên video hoặc file name..."
          className="w-full bg-transparent text-sm font-bold text-white placeholder-white/30 outline-none"
        />
        {searchTerm && (
          <button
            type="button"
            onClick={() => setSearchTerm('')}
            className="text-xs text-white/50 hover:text-white"
          >
            Clear
          </button>
        )}
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

      {/* Video Grid */}
      {!loading && filteredVideos.length > 0 && (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filteredVideos.map((video, idx) => {
            const accent = ACCENTS[idx % ACCENTS.length];
            const videoSrc = `${API_BASE}${video.url}`;

            return (
              <div
                key={video.id}
                className="group relative flex flex-col justify-between overflow-hidden rounded-3xl border-4 bg-[#2D1B4E] p-4 transition-all duration-300 hover:scale-[1.02]"
                style={{
                  borderColor: accent,
                  boxShadow: `6px 6px 0 ${ACCENTS[(idx + 1) % ACCENTS.length]}`,
                }}
              >
                <div>
                  {/* Video Player */}
                  <div className="relative mb-3 overflow-hidden rounded-2xl border-2 border-[#0D0D1A] bg-[#0D0D1A]">
                    <video
                      src={videoSrc}
                      controls
                      preload="metadata"
                      className="aspect-[9/16] w-full object-contain"
                    />
                  </div>

                  {/* Title & Info */}
                  <h3 className="font-['Unbounded'] text-sm font-black text-white group-hover:text-[#00F5D4]">
                    {video.title || `Remix Video #${idx + 1}`}
                  </h3>
                  <p className="mt-1 truncate text-xs font-bold text-white/60">
                    📄 {video.filename}
                  </p>
                  <div className="mt-3 flex items-center justify-between text-[11px] font-bold text-white/50">
                    <span>⏱ {video.duration}s</span>
                    <span>💾 {formatFileSize(video.size)}</span>
                  </div>
                  {video.createdAt && (
                    <div className="mt-1 text-[10px] text-white/40">
                      📅 {formatDate(video.createdAt)}
                    </div>
                  )}
                </div>

                {/* Actions */}
                <div className="mt-4 flex gap-2 pt-3 border-t-2 border-dashed border-white/10">
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
      )}
    </div>
  );
}
