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

export const STATS = [
  {
    label: "Videos Created",
    value: "1,247",
    unit: "total",
    icon: "🎬",
    delta: "+128 this week",
    accent: ACCENTS[0],
  },

  {
    label: "Templates Used",
    value: "89",
    unit: "active",
    icon: "✨",
    delta: "+12 new added",
    accent: ACCENTS[1],
  },

  {
    label: "Credits Left",
    value: "∞",
    unit: "FREE",
    icon: "💫",
    delta: "Unlimited forever",
    accent: ACCENTS[2],
  },

  {
    label: "Export Ready",
    value: "34",
    unit: "queued",
    icon: "🚀",
    delta: "5 processing now",
    accent: ACCENTS[3],
  },

  {
    label: "Avg Render Time",
    value: "1.2s",
    unit: "per video",
    icon: "⚡",
    delta: "-0.3s vs last week",
    accent: ACCENTS[4],
  },
]

export const PROJECTS = [
  {
    name: "Summer Sale Campaign",
    status: "processing",
    progress: 72,
    videos: 120,
    done: 86,
    accent: ACCENTS[0],
  },

  {
    name: "Product Launch Series",
    status: "done",
    progress: 100,
    videos: 50,
    done: 50,
    accent: ACCENTS[1],
  },

  {
    name: "Flash Deal Reels 08/24",
    status: "queued",
    progress: 0,
    videos: 200,
    done: 0,
    accent: ACCENTS[2],
  },

  {
    name: "Brand Story Montage",
    status: "processing",
    progress: 38,
    videos: 30,
    done: 11,
    accent: ACCENTS[3],
  },

  {
    name: "Holiday Mega Bundle",
    status: "draft",
    progress: 0,
    videos: 500,
    done: 0,
    accent: ACCENTS[4],
  },
]

export const TEMPLATES = [
  {
    name: "TikTok Burst",
    ratio: "9:16",
    category: "Social",
    uses: 4231,
    hot: true,
    accent: ACCENTS[0],
  },

  {
    name: "Reels Banger",
    ratio: "9:16",
    category: "Instagram",
    uses: 3872,
    hot: true,
    accent: ACCENTS[1],
  },

  {
    name: "YouTube Pre-roll",
    ratio: "16:9",
    category: "YouTube",
    uses: 1924,
    hot: false,
    accent: ACCENTS[2],
  },

  {
    name: "Square Pop",
    ratio: "1:1",
    category: "Facebook",
    uses: 2201,
    hot: false,
    accent: ACCENTS[3],
  },

  {
    name: "Story Swipe",
    ratio: "9:16",
    category: "Stories",
    uses: 5110,
    hot: true,
    accent: ACCENTS[4],
  },

  {
    name: "Product Zoom",
    ratio: "1:1",
    category: "E-comm",
    uses: 987,
    hot: false,
    accent: ACCENTS[0],
  },
]

export const RECENT_EXPORTS = [
  {
    name: "sale_promo_v3.mp4",
    size: "2.1 MB",
    duration: "0:15",
    time: "2 min ago",
    platform: "TikTok",
  },

  {
    name: "launch_teaser.mp4",
    size: "4.7 MB",
    duration: "0:30",
    time: "14 min ago",
    platform: "YouTube",
  },

  {
    name: "flash_deal_001.mp4",
    size: "1.3 MB",
    duration: "0:08",
    time: "1h ago",
    platform: "Reels",
  },

  {
    name: "brand_story_final.mp4",
    size: "8.2 MB",
    duration: "1:00",
    time: "3h ago",
    platform: "YouTube",
  },
]

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; text: string; label: string }> = {
    processing: {
      bg: "bg-[#FF6B35]/20",
      text: "text-[#FF6B35]",
      label: "⚡ Processing",
    },

    done: { bg: "bg-[#00F5D4]/20", text: "text-[#00F5D4]", label: "✓ Done" },

    queued: {
      bg: "bg-[#7B2FFF]/20",
      text: "text-[#7B2FFF]",
      label: "⏳ Queued",
    },

    draft: { bg: "bg-[#FF3AF2]/20", text: "text-[#FF3AF2]", label: "✏ Draft" },
  }

  const s = map[status] ?? map.draft

  return (
    <span
      className={`${s.bg} ${s.text} text-xs font-bold uppercase tracking-widest px-3 py-1 rounded-full border border-current`}
    >
      {s.label}
    </span>
  )
}
