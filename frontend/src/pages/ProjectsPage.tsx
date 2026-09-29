import { ACCENTS, BORDER_COLORS, PROJECTS, StatusBadge } from "./shared"

export default function ProjectsPage() {
  return (
    <div className="space-y-8">
      <div className="flex items-end justify-between gap-4 flex-wrap">
        <div>
          <div className="text-[#FF6B35] text-sm font-black uppercase tracking-widest mb-1">
            📁 All Work
          </div>
          <h1 className="font-['Unbounded'] font-black text-3xl text-shadow-lg">
            <span className="gradient-text">Projects</span>
          </h1>
        </div>
        <button className="rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] to-[#7B2FFF] text-white font-black uppercase tracking-widest px-7 py-3 text-sm hover:scale-105 transition-all duration-200 animate-pulse-glow">
          + New Project
        </button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {PROJECTS.map((proj, i) => (
          <div
            key={proj.name}
            className={`relative rounded-3xl border-4 ${BORDER_COLORS[i % 5]} bg-[#2D1B4E] p-6 overflow-hidden transition-all duration-300 hover:scale-[1.02] hover:rotate-1 ${
              i % 2 === 1 ? "md:translate-y-4" : ""
            }`}
            style={{
              boxShadow: `8px 8px 0 ${ACCENTS[(i + 1) % 5]}, 16px 16px 0 ${ACCENTS[(i + 2) % 5]}`,
            }}
          >
            <div
              className="pointer-events-none absolute inset-0 pattern-checker opacity-[0.05]"
              aria-hidden="true"
            />
            <div className="relative z-10">
              <div className="flex items-start justify-between gap-3 mb-4">
                <div>
                  <div
                    className="font-['Unbounded'] font-black text-sm leading-tight"
                    style={{
                      color: ACCENTS[i % 5],
                      textShadow: "1px 1px 0 #0D0D1A",
                    }}
                  >
                    {proj.name}
                  </div>
                  <div className="text-white/50 text-xs mt-1">
                    {proj.done} of {proj.videos} videos
                  </div>
                </div>
                <StatusBadge status={proj.status} />
              </div>
              <div className="h-3 rounded-full bg-[#0D0D1A] border border-white/10 overflow-hidden mb-2">
                <div
                  className="h-full rounded-full transition-all duration-1000"
                  style={{
                    width: `${proj.progress}%`,
                    background: `linear-gradient(90deg, ${ACCENTS[i % 5]}, ${ACCENTS[(i + 1) % 5]})`,
                    boxShadow:
                      proj.progress > 0
                        ? `0 0 8px ${ACCENTS[i % 5]}88`
                        : "none",
                    minWidth: proj.progress > 0 ? "8px" : "0",
                  }}
                />
              </div>
              <div className="text-xs text-white/40 font-bold text-right">
                {proj.progress}%
              </div>
              <div className="flex gap-3 mt-4">
                <button
                  className="flex-1 rounded-full border-2 text-xs font-black uppercase tracking-widest py-2 transition-all duration-200 hover:scale-105"
                  style={{ borderColor: ACCENTS[i % 5], color: ACCENTS[i % 5] }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = ACCENTS[i % 5]
                    e.currentTarget.style.color = "#0D0D1A"
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = "transparent"
                    e.currentTarget.style.color = ACCENTS[i % 5]
                  }}
                >
                  Open
                </button>
                <button className="rounded-full border-2 border-dashed border-white/20 text-white/40 text-xs font-black uppercase tracking-widest px-4 py-2 hover:border-[#FF3AF2] hover:text-[#FF3AF2] transition-all duration-200">
                  ···
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
