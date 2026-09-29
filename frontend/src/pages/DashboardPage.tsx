import {
  ACCENTS,
  BORDER_COLORS,
  PROJECTS,
  RECENT_EXPORTS,
  STATS,
  StatusBadge,
} from "./shared"

export default function DashboardPage() {
  return (
    <div className="space-y-10">
      <div
        className="relative rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] overflow-hidden p-8 md:p-12"
        style={{ boxShadow: "12px 12px 0 #FFE600, 24px 24px 0 #FF3AF2" }}
      >
        <div
          className="pointer-events-none absolute inset-0 pattern-mesh opacity-60"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute inset-0 pattern-dots opacity-[0.06]"
          aria-hidden="true"
        />
        <div
          className="pointer-events-none absolute -bottom-8 -right-4 font-['Bangers'] text-[10rem] leading-none text-[#FF3AF2] opacity-10 select-none"
          aria-hidden="true"
        >
          MAKE ADS
        </div>
        <div className="relative z-10 flex flex-col md:flex-row md:items-center gap-6 justify-between">
          <div>
            <div className="text-[#00F5D4] text-sm font-black uppercase tracking-widest mb-2">
              👋 Welcome back, Creator
            </div>
            <h1 className="font-['Unbounded'] font-black text-4xl md:text-5xl leading-tight text-shadow-lg">
              <span className="gradient-text">Video Ads</span>
              <br />
              at Scale
            </h1>
            <p className="mt-3 text-white/70 text-lg max-w-md">
              Generate hundreds of high-converting video ads from a single
              template. 100% free, forever.
            </p>
          </div>
          <div className="flex gap-4 flex-wrap">
            <button
              className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] text-white font-black uppercase tracking-widest px-8 py-4 text-sm hover:scale-110 transition-all duration-200 animate-pulse-glow"
              style={{ backgroundSize: "200%" }}
            >
              🚀 New Batch
            </button>
            <button className="rounded-full border-4 border-dashed border-[#00F5D4] text-[#00F5D4] font-black uppercase tracking-widest px-8 py-4 text-sm hover:bg-[#00F5D4] hover:text-[#0D0D1A] hover:border-solid transition-all duration-200">
              Import CSV
            </button>
          </div>
        </div>
      </div>

      <div>
        <h2 className="font-['Unbounded'] font-black text-xl uppercase tracking-wide text-shadow-sm mb-6">
          📊 Your Stats
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
          {STATS.map((stat, i) => (
            <div
              key={stat.label}
              className={`relative rounded-3xl border-4 ${BORDER_COLORS[i % 5]} bg-[#2D1B4E] p-5 overflow-hidden transition-all duration-300 hover:scale-[1.04] hover:-rotate-1 ${
                i % 2 === 1 ? "md:translate-y-4" : ""
              }`}
              style={{
                boxShadow: `4px 4px 0 ${ACCENTS[(i + 1) % 5]}, 8px 8px 0 ${ACCENTS[(i + 2) % 5]}`,
              }}
            >
              <div
                className="pointer-events-none absolute inset-0 pattern-checker opacity-[0.06]"
                aria-hidden="true"
              />
              <div
                className="text-3xl animate-bounce-up inline-block"
                aria-hidden="true"
              >
                {stat.icon}
              </div>
              <div
                className="font-['Unbounded'] font-black text-2xl mt-2 leading-none"
                style={{
                  color: stat.accent,
                  textShadow: "2px 2px 0 rgba(0,0,0,0.4)",
                }}
              >
                {stat.value}
              </div>
              <div className="text-xs font-bold uppercase tracking-widest text-white/50 mt-0.5">
                {stat.unit}
              </div>
              <div className="font-bold text-xs text-white mt-2 leading-tight">
                {stat.label}
              </div>
              <div className="text-xs text-white/50 mt-1">{stat.delta}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div
          className="lg:col-span-2 rounded-3xl border-4 border-[#7B2FFF] bg-[#2D1B4E] overflow-hidden"
          style={{ boxShadow: "8px 8px 0 #FF3AF2" }}
        >
          <div className="border-b-4 border-dashed border-[#7B2FFF] px-6 py-4 flex items-center justify-between">
            <div className="font-['Unbounded'] font-black text-base uppercase tracking-wide text-shadow-sm">
              📁 Active Projects
            </div>
            <button className="text-[#FF3AF2] text-xs font-black uppercase tracking-widest hover:text-[#FFE600] transition-colors">
              View All →
            </button>
          </div>
          <div className="divide-y-2 divide-dashed divide-[#7B2FFF]/40">
            {PROJECTS.map((proj, i) => (
              <div
                key={proj.name}
                className="px-6 py-4 hover:bg-[#7B2FFF]/10 transition-colors duration-200 group"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="font-bold text-sm text-white group-hover:text-[#FF3AF2] transition-colors truncate">
                      {proj.name}
                    </div>
                    <div className="text-white/50 text-xs mt-1">
                      {proj.done}/{proj.videos} videos
                    </div>
                  </div>
                  <StatusBadge status={proj.status} />
                </div>
                {proj.progress > 0 && (
                  <div className="mt-3 h-2 rounded-full bg-[#0D0D1A] border border-[#7B2FFF]/40 overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-1000"
                      style={{
                        width: `${proj.progress}%`,
                        background: `linear-gradient(90deg, ${ACCENTS[i % 5]}, ${ACCENTS[(i + 1) % 5]})`,
                      }}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        <div
          className="rounded-3xl border-4 border-[#FF6B35] bg-[#2D1B4E] overflow-hidden"
          style={{ boxShadow: "8px 8px 0 #FFE600" }}
        >
          <div className="border-b-4 border-dashed border-[#FF6B35] px-6 py-4">
            <div className="font-['Unbounded'] font-black text-base uppercase tracking-wide text-shadow-sm">
              🎯 Recent Exports
            </div>
          </div>
          <div className="divide-y-2 divide-dashed divide-[#FF6B35]/30">
            {RECENT_EXPORTS.map((file) => (
              <div
                key={file.name}
                className="px-6 py-4 hover:bg-[#FF6B35]/10 transition-colors group"
              >
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl border-2 border-[#FF6B35] bg-[#FF6B35]/20 flex items-center justify-center text-base shrink-0 group-hover:scale-110 transition-transform">
                    🎬
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-xs font-bold text-white truncate">
                      {file.name}
                    </div>
                    <div className="text-white/40 text-xs">
                      {file.duration} · {file.size}
                    </div>
                  </div>
                </div>
                <div className="mt-2 flex items-center justify-between">
                  <span className="text-xs font-bold text-[#FF6B35] uppercase tracking-widest">
                    {file.platform}
                  </span>
                  <span className="text-xs text-white/40">{file.time}</span>
                </div>
              </div>
            ))}
          </div>
          <div className="px-6 pb-5 pt-3">
            <button className="w-full rounded-full border-2 border-dashed border-[#FF6B35] text-[#FF6B35] text-xs font-black uppercase tracking-widest py-2.5 hover:bg-[#FF6B35] hover:text-[#0D0D1A] hover:border-solid transition-all duration-200">
              Download All
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
