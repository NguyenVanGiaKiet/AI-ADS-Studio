import { useCallback, useEffect, useRef, useState } from 'react';
import { ACCENTS, API_BASE, apiFetch } from './shared';

type DeduplicationLevel = 'off' | 'light' | 'medium' | 'strong';

const ACTIVE_REMIX_TASK_KEY = 'ai-ads-studio:active-remix-task';
const VOICE_PREVIEW_TEXT = 'AI ADS Studio là nền tảng AI giúp tự động hóa quy trình tạo video quảng cáo chuyên nghiệp từ hình ảnh và thông tin sản phẩm, nhanh chóng, dễ dàng và tiết kiệm chi phí.';
const ACCEPTED_VIDEO_TYPES = ['mp4', 'mov', 'avi', 'mkv', 'webm'];

interface OutputVideo {
  id: string;
  taskId: string;
  title: string;
  filename: string;
  url: string;
  duration: number;
  size: number;
  script?: string;
  createdAt: string;
}

interface RemixTaskResponse {
  id: string;
  status: string;
  progress: number;
  message: string;
  script?: string;
  scripts?: string[];
  outputVideos?: OutputVideo[];
}

interface VoiceOption {
  id: string;
  name: string;
  description: string;
  gender: string;
  style: string;
}

function formatFileSize(size: number) {
  return size < 1024 * 1024
    ? `${(size / 1024).toFixed(0)} KB`
    : `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function sourceFileKey(file: File) {
  return `${file.name}-${file.size}-${file.lastModified}`;
}

function formatDuration(seconds: number) {
  const rounded = Math.max(0, Math.round(seconds));
  return `${Math.floor(rounded / 60)}:${String(rounded % 60).padStart(2, '0')}`;
}

function resolutionLabel(height: number) {
  const standardHeights = [2160, 1440, 1080, 720, 640, 480, 360, 240];
  return `${standardHeights.find(value => height >= value) ?? 240}p`;
}

function SourceVideoRow({
  file,
  index,
  onRemove,
  onDurationChange,
}: {
  file: File;
  index: number;
  onRemove: () => void;
  onDurationChange: (key: string, duration: number) => void;
}) {
  const [thumbnail, setThumbnail] = useState('');
  const [videoDuration, setVideoDuration] = useState<number | null>(null);
  const [resolution, setResolution] = useState('');

  useEffect(() => {
    const video = document.createElement('video');
    const objectUrl = URL.createObjectURL(file);
    let active = true;

    const captureThumbnail = () => {
      if (!active || !video.videoWidth || !video.videoHeight) return;
      const canvas = document.createElement('canvas');
      const scale = Math.min(96 / video.videoWidth, 96 / video.videoHeight, 1);
      canvas.width = Math.max(1, Math.round(video.videoWidth * scale));
      canvas.height = Math.max(1, Math.round(video.videoHeight * scale));
      const context = canvas.getContext('2d');
      if (!context) return;
      context.drawImage(video, 0, 0, canvas.width, canvas.height);
      setThumbnail(canvas.toDataURL('image/jpeg', 0.75));
    };

    const handleMetadata = () => {
      if (!active) return;
      if (Number.isFinite(video.duration)) {
        setVideoDuration(video.duration);
        onDurationChange(sourceFileKey(file), video.duration);
      }
      if (video.videoHeight > 0) setResolution(resolutionLabel(video.videoHeight));
      if (video.duration > 0) video.currentTime = Math.min(0.1, video.duration / 2);
    };

    const handleError = () => {
      if (active) setResolution('—');
    };

    video.muted = true;
    video.preload = 'metadata';
    video.addEventListener('loadedmetadata', handleMetadata);
    video.addEventListener('seeked', captureThumbnail);
    video.addEventListener('loadeddata', captureThumbnail);
    video.addEventListener('error', handleError);
    video.src = objectUrl;
    video.load();

    return () => {
      active = false;
      video.pause();
      video.removeAttribute('src');
      video.load();
      video.removeEventListener('loadedmetadata', handleMetadata);
      video.removeEventListener('seeked', captureThumbnail);
      video.removeEventListener('loadeddata', captureThumbnail);
      video.removeEventListener('error', handleError);
      URL.revokeObjectURL(objectUrl);
    };
  }, [file, onDurationChange]);

  return (
    <div className="flex min-h-[78px] min-w-0 items-center gap-3 rounded-xl border border-[#142338] bg-[#080B14] px-3 py-2">
      <div className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-white/10 bg-[#111827]">
        {thumbnail ? (
          <img src={thumbnail} alt="" className="size-full object-contain" />
        ) : (
          <span className="text-lg" aria-hidden="true">🎞️</span>
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs font-bold text-white/85">{file.name}</p>
        <p className="mt-1 text-[10px] text-white/40">
          Video {index + 1}{videoDuration === null ? '' : ` · ${formatDuration(videoDuration)}`}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-[11px] text-white/50">{formatFileSize(file.size)}</span>
        <span className="rounded bg-[#392D10] px-1.5 py-0.5 text-[10px] font-black text-[#FFE600]">
          {resolution || '…'}
        </span>
        <button
          type="button"
          aria-label={`Xóa ${file.name}`}
          onClick={onRemove}
          className="rounded px-1.5 py-1 text-sm font-bold text-white/35 hover:bg-[#FF3AF2]/15 hover:text-[#FF3AF2]"
        >
          ×
        </button>
      </div>
    </div>
  );
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
  const [videoDurations, setVideoDurations] = useState<Record<string, number>>({});
  const [isDragging, setIsDragging] = useState(false);
  const [outputCount, setOutputCount] = useState(5);
  const [duration, setDuration] = useState(30);
  const [cutSensitivity, setCutSensitivity] = useState('medium');
  const [remixMode, setRemixMode] = useState('standard');
  const [deduplication, setDeduplication] = useState<DeduplicationLevel>('off');
  const [replaceVoice, setReplaceVoice] = useState(true);
  const [followSubtitles, setFollowSubtitles] = useState(false);
  const [subtitlePosition, setSubtitlePosition] = useState('bottom');
  const [subtitleStyle, setSubtitleStyle] = useState('white_yellow');
  const [productDescription, setProductDescription] = useState('');
  const [scriptStyle, setScriptStyle] = useState('professional');
  const [voice, setVoice] = useState('');
  const [voices, setVoices] = useState<VoiceOption[]>([]);
  const [speechRate, setSpeechRate] = useState(1);
  const [notice, setNotice] = useState('');
  const [isSpeaking, setIsSpeaking] = useState(false);
  const previewAudioRef = useRef<HTMLAudioElement | null>(null);
  const previewRequestRef = useRef<AbortController | null>(null);

  // Backend Task State
  const [currentTask, setCurrentTask] = useState<RemixTaskResponse | null>(null);
  const [outputVideos, setOutputVideos] = useState<OutputVideo[]>([]);
  const [outputTaskId, setOutputTaskId] = useState<string | null>(null);
  const [activeTaskId, setActiveTaskId] = useState<string | null>(() => window.localStorage.getItem(ACTIVE_REMIX_TASK_KEY));
  const [isProcessing, setIsProcessing] = useState(() => Boolean(window.localStorage.getItem(ACTIVE_REMIX_TASK_KEY)));
  const handleDurationChange = useCallback((key: string, videoDuration: number) => {
    setVideoDurations(current => current[key] === videoDuration
      ? current
      : { ...current, [key]: videoDuration });
  }, []);
  const totalSourceDuration = videos.reduce(
    (total, file) => total + (videoDurations[sourceFileKey(file)] ?? 0),
    0,
  );
  const totalSourceSize = videos.reduce((total, file) => total + file.size, 0);

  useEffect(() => {
    let isActive = true;
    apiFetch('/api/tts/voices')
      .then(async response => {
        if (!response.ok) throw new Error((await response.text()) || `HTTP ${response.status}`);
        return response.json();
      })
      .then(result => {
        if (!isActive || !Array.isArray(result.data)) return;
        setVoices(result.data);
        setVoice(result.data[0]?.id ?? '');
        if (result.data.length === 0) setNotice('Chưa có giọng đọc tiếng Việt miễn phí khả dụng.');
      })
      .catch(error => {
        if (isActive) setNotice(`Không tải được danh sách giọng đọc: ${error instanceof Error ? error.message : 'Lỗi kết nối backend'}`);
      });
    return () => {
      isActive = false;
      previewRequestRef.current?.abort();
      previewAudioRef.current?.pause();
    };
  }, []);

  useEffect(() => {
    if (!activeTaskId) return;

    let isActive = true;
    let timeoutId: number | undefined;

    const stopTracking = () => {
      window.localStorage.removeItem(ACTIVE_REMIX_TASK_KEY);
      setActiveTaskId(null);
      setIsProcessing(false);
    };

    const pollTask = async () => {
      try {
        const response = await apiFetch(`/api/remix/tasks/${encodeURIComponent(activeTaskId)}`);
        if (response.status === 404) {
          const outputsResponse = await apiFetch('/api/videos');
          if (!outputsResponse.ok) throw new Error(`Không tải được video đã tạo (HTTP ${outputsResponse.status})`);
          const outputsJson = await outputsResponse.json();
          const restoredOutputs: OutputVideo[] = Array.isArray(outputsJson.data)
            ? outputsJson.data.filter((video: OutputVideo) => video.taskId === activeTaskId)
            : [];

          if (isActive && restoredOutputs.length > 0) {
            setOutputVideos(restoredOutputs);
            setOutputTaskId(activeTaskId);
            setNotice('Tiến trình đã hoàn tất trước đó. Đã khôi phục video đầu ra từ thư viện.');
          } else if (isActive) {
            setNotice('Không tìm thấy tiến trình đang chạy trên backend. Hãy kiểm tra thư viện video đã tạo.');
          }
          if (isActive) stopTracking();
          return;
        }
        if (!response.ok) throw new Error(`Không đọc được tiến trình (HTTP ${response.status})`);

        const result = await response.json();
        const task: RemixTaskResponse | undefined = result.data;
        if (!task?.id || task.id !== activeTaskId) throw new Error('Backend trả về trạng thái tiến trình không hợp lệ.');
        if (!isActive) return;

        setCurrentTask(task);
        setNotice(task.message);
        if (task.status === 'completed') {
          setOutputVideos(task.outputVideos ?? []);
          setOutputTaskId(task.id);
          stopTracking();
          return;
        }
        if (task.status === 'failed') {
          stopTracking();
          return;
        }
      } catch (error) {
        if (isActive) {
          setNotice(`Đang chờ kết nối lại để theo dõi tiến trình: ${error instanceof Error ? error.message : 'Lỗi kết nối backend'}`);
        }
      }

      if (isActive) timeoutId = window.setTimeout(pollTask, 1200);
    };

    void pollTask();
    return () => {
      isActive = false;
      if (timeoutId !== undefined) window.clearTimeout(timeoutId);
    };
  }, [activeTaskId]);

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
    if (isSpeaking) {
      previewRequestRef.current?.abort();
      previewRequestRef.current = null;
      previewAudioRef.current?.pause();
      if (previewAudioRef.current) previewAudioRef.current.currentTime = 0;
      previewAudioRef.current = null;
      setIsSpeaking(false);
      setNotice('Đã dừng nghe thử giọng đọc.');
      return;
    }

    const controller = new AbortController();
    previewRequestRef.current = controller;
    setIsSpeaking(true);
    try {
      const res = await apiFetch('/api/tts/preview', {
        method: 'POST',
        signal: controller.signal,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          text: VOICE_PREVIEW_TEXT,
          style: scriptStyle,
          voice,
          rate: speechRate,
          duration,
        }),
      });
      if (!res.ok) {
        throw new Error((await res.text()) || `HTTP ${res.status}`);
      }
      const result = await res.json();
      if (controller.signal.aborted) return;
      if (!result.data?.audioUrl) throw new Error('Backend không trả về audio preview.');
      previewAudioRef.current?.pause();
      const audio = new Audio();
      audio.crossOrigin = 'use-credentials';
      audio.src = `${API_BASE}${result.data.audioUrl}`;
      previewAudioRef.current = audio;
      audio.onended = () => {
        if (previewAudioRef.current === audio) {
          previewAudioRef.current = null;
          previewRequestRef.current = null;
          setIsSpeaking(false);
        }
      };
      audio.onerror = () => {
        if (previewAudioRef.current === audio) {
          previewAudioRef.current = null;
          previewRequestRef.current = null;
          setIsSpeaking(false);
          setNotice('Không phát được audio preview do Piper tạo.');
        }
      };
      await audio.play();
      if (controller.signal.aborted) {
        audio.pause();
        return;
      }
      setNotice('Đang phát giọng đọc Piper tiếng Việt chạy cục bộ trên backend.');
    } catch (error) {
      if (!controller.signal.aborted) {
        setIsSpeaking(false);
        setNotice(`Không thể tạo giọng đọc thử: ${error instanceof Error ? error.message : 'Lỗi không xác định'}`);
      }
    } finally {
      if (previewRequestRef.current === controller && !previewAudioRef.current) {
        previewRequestRef.current = null;
        setIsSpeaking(false);
      }
    }
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
    setOutputTaskId(null);

    try {
      // Step 1: Upload Videos
      const formData = new FormData();
      videos.forEach(file => formData.append('files', file));

      const uploadRes = await apiFetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      let uploadedVideoIds: string[] = [];
      if (uploadRes.ok) {
        const uploadData = await uploadRes.json();
        uploadedVideoIds = uploadData.data ? uploadData.data.map((v: { id: string }) => v.id) : [];
      }

      // Step 2: Create Remix Task
      const taskRes = await apiFetch('/api/remix/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          videoIds: uploadedVideoIds,
          outputCount,
          duration,
          cutSensitivity,
          remixMode,
          deduplication,
          replaceVoice,
          followSubtitles,
          subtitlePosition,
          subtitleStyle,
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
      const task: RemixTaskResponse | undefined = taskJson.data;
      if (!task?.id) throw new Error('Backend không trả về mã tiến trình remix.');
      window.localStorage.setItem(ACTIVE_REMIX_TASK_KEY, task.id);
      setCurrentTask(task);
      setActiveTaskId(task.id);

    } catch (error) {
      setIsProcessing(false);
      setNotice(`Lỗi: ${error instanceof Error ? error.message : 'Chưa thể kết nối tới Go Backend'}`);
    }
  }

  return (
    <div className="space-y-8 pb-8 text-white">
      <div>
        <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#00F5D4]">
          🎬 Video Editor
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
          <div className="mt-4">
            <p className="mb-2 text-xs font-bold text-white/75">
              {videos.length} video
              <span className="font-normal text-white/40">
                {' '}· {Math.round(totalSourceDuration)} giây phim gốc · {formatFileSize(totalSourceSize)}
              </span>
            </p>
            <div className="max-h-64 space-y-1.5 overflow-y-auto rounded-xl pr-1">
            {videos.map((video, index) => (
              <SourceVideoRow
                key={sourceFileKey(video)}
                file={video}
                index={index}
                onRemove={() => setVideos(current => current.filter((_, fileIndex) => fileIndex !== index))}
                onDurationChange={handleDurationChange}
              />
            ))}
            </div>
            <div className="mt-2">
            <button type="button" onClick={() => fileInputRef.current?.click()} className="text-xs font-black uppercase tracking-widest text-[#00F5D4] hover:text-[#FFE600]">+ Thêm video</button>
            </div>
          </div>
        )}
      </section>

      <section className={panelClassNames[1]}>
        <StepTitle number={2}>Cài đặt video</StepTitle>
        <div className="grid grid-cols-1 gap-x-4 gap-y-5 sm:grid-cols-2 xl:grid-cols-4">
          <Field label="Số lượng video đầu ra" hint="Số video mới cần tạo (1-50)">
            <input className={inputClassName} type="number" min={1} max={50} value={outputCount} onChange={event => setOutputCount(Math.min(50, Math.max(1, Number(event.target.value) || 1)))} />
          </Field>
          <Field label="Thời lượng tối đa (giây)" hint="Ghép clip tuần tự; nếu tổng nguồn ngắn hơn, video giữ nguyên thời lượng thực.">
            <input
              className={inputClassName}
              type="number"
              min={10}
              max={60}
              step={1}
              value={duration}
              onChange={event => {
                const value = event.currentTarget.valueAsNumber;
                if (Number.isFinite(value)) setDuration(Math.min(60, value));
              }}
              onBlur={event => {
                const value = event.currentTarget.valueAsNumber;
                if (Number.isFinite(value)) setDuration(Math.min(60, Math.max(10, Math.round(value))));
                else setDuration(current => Math.min(60, Math.max(10, current)));
              }}
            />
          </Field>
          <Field label="Độ nhạy cắt cảnh" hint="Tốc độ chuyển cảnh của video">
            <select className={inputClassName} value={cutSensitivity} onChange={event => setCutSensitivity(event.target.value)}>
              <option value="high">Cao — cảnh ngắn 2,5-5 giây, nhịp nhanh</option>
              <option value="medium">Trung bình(khuyên dùng) — máy tự chọn</option>
              <option value="low">Thấp — cảnh dài 8-14 giây, mượt</option>
            </select>
          </Field>
          <Field label="Chế độ Remix">
            <select className={inputClassName} value={remixMode} onChange={event => setRemixMode(event.target.value)}>
              <option value="standard">Tiêu chuẩn — Cắt ghép bình thường</option>
              <option value="exclude_faces">Thông minh — Cắt bỏ cảnh có mặt người</option>
              <option value="product_zoom">Thông minh — Zoom cận sản phẩm, bỏ phần mặt</option>
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
          <p className="mt-1 text-[11px] text-white/40">Backend so sánh khung hình với video đã tạo và thử tối đa 5 biến thể. Nếu vẫn tương tự, hệ thống chọn bản khác biệt nhất và cảnh báo.</p>
        </div>

        <div className="mt-5 space-y-3 border-t-2 border-dashed border-[#7B2FFF]/50 pt-4">
          <label className="flex cursor-pointer items-start gap-2.5">
            <input type="checkbox" checked={replaceVoice} onChange={event => { const enabled = event.target.checked; setReplaceVoice(enabled); if (!enabled) setFollowSubtitles(false); }} className="mt-0.5 size-4 accent-[#00F5D4]" />
            <span><span className="block text-xs font-black uppercase tracking-wide text-white/75">Thay âm thanh video bằng giọng đọc quảng cáo</span><span className="mt-1 block text-[11px] leading-relaxed text-white/40">Piper tạo giọng tiếng Việt trực tiếp trong backend rồi FFmpeg ghép vào video.</span></span>
          </label>
          <label className="flex cursor-pointer items-start gap-2.5">
            <input type="checkbox" checked={followSubtitles} disabled={!replaceVoice} onChange={event => setFollowSubtitles(event.target.checked)} className="mt-0.5 size-4 accent-[#FF3AF2] disabled:cursor-not-allowed disabled:opacity-50" />
            <span><span className="block text-xs font-black uppercase tracking-wide text-white/75">Phụ đề chạy theo lời giọng đọc</span><span className="mt-1 block text-[11px] leading-relaxed text-white/40">Ước lượng thời điểm từng tiếng từ kịch bản rồi tô sáng theo giọng đọc. Cần bật giọng đọc thay thế.</span></span>
          </label>
          {followSubtitles && replaceVoice && (
            <div className="grid gap-3 rounded-2xl border-2 border-[#FF3AF2]/40 bg-[#0D0D1A]/50 p-3 sm:grid-cols-2">
              <Field label="Vị trí phụ đề">
                <select className={inputClassName} value={subtitlePosition} onChange={event => setSubtitlePosition(event.target.value)}>
                  <option value="bottom">Chữ phía dưới (trên giỏ hàng)</option>
                  <option value="top">Chữ phía trên (dưới ô tìm kiếm)</option>
                </select>
              </Field>
              <Field label="Kiểu màu phụ đề">
                <select className={inputClassName} value={subtitleStyle} onChange={event => setSubtitleStyle(event.target.value)}>
                  <option value="white_yellow">Trắng - vàng (đang đọc sáng vàng)</option>
                  <option value="white_gray">Trắng - xám (chưa đọc màu xám)</option>
                </select>
              </Field>
            </div>
          )}
        </div>
      </section>

      <section className={`${panelClassNames[2]} ${!replaceVoice ? 'opacity-60' : ''}`}>
        <StepTitle number={3}>AI Voice - Giọng đọc quảng cáo</StepTitle>
        <div className="mb-5 rounded-2xl border-2 border-dashed border-[#00F5D4]/50 bg-[#0D0D1A]/60 px-4 py-3 text-xs leading-relaxed text-white/60">
          <span className="font-black uppercase tracking-wide text-[#00F5D4]">Cách hoạt động:</span> Groq viết kịch bản → Piper đọc tiếng Việt bằng model cục bộ → backend tự căn thời lượng lời đọc theo video → FFmpeg ghép vào video. Cần GROQ_API_KEY để viết kịch bản; TTS không cần API key hay kết nối dịch vụ bên ngoài. <span className="font-bold text-[#FFE600]">Để trống nếu chỉ muốn cắt ghép video.</span>
        </div>
        {!replaceVoice && <p className="mb-4 text-xs font-bold text-[#FF6B35]">Đã tắt giọng đọc thay thế trong cài đặt video.</p>}
        <Field label="Mô tả sản phẩm (càng chi tiết càng tốt)" hint="Groq dùng mô tả này để viết kịch bản; chỉ nêu thông tin bạn cung cấp.">
          <textarea disabled={!replaceVoice} rows={5} value={productDescription} onChange={event => setProductDescription(event.target.value)} placeholder="VD: Dép nữ quai ngang bản to D07, chất liệu da tổng hợp cao cấp, đế chống trơn, có 2 màu đen và kem, giá 199.000 đồng, mua 2 tặng 1, phù hợp đi làm đi chơi, sản phẩm bán chạy nhất shop..." className={`${inputClassName} min-h-32 resize-y leading-relaxed disabled:cursor-not-allowed`} />
        </Field>
        <div className="mt-5 grid gap-5 sm:grid-cols-2">
          <Field label="Phong cách kịch bản">
            <select disabled={!replaceVoice} className={`${inputClassName} disabled:cursor-not-allowed`} value={scriptStyle} onChange={event => setScriptStyle(event.target.value)}>
              <option value="professional">Tiêu chuẩn — Giọng quảng cáo chuyên nghiệp</option>
              <option value="adam_drama">Phong cách Adam — Vui nhộn, drama, xưng anh/chồng</option>
              <option value="adam_viral">Adam Viral — Adam + cảm thán mạnh (Trời má, Đậu xanh...)</option>
              <option value="dan_da">Dân Dã Gần Gũi — Giọng miền quê chân chất, mộc mạc</option>
            </select>
          </Field>
          <Field label="Giọng đọc">
            <div className="flex gap-2">
              <select disabled={!replaceVoice || voices.length === 0} className={`${inputClassName} min-w-0 flex-1 disabled:cursor-not-allowed`} value={voice} onChange={event => setVoice(event.target.value)}>
                {voices.length === 0 && <option value="">Chưa tải được giọng đọc tiếng Việt</option>}
                {voices.map(option => (
                  <option key={option.id} value={option.id}>{option.name}{option.gender ? ` — ${option.gender}` : ''}</option>
                ))}
              </select>
              <button type="button" disabled={!replaceVoice} onClick={previewVoice} aria-pressed={isSpeaking} className="shrink-0 rounded-full border-2 border-[#FF6B35] bg-[#0D0D1A]/70 px-4 text-xs font-black uppercase tracking-wide text-[#FF6B35] transition hover:bg-[#FF6B35] hover:text-[#0D0D1A] disabled:cursor-not-allowed disabled:opacity-50">{isSpeaking ? '■ Dừng' : '▶ Nghe thử'}</button>
            </div>
            <span className="mt-1.5 block text-[11px] text-white/40">Các voice chạy cục bộ, không tốn phí API. Bấm để nghe thử voice đang chọn.</span>
          </Field>
        </div>
        <p className="mt-3 text-[11px] font-bold text-[#FFE600]">
          Kịch bản được ước lượng độ dài theo video {duration} giây ở mức {speechRate.toFixed(1)}x; audio sẽ được căn lại sau khi tạo.
        </p>
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
          <button type="button" disabled={isProcessing} onClick={() => { setVideos([]); setProductDescription(''); setNotice(''); setCurrentTask(null); setOutputVideos([]); setOutputTaskId(null); }} className="rounded-full border-4 border-dashed border-[#00F5D4] bg-[#0D0D1A]/50 px-5 py-3 text-xs font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A] disabled:cursor-not-allowed disabled:opacity-40">+ Remix video mới</button>
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
            {(currentTask.scripts?.length || currentTask.script) && (
              <div className="mt-3 border-t border-dashed border-[#00F5D4]/30 pt-3">
                <p className="text-[10px] font-black uppercase tracking-widest text-[#FFE600]">Kịch bản Groq ({currentTask.scripts?.length ?? 1})</p>
                <div className="mt-2 space-y-2">
                  {(currentTask.scripts ?? [currentTask.script!]).map((script, index) => (
                    <details key={`${index}-${script.slice(0, 20)}`} className="rounded-xl border border-white/10 bg-white/[0.03] p-2">
                      <summary className="cursor-pointer text-[11px] font-bold text-[#00F5D4]">Video {index + 1} — kịch bản riêng</summary>
                      <p className="mt-2 text-xs leading-relaxed text-white/80">{script}</p>
                    </details>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {notice && !currentTask && (
          <p role="status" className="mt-4 rounded-2xl border-2 border-dashed border-[#00F5D4]/60 bg-[#00F5D4]/10 px-4 py-3 text-xs font-bold leading-relaxed text-[#00F5D4]">{notice}</p>
        )}

        {/* Generated Output Videos */}
        {outputVideos.length > 0 && (
          <div className="mt-6 border-t-2 border-dashed border-[#00F5D4]/40 pt-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="font-['Unbounded'] text-base font-black uppercase text-[#FFE600]">🎉 Video đã tạo ({outputVideos.length})</h3>
                <p className="mt-1 text-xs text-white/50">Xem trước, tải riêng từng video hoặc tải toàn bộ dưới dạng ZIP.</p>
              </div>
              {outputTaskId && (
                <a
                  href={`${API_BASE}/api/remix/tasks/${encodeURIComponent(outputTaskId)}/download`}
                  download
                  className="rounded-full border-2 border-[#FFE600] bg-[#FFE600]/10 px-4 py-2 text-xs font-black uppercase tracking-widest text-[#FFE600] transition hover:bg-[#FFE600] hover:text-[#0D0D1A]"
                >
                  Tải tất cả (.zip) ⬇
                </a>
              )}
            </div>
            <div className="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {outputVideos.map((video, idx) => (
                <article key={video.id} className="overflow-hidden rounded-2xl border-2 border-[#FF3AF2]/60 bg-[#0D0D1A]/90">
                  <video
                    src={`${API_BASE}${video.url}`}
                    crossOrigin="use-credentials"
                    controls
                    preload="metadata"
                    className="aspect-[9/16] max-h-[420px] w-full bg-black object-contain"
                    aria-label={`Xem trước ${video.title || video.filename}`}
                  />
                  <div className="p-3">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-xs font-black text-white">{video.title || `Video Remix #${idx + 1}`}</p>
                        <p className="mt-1 truncate text-[10px] text-white/50">{video.filename}</p>
                      </div>
                      <span className="shrink-0 rounded-full border border-[#00F5D4]/60 bg-[#00F5D4]/10 px-2 py-1 text-[9px] font-black uppercase text-[#00F5D4]">Đã tạo</span>
                    </div>
                    <div className="mt-3 flex items-center justify-between text-[10px] font-bold text-white/50">
                      <span>⏱ {video.duration}s</span>
                      <span>💾 {formatFileSize(video.size)}</span>
                    </div>
                    {video.script && (
                      <details className="mt-3 rounded-xl border border-white/10 bg-white/[0.03] p-2">
                        <summary className="cursor-pointer text-[10px] font-bold text-[#FFE600]">Xem kịch bản riêng của video</summary>
                        <p className="mt-2 text-[11px] leading-relaxed text-white/75">{video.script}</p>
                      </details>
                    )}
                    <a
                      href={`${API_BASE}/api/videos/${encodeURIComponent(video.id)}/download`}
                      download={video.filename}
                      className="mt-3 block rounded-xl border-2 border-[#00F5D4] bg-[#00F5D4]/10 px-3 py-2 text-center text-[10px] font-black uppercase tracking-widest text-[#00F5D4] transition hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
                    >
                      Tải video ⬇
                    </a>
                  </div>
                </article>
              ))}
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
