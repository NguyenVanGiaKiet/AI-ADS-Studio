import { ACCENTS } from "./shared"

const features = [
  {
    title: "Chế độ dựng",
    icon: "🎞️",
    description:
      "Tiêu chuẩn cắt ghép bình thường; loại cảnh có mặt người bằng nhận diện cục bộ; hoặc zoom vùng dưới-trung tâm để hạn chế lộ mặt.",
    note: "Zoom sản phẩm là heuristic, không tự nhận diện được sản phẩm.",
  },
  {
    title: "Lời thoại và giọng đọc",
    icon: "🎙️",
    description:
      "Nhập thông tin sản phẩm để Groq viết lời quảng cáo; Piper tạo giọng tiếng Việt cục bộ. Có thể chọn VAI 1000, 25Hours hoặc một trong năm giọng Piper v3.",
    note: "Tạo lời thoại cần cấu hình GROQ_API_KEY ở backend; TTS chạy cục bộ.",
  },
  {
    title: "Phụ đề và thời lượng",
    icon: "💬",
    description:
      "Bật phụ đề karaoke theo lời thoại, chọn vị trí và kiểu hiển thị. Điều chỉnh tốc độ đọc trong khoảng 0,7x–1,3x và thời lượng đầu ra.",
    note: "Cần bật thay giọng đọc để dùng phụ đề lời thoại.",
  },
  {
    title: "Đầu ra và tải video",
    icon: "📦",
    description:
      "Một tác vụ có thể tạo nhiều phiên bản từ video nguồn. Kiểm tra tiến trình ở Lịch sử tác vụ và xem hoặc tải file trong Video đã tạo.",
    note: "Ứng dụng chưa có template dựng sẵn hay số liệu hiệu quả mạng xã hội.",
  },
]

export default function TemplatesPage({
  onNavigateToCreate,
}: {
  onNavigateToCreate: () => void
}) {
  return (
    <div className="space-y-8 text-white">
      <header>
        <div className="mb-1 text-sm font-black uppercase tracking-widest text-[#FFE600]">
          ✨ Hướng dẫn sử dụng
        </div>
        <h1 className="font-['Unbounded'] text-3xl font-black md:text-4xl">
          Tạo video với <span className="gradient-text">AI ADS Studio</span>
        </h1>
        <p className="mt-3 max-w-3xl text-sm leading-relaxed text-white/60">
          Đây là hướng dẫn theo đúng các chức năng đang có trong ứng dụng, không
          phải thư viện template hoặc dữ liệu mẫu.
        </p>
      </header>

      <ol className="grid gap-5 md:grid-cols-2">
        {features.map((feature, index) => (
          <li
            key={feature.title}
            className="rounded-3xl border-4 bg-[#2D1B4E] p-6"
            style={{
              borderColor: ACCENTS[index],
              boxShadow: `7px 7px 0 ${ACCENTS[(index + 1) % ACCENTS.length]}`,
            }}
          >
            <div className="flex items-center gap-3">
              <span className="text-3xl" aria-hidden="true">{feature.icon}</span>
              <h2 className="font-['Unbounded'] text-base font-black" style={{ color: ACCENTS[index] }}>
                {index + 1}. {feature.title}
              </h2>
            </div>
            <p className="mt-4 text-sm leading-relaxed text-white/75">
              {feature.description}
            </p>
            <p className="mt-3 border-t border-dashed border-white/15 pt-3 text-xs leading-relaxed text-white/45">
              {feature.note}
            </p>
          </li>
        ))}
      </ol>

      <button
        type="button"
        onClick={onNavigateToCreate}
        className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-7 py-3 text-xs font-black uppercase tracking-widest text-white transition hover:scale-105"
      >
        Bắt đầu remix video
      </button>
    </div>
  )
}
