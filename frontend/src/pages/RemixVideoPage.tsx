import { useEffect, useRef, useState } from 'react';
import { ACCENTS } from './shared';

type DeduplicationLevel = 'off' | 'light' | 'medium' | 'strong';

const API_BASE = 'http://localhost:8080';
const ACCEPTED_VIDEO_TYPES = ['mp4', 'mov', 'avi', 'mkv', 'webm'];
const VOICES = [
  'Ngọc Huyền (Vbee) - Nữ Bắc - trong, rõ ràng',
  'Minh Anh - Nữ Nam - trẻ trung, thân thiện',
  'Hoàng Nam - Nam Bắc - ấm, tin cậy',
  'Gia Bảo - Nam Nam - năng động, gần gũi',
];

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

interface RemixTaskResponse {
  id: string;
  status: string;
  progress: number;
  message: string;
  outputVideos?: OutputVideo[];
}

function formatFileSize(size: number) {
  return size < 1024 * 1024
    ? `${(size / 1024).toFixed(0)} KB`
    : `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function StepTitle({ number, children }: { number: number; children: React.ReactNode }) {
  const accent = ACCENTS[(number - 1) % ACCENTS.length];

  return (
    <div className="mb-5 flex items-center gap-3">
      <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl border-4 border-[#FFE600] text-sm font-black text-[#0D0D1A]" style={{ background: accent, boxShadow: `4px 4px 0 ${ACCENTS[number % ACCENTS.length]}` }}>
        {number}
      </span>
      <h2 className="font-['Unbounded'] text-sm font-black uppercase tracking-wide text-white text-shadow-sm sm:text-base">{children}</h2>
    </div>
  );
}

function Field({ label, hint, children, accent = ACCENTS[1] }: { label: string; hint?: string; children: React.ReactNode; accent?: string }) {
  return (
    <label className="block min-w-0">
      <span className="mb-2 block text-xs font-black uppercase tracking-widest" style={{ color: accent }}>{label}</span>
      {children}
      {hint && <span className="mt-1.5 block text-[11px] leading-relaxed text-white/40">{hint}</span>}
    </label>
  );
}

const inputClassName = 'w-full rounded-2xl border-4 border-[#7B2FFF]/70 bg-[#0D0D1A]/80 px-4 py-3 text-sm font-bold text-white outline-none transition placeholder:text-white/25 focus:border-[#00F5D4] focus:ring-4 focus:ring-[#00F5D4]/15';
const panelClassNames = [
  'rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-5 shadow-[8px_8px_0_#FFE600] sm:p-6',
  'rounded-3xl border-4 border-[#00F5D4] bg-[#2D1B4E] p-5 shadow-[8px_8px_0_#FF6B35] sm:p-6',
  'rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E] p-5 shadow-[8px_8px_0_#FF3AF2] sm:p-6',
  'rounded-3xl border-4 border-[#7B2FFF] bg-[#2D1B4E] p-5 shadow-[8px_8px_0_#00F5D4] sm:p-6',
];

export default function RemixVideoPage() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [videos, setVideos] = useState<File[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const [outputCount, setOutputCount] = useState(5);
  const [duration, setDuration] = useState(30);
  const [aspectRatio, setAspectRatio] = useState('vertical');
  const [remixMode, setRemixMode] = useState('standard');
  const [deduplication, setDeduplication] = useState<DeduplicationLevel>('off');
  const [replaceVoice, setReplaceVoice] = useState(true);
  const [followSubtitles, setFollowSubtitles] = useState(false);
  const [productDescription, setProductDescription] = useState('');
  const [scriptStyle, setScriptStyle] = useState('professional');
  const [voice, setVoice] = useState(VOICES[0]);
  const [speechRate, setSpeechRate] = useState(1);
  const [notice, setNotice] = useState('');
  const [isSpeaking, setIsSpeaking] = useState(false);

  // Backend Task State
  const [isProcessing, setIsProcessing] = useState(false);
  const [currentTask, setCurrentTask] = useState<RemixTaskResponse | null>(null);
  const [outputVideos, setOutputVideos] = useState<OutputVideo[]>([]);

  useEffect(() => () => window.speechSynthesis?.cancel(), []);

  function addVideos(incomingFiles: FileList | File[]) {
    const incoming = Array.from(incomingFiles);
    const validFiles = incoming.filter(file => {
      const extension = file.name.split('.').pop()?.toLowerCase();
      return extension && ACCEPTED_VIDEO_TYPES.includes(extension);
    });
    const rejectedCount = incoming.length - validFiles.length;

    setVideos(current => {
      const nextFiles = [...current];
      for (const file of validFiles) {
        const alreadyAdded = nextFiles.some(existing =>
          existing.name === file.name && existing.size === file.size && existing.lastModified === file.lastModified,
        );
        if (!alreadyAdded) nextFiles.push(file);
      }
      return nextFiles;
    });

    if (rejectedCount > 0) {
      setNotice(`Đã bỏ qua ${rejectedCount} tệp không đúng định dạng. Hỗ trợ MP4, MOV, AVI, MKV và WEBM.`);
    } else if (validFiles.length > 0) {
      setNotice('');
    }
  }

  function handleFileSelection(event: React.ChangeEvent<HTMLInputElement>) {
    if (event.currentTarget.files) addVideos(event.currentTarget.files);
    event.currentTarget.value = '';
  }

  function handleDrop(event: React.DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setIsDragging(false);
    addVideos(event.dataTransfer.files);
  }

  async function previewVoice() {
    setIsSpeaking(true);
    try {
      const res = await fetch(`${API_BASE}/api/tts/preview`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: productDescription || 'Xin chào, đây là phần nghe thử giọng đọc quảng cáo của bạn.',
          voice,
          rate: speechRate,
        }),
      });
      if (res.ok) {
        setNotice('Đã nhận phản hồi mẫu từ Go Backend!');
      } else {
        fallbackWebSpeech();
      }
    } catch {
      fallbackWebSpeech();
    } finally {
      setTimeout(() => setIsSpeaking(false), 2000);
    }
  }

  function fallbackWebSpeech() {
    if (!('speechSynthesis' in window)) {
      setNotice('Trình duyệt này chưa hỗ trợ nghe thử giọng đọc.');
      return;
    }
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance('Xin chào, đây là phần nghe thử giọng đọc quảng cáo của bạn.');
    utterance.rate = speechRate;
    utterance.lang = 'vi-VN';
    utterance.onstart = () => setIsSpeaking(true);
    utterance.onend = () => setIsSpeaking(false);
    utterance.onerror = () => setIsSpeaking(false);
    window.speechSynthesis.speak(utterance);
  }

  async function startRemix() {
    if (videos.length < 2) {
      setNotice('Vui lòng chọn ít nhất 2 video đầu vào trước khi bắt đầu.');
      return;
    }

    setIsProcessing(true);
    setNotice('Đang kết nối Go Backend & tải lên video nguồn...');
    setCurrentTask(null);
    setOutputVideos([]);

    try {
      // Step 1: Upload Videos
      const formData = new FormData();
      videos.forEach(file => formData.append('files', file));

      const uploadRes = await fetch(`${API_BASE}/api/upload`, {
        method: 'POST',
        body: formData,
      });

      let uploadedVideoIds: string[] = [];
      if (uploadRes.ok) {
        const uploadData = await uploadRes.json();
        uploadedVideoIds = uploadData.data ? uploadData.data.map((v: { id: string }) => v.id) : [];
      }

      // Step 2: Create Remix Task
      const taskRes = await fetch(`${API_BASE}/api/remix/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          videoIds: uploadedVideoIds,
          outputCount,
          duration,
          aspectRatio,
          remixMode,
          deduplication,
          replaceVoice,
          followSubtitles,
          productDescription,
          scriptStyle,
          voice,
          speechRate,
        }),
      });

      if (!taskRes.ok) {
        throw new Error('Không thể tạo tiến trình trên Go Backend');
      }

      const taskJson = await taskRes.json();
      const taskId = taskJson.data.id;
      setCurrentTask(taskJson.data);

      // Step 3: Poll Task Status
      const pollInterval = setInterval(async () => {
        try {
          const statusRes = await fetch(`${API_BASE}/api/remix/tasks/${taskId}`);
          if (statusRes.ok) {
            const statusJson = await statusRes.json();
            const taskData: RemixTaskResponse = statusJson.data;
            setCurrentTask(taskData);
            setNotice(taskData.message);

            if (taskData.status === 'completed') {
              clearInterval(pollInterval);
              setIsProcessing(false);
              if (taskData.outputVideos) {
                setOutputVideos(taskData.outputVideos);
              }
            } else if (taskData.status === 'failed') {
              clearInterval(pollInterval);
              setIsProcessing(false);
            }
          }
        } catch (err) {
          console.error('Polling error:', err);
        }
      }, 800);

    } catch (error) {
      setIsProcessing(false);
      setNotice(`Lỗi: ${error instanceof Error ? error.message : 'Chưa thể kết nối tới Go Backend'}`);
    }
  }

  return (
    <div className="space-y-8 pb-8 text-white">
      <div>
        <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#00F5D4]">
          🎬 Video Editor · Go Backend Powered
        </div>
        <h1 className="font-['Unbounded'] text-3xl font-black text-shadow-lg md:text-4xl">
          Remix <span className="gradient-text">Video</span>
        </h1>
      </div>

      <section className={panelClassNames[0]}>
        <StepTitle number={1}>Chọn video đầu vào</StepTitle>
        <input ref={fileInputRef} type="file" accept="video/mp4,video/quicktime,video/x-msvideo,video/x-matroska,video/webm,.mp4,.mov,.avi,.mkv,.webm" multiple className="sr-only" onChange={handleFileSelection} />
        <div
          role="button"
          tabIndex={0}
          onClick={() => fileInputRef.current?.click()}
          onKeyDown={event => { if (event.key === 'Enter' || event.key === ' ') fileInputRef.current?.click(); }}
          onDragOver={event => { event.preventDefault(); setIsDragging(true); }}
          onDragLeave={event => { if (!event.currentTarget.contains(event.relatedTarget as Node)) setIsDragging(false); }}
          onDrop={handleDrop}
          className={`flex min-h-36 cursor-pointer flex-col items-center justify-center rounded-2xl border-4 border-dashed px-4 py-6 text-center transition ${isDragging ? 'border-[#FFE600] bg-[#FFE600]/10' : 'border-[#00F5D4]/70 bg-[#0D0D1A]/70 hover:border-[#FF3AF2] hover:bg-[#00F5D4]/[0.06]'}`}
        >
          <span className="mb-2 text-3xl" aria-hidden="true">📹</span>
          <p className="text-sm font-bold text-white/70">Kéo thả video vào đây hoặc <span className="font-black text-[#00F5D4]">click để chọn file</span></p>
          <p className="mt-1 text-[11px] text-white/40">MP4, MOV, AVI, MKV, WEBM · cần tối thiểu 2 video</p>
        </div>
        {videos.length > 0 && (
          <div className="mt-3 space-y-2">
            {videos.map((video, index) => (
              <div key={`${video.name}-${video.lastModified}`} className="flex items-center gap-3 rounded-2xl border-2 border-[#7B2FFF]/60 bg-[#0D0D1A]/70 px-3 py-2">
                <span className="flex size-8 shrink-0 items-center justify-center rounded-xl border-2 border-[#00F5D4] bg-[#00F5D4]/10 text-[#00F5D4]" aria-hidden="true">▶</span>
                <div className="min-w-0 flex-1"><p className="truncate text-xs font-bold text-white">{video.name}</p><p className="text-[11px] text-white/40">Video {index + 1} · {formatFileSize(video.size)}</p></div>
                <button type="button" aria-label={`Xóa ${video.name}`} onClick={() => setVideos(current => current.filter((_, fileIndex) => fileIndex !== index))} className="rounded-full px-3 py-1 text-xs font-bold text-white/50 hover:bg-[#FF3AF2]/15 hover:text-[#FF3AF2]">Xóa</button>
              </div>
            ))}
            <button type="button" onClick={() => fileInputRef.current?.click()} className="text-xs font-black uppercase tracking-widest text-[#00F5D4] hover:text-[#FFE600]">+ Thêm video</button>
          </div>
        )}
      </section>

      <section className={panelClassNames[1]}>
        <StepTitle number={2}>Cài đặt video</StepTitle>
        <div className="grid grid-cols-1 gap-x-4 gap-y-5 sm:grid-cols-2 xl:grid-cols-4">
          <Field label="Số lượng video đầu ra" hint="Số video mới cần tạo (1-50)">
            <input className={inputClassName} type="number" min={1} max={50} value={outputCount} onChange={event => setOutputCount(Math.min(50, Math.max(1, Number(event.target.value) || 1)))} />
          </Field>
          <Field label="Thời lượng mong muốn (giây)" hint="Thời lượng mỗi video đầu ra (10-60 giây)">
            <input className={inputClassName} type="number" min={10} max={60} value={duration} onChange={event => setDuration(Math.min(60, Math.max(10, Number(event.target.value) || 10)))} />
          </Field>
          <Field label="Tỷ lệ khung hình" hint="Độ phân giải & định dạng xuất video">
            <select className={inputClassName} value={aspectRatio} onChange={event => setAspectRatio(event.target.value)}>
              <option value="vertical">Dọc — 9:16 (TikTok / Reels / Shorts)</option>
              <option value="square">Vuông — 1:1</option>
              <option value="landscape">Ngang — 16:9</option>
              <option value="portrait">Chân dung — 9:16</option>
            </select>
          </Field>
          <Field label="Chế độ Remix">
            <select className={inputClassName} value={remixMode} onChange={event => setRemixMode(event.target.value)}>
              <option value="standard">Tiêu chuẩn — Cắt ghép bình thường</option>
              <option value="dynamic">Năng động — Nhịp cắt nhanh</option>
              <option value="story">Kể chuyện — Giữ mạch nội dung</option>
            </select>
          </Field>
        </div>

        <div className="mt-6">
          <p className="text-xs font-black uppercase tracking-widest text-[#FFE600]">Chống trùng trên nền tảng</p>
          <p className="mt-1 text-[11px] text-white/45">Mỗi video chỉnh khác nhau một chút để nền tảng khó coi hai video là một bản.</p>
          <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4" role="group" aria-label="Mức độ chống trùng">
            {([
              ['off', 'Tắt'],
              ['light', 'Nhẹ'],
              ['medium', 'Vừa'],
              ['strong', 'Mạnh'],
            ] as [DeduplicationLevel, string][]).map(([value, label], index) => (
              <button key={value} type="button" aria-pressed={deduplication === value} onClick={() => setDeduplication(value)} className={`rounded-2xl border-4 px-3 py-2 text-xs font-black uppercase tracking-widest transition ${deduplication === value ? '' : 'border-[#7B2FFF]/50 bg-[#0D0D1A]/70 text-white/45 hover:border-white/40 hover:text-white'}`} style={deduplication === value ? { borderColor: ACCENTS[index % ACCENTS.length], background: `${ACCENTS[index % ACCENTS.length]}22`, color: ACCENTS[index % ACCENTS.length] } : undefined}>
                {label}
              </button>
            ))}
          </div>
          <p className="mt-2 text-[11px] text-white/40">{deduplication === 'off' ? 'Giữ nguyên hình ảnh gốc, không chỉnh gì.' : `Đang chọn mức ${deduplication === 'light' ? 'nhẹ' : deduplication === 'medium' ? 'vừa' : 'mạnh'} để tạo khác biệt giữa các video.`}</p>
        </div>

        <div className="mt-5 space-y-3 border-t-2 border-dashed border-[#7B2FFF]/50 pt-4">
          <label className="flex cursor-pointer items-start gap-2.5">
            <input type="checkbox" checked={replaceVoice} onChange={event => setReplaceVoice(event.target.checked)} className="mt-0.5 size-4 accent-[#00F5D4]" />
            <span><span className="block text-xs font-black uppercase tracking-wide text-white/75">Giọng tôi chọn bạn thì dùng giọng khác thay</span><span className="mt-1 block text-[11px] leading-relaxed text-white/40">Nhà cung cấp giọng đôi khi quá tải. Bật thì hệ thống đổi sang giọng khác cùng giới tính để video vẫn ra, và ghi rõ đã đổi giọng gì.</span></span>
          </label>
          <label className="flex cursor-pointer items-start gap-2.5">
            <input type="checkbox" checked={followSubtitles} onChange={event => setFollowSubtitles(event.target.checked)} className="mt-0.5 size-4 accent-[#FF3AF2]" />
            <span><span className="block text-xs font-black uppercase tracking-wide text-white/75">Phụ đề chạy theo lời giọng đọc</span><span className="mt-1 block text-[11px] leading-relaxed text-white/40">Chỉ hiển thị video khớp với câu đang đọc. Cần có mô tả sản phẩm (có giọng đọc mới có phụ đề), mỗi video lâu thêm một giây.</span></span>
          </label>
        </div>
      </section>

      <section className={`${panelClassNames[2]} ${!replaceVoice ? 'opacity-60' : ''}`}>
        <StepTitle number={3}>AI Voice - Giọng đọc quảng cáo</StepTitle>
        <div className="mb-5 rounded-2xl border-2 border-dashed border-[#00F5D4]/50 bg-[#0D0D1A]/60 px-4 py-3 text-xs leading-relaxed text-white/60">
          <span className="font-black uppercase tracking-wide text-[#00F5D4]">Cách hoạt động:</span> Nhập mô tả sản phẩm → AI tự động viết kịch bản → Tạo giọng đọc trên máy bạn → Ghép vào video. <span className="font-bold text-[#FFE600]">Để trống nếu chỉ muốn cắt ghép video.</span>
        </div>
        {!replaceVoice && <p className="mb-4 text-xs font-bold text-[#FF6B35]">Đã tắt giọng đọc thay thế trong cài đặt video.</p>}
        <Field label="Mô tả sản phẩm (càng chi tiết càng tốt)" hint="Nhập tất cả thông tin: tên, đặc điểm, giá, khuyến mãi... AI sẽ tự viết kịch bản.">
          <textarea disabled={!replaceVoice} rows={5} value={productDescription} onChange={event => setProductDescription(event.target.value)} placeholder="VD: Dép nữ quai ngang bản to D07, chất liệu da tổng hợp cao cấp, đế chống trơn, có 2 màu đen và kem, giá 199.000 đồng, mua 2 tặng 1, phù hợp đi làm đi chơi, sản phẩm bán chạy nhất shop..." className={`${inputClassName} min-h-32 resize-y leading-relaxed disabled:cursor-not-allowed`} />
        </Field>
        <div className="mt-5 grid gap-5 sm:grid-cols-2">
          <Field label="Phong cách kịch bản">
            <select disabled={!replaceVoice} className={`${inputClassName} disabled:cursor-not-allowed`} value={scriptStyle} onChange={event => setScriptStyle(event.target.value)}>
              <option value="professional">Tiêu chuẩn — Giọng quảng cáo chuyên nghiệp</option>
              <option value="friendly">Thân thiện — Gần gũi, tự nhiên</option>
              <option value="energetic">Năng lượng — Nhanh, bắt tai</option>
              <option value="storytelling">Kể chuyện — Chậm rãi, truyền cảm</option>
            </select>
          </Field>
          <Field label="Giọng đọc">
            <div className="flex gap-2">
              <select disabled={!replaceVoice} className={`${inputClassName} min-w-0 flex-1 disabled:cursor-not-allowed`} value={voice} onChange={event => setVoice(event.target.value)}>
                {VOICES.map(option => <option key={option} value={option}>{option}</option>)}
              </select>
              <button type="button" disabled={!replaceVoice || isSpeaking} onClick={previewVoice} className="shrink-0 rounded-full border-2 border-[#FF6B35] bg-[#0D0D1A]/70 px-4 text-xs font-black uppercase tracking-wide text-[#FF6B35] transition hover:bg-[#FF6B35] hover:text-[#0D0D1A] disabled:cursor-not-allowed disabled:opacity-50">{isSpeaking ? 'Đang phát…' : '▶ Nghe thử'}</button>
            </div>
            <span className="mt-1.5 block text-[11px] text-white/40">Bấm để nghe một câu mẫu của giọng đang chọn.</span>
          </Field>
        </div>
        <div className="mt-5">
          <div className="mb-2 flex items-center justify-between text-xs font-black uppercase tracking-wide text-[#00F5D4]"><label htmlFor="speech-rate">Tốc độ đọc: {speechRate.toFixed(1)}x</label><span className="text-white/40">0.7x - 1.3x</span></div>
          <input id="speech-rate" disabled={!replaceVoice} type="range" min={0.7} max={1.3} step={0.1} value={speechRate} onChange={event => setSpeechRate(Number(event.target.value))} className="h-2 w-full cursor-pointer accent-[#00F5D4] disabled:cursor-not-allowed" />
          <div className="mt-1 flex justify-between text-[10px] text-white/40"><span>0.7 = chậm rãi</span><span>1.0 = bình thường</span><span>1.3 = nhanh</span></div>
        </div>
      </section>

      <section className={panelClassNames[3]}>
        <StepTitle number={4}>Xử lý</StepTitle>
        <div className="flex flex-wrap gap-2.5">
          <button type="button" disabled={videos.length < 2 || isProcessing} onClick={startRemix} className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-6 py-3 text-xs font-black uppercase tracking-widest text-white transition hover:scale-105 disabled:cursor-not-allowed disabled:opacity-40">
            {isProcessing ? 'Đang xử lý trên Server Go...' : 'Bắt đầu tạo video'}
          </button>
          <button type="button" onClick={() => { setVideos([]); setProductDescription(''); setNotice(''); setCurrentTask(null); setOutputVideos([]); }} className="rounded-full border-4 border-dashed border-[#00F5D4] bg-[#0D0D1A]/50 px-5 py-3 text-xs font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A]">+ Remix video mới</button>
        </div>

        {/* Live Progress Indicator */}
        {currentTask && (
          <div className="mt-5 rounded-2xl border-2 border-[#00F5D4] bg-[#0D0D1A]/80 p-4">
            <div className="flex items-center justify-between text-xs font-black uppercase tracking-wide text-[#00F5D4]">
              <span>Tiến trình xử lý: {currentTask.progress}%</span>
              <span>Trạng thái: {currentTask.status}</span>
            </div>
            <div className="mt-2 h-3 w-full overflow-hidden rounded-full bg-white/10">
              <div className="h-full bg-gradient-to-r from-[#FF3AF2] via-[#FFE600] to-[#00F5D4] transition-all duration-300" style={{ width: `${currentTask.progress}%` }} />
            </div>
            <p className="mt-2 text-xs text-white/70">{currentTask.message}</p>
          </div>
        )}

        {notice && !currentTask && (
          <p role="status" className="mt-4 rounded-2xl border-2 border-dashed border-[#00F5D4]/60 bg-[#00F5D4]/10 px-4 py-3 text-xs font-bold leading-relaxed text-[#00F5D4]">{notice}</p>
        )}

        {/* Generated Output Videos */}
        {outputVideos.length > 0 && (
          <div className="mt-6 border-t-2 border-dashed border-[#00F5D4]/40 pt-5">
            <h3 className="font-['Unbounded'] text-base font-black uppercase text-[#FFE600]">🎉 Danh sách Video đã Remix ({outputVideos.length})</h3>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              {outputVideos.map((video, idx) => (
                <div key={video.id} className="flex items-center justify-between rounded-2xl border-2 border-[#FF3AF2]/60 bg-[#0D0D1A]/90 p-3">
                  <div className="min-w-0 pr-2">
                    <p className="truncate text-xs font-black text-white">{video.title || `Video Remix #${idx + 1}`}</p>
                    <p className="text-[10px] text-white/50">{video.filename} · {formatFileSize(video.size)}</p>
                  </div>
                  <a
                    href={`${API_BASE}${video.url}`}
                    target="_blank"
                    rel="noreferrer"
                    download
                    className="shrink-0 rounded-full border-2 border-[#00F5D4] bg-[#00F5D4]/10 px-3 py-1 text-xs font-black uppercase tracking-wide text-[#00F5D4] hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
                  >
                    Tải về ⬇
                  </a>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
