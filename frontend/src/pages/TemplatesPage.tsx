import { useState } from "react"

import { ACCENTS, BORDER_COLORS, TEMPLATES } from "./shared"

export default function TemplatesPage() {
  const [filter, setFilter] = useState("All")

  const categories = [
    "All",
    "Social",
    "Instagram",
    "YouTube",
    "Facebook",
    "Stories",
    "E-comm",
  ]

  return (
    <div className="space-y-8">
      <div>
        <div className="text-[#FFE600] text-sm font-black uppercase tracking-widest mb-1">
          ✨ Library
        </div>
        <h1 className="font-['Unbounded'] font-black text-3xl md:text-4xl text-shadow-lg">
          <span className="gradient-text">Templates</span>
        </h1>
      </div>
      <div className="flex gap-3 flex-wrap">
        {categories.map((cat, i) => (
          <button
            key={cat}
            onClick={() => setFilter(cat)}
            className="rounded-full border-4 px-5 py-2 font-black text-xs uppercase tracking-widest transition-all duration-200 hover:scale-105"
            style={{
              borderColor:
                filter === cat ? ACCENTS[i % 5] : "rgba(255,255,255,0.15)",
              background:
                filter === cat ? `${ACCENTS[i % 5]}22` : "transparent",
              color: filter === cat ? ACCENTS[i % 5] : "rgba(255,255,255,0.5)",
            }}
          >
            {cat}
          </button>
        ))}
      </div>
      <div className="grid grid-cols-2 md:grid-cols-3 gap-6">
        {TEMPLATES.filter((t) => filter === "All" || t.category === filter).map(
          (t, i) => (
            <div
              key={t.name}
              className={`group relative rounded-3xl border-4 ${BORDER_COLORS[i % 5]} bg-[#2D1B4E] p-5 overflow-hidden cursor-pointer transition-all duration-300 hover:scale-[1.04] hover:-rotate-1 ${
                i % 2 === 1 ? "translate-y-4" : ""
              }`}
              style={{ boxShadow: `8px 8px 0 ${ACCENTS[(i + 1) % 5]}` }}
            >
              <div
                className="pointer-events-none absolute inset-0 pattern-stripes opacity-[0.06]"
                aria-hidden="true"
              />
              {t.hot && (
                <span className="absolute top-3 right-3 text-xs font-black bg-[#FF3AF2] text-[#0D0D1A] rounded-full px-2 py-0.5 border-2 border-[#FFE600]">
                  🔥
                </span>
              )}
              <div
                className="w-full aspect-[9/16] max-h-48 rounded-2xl border-2 flex items-center justify-center mb-4 text-4xl transition-transform duration-300 group-hover:scale-110"
                style={{
                  borderColor: ACCENTS[i % 5],
                  background: `linear-gradient(135deg, ${ACCENTS[i % 5]}22, ${ACCENTS[(i + 1) % 5]}22)`,
                }}
              >
                🎬
              </div>
              <div
                className="font-['Unbounded'] font-black text-sm leading-tight"
                style={{ color: ACCENTS[i % 5] }}
              >
                {t.name}
              </div>
              <div className="flex gap-2 mt-2 items-center flex-wrap">
                <span className="text-xs font-bold text-white/40 uppercase tracking-widest">
                  {t.ratio}
                </span>
                <span className="text-xs bg-[#0D0D1A]/60 border border-white/10 rounded-full px-2 py-0.5 text-white/40">
                  {t.category}
                </span>
              </div>
              <div className="text-xs text-white/40 mt-2">
                {t.uses.toLocaleString()} uses
              </div>
              <button
                className="mt-3 w-full rounded-full border-2 font-black uppercase tracking-widest text-xs py-2 transition-all duration-200 group-hover:scale-105"
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
                Use Template
              </button>
            </div>
          ),
        )}
      </div>
    </div>
  )
}
