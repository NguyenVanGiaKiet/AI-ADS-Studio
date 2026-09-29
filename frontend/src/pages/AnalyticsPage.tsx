import { ACCENTS } from "./shared"

import { StatusBadge } from "./shared"

export default function AnalyticsPage() {
  const weekData = [42, 78, 55, 91, 113, 88, 127]

  const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

  const maxVal = Math.max(...weekData)

  return (
    <div className="space-y-8">
      <div>
        <div className="text-[#FF6B35] text-sm font-black uppercase tracking-widest mb-1">
          📊 Insights
        </div>
        <h1 className="font-['Unbounded'] font-black text-3xl md:text-4xl text-shadow-lg">
          <span className="gradient-text">Analytics</span>
        </h1>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div
          className="rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-6"
          style={{ boxShadow: "8px 8px 0 #FFE600, 16px 16px 0 #7B2FFF" }}
        >
          <div className="font-['Unbounded'] font-black text-sm uppercase tracking-wide text-[#FF3AF2] mb-4">
            Videos This Week
          </div>
          <div className="flex items-end gap-2 h-40">
            {weekData.map((val, i) => (
              <div key={i} className="flex-1 flex flex-col items-center gap-1">
                <div className="text-xs font-bold text-white/60">{val}</div>
                <div
                  className="w-full rounded-t-lg transition-all duration-700 hover:opacity-80"
                  style={{
                    height: `${(val / maxVal) * 100}%`,
                    background: `linear-gradient(180deg, ${ACCENTS[i % 5]}, ${ACCENTS[(i + 1) % 5]})`,
                    boxShadow: `0 0 12px ${ACCENTS[i % 5]}66`,
                    minHeight: "8px",
                  }}
                />
                <div className="text-xs text-white/40 font-bold">{days[i]}</div>
              </div>
            ))}
          </div>
        </div>
        <div
          className="rounded-3xl border-4 border-[#00F5D4] bg-[#2D1B4E] p-6"
          style={{ boxShadow: "8px 8px 0 #FF6B35" }}
        >
          <div className="font-['Unbounded'] font-black text-sm uppercase tracking-wide text-[#00F5D4] mb-4">
            Platform Split
          </div>
          <div className="space-y-4">
            {[
              { platform: "TikTok", pct: 42, accent: ACCENTS[0] },
              { platform: "Instagram", pct: 28, accent: ACCENTS[1] },
              { platform: "YouTube", pct: 18, accent: ACCENTS[2] },
              { platform: "Facebook", pct: 12, accent: ACCENTS[3] },
            ].map((p) => (
              <div key={p.platform}>
                <div className="flex justify-between text-xs font-bold mb-1">
                  <span className="text-white/80">{p.platform}</span>
                  <span style={{ color: p.accent }}>{p.pct}%</span>
                </div>
                <div className="h-3 rounded-full bg-[#0D0D1A] border border-white/10 overflow-hidden">
                  <div
                    className="h-full rounded-full transition-all duration-1000"
                    style={{
                      width: `${p.pct}%`,
                      background: p.accent,
                      boxShadow: `0 0 8px ${p.accent}88`,
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
        <div
          className="md:col-span-2 rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E] overflow-hidden"
          style={{ boxShadow: "8px 8px 0 #FF3AF2" }}
        >
          <div className="border-b-4 border-dashed border-[#FFE600] px-6 py-4">
            <div className="font-['Unbounded'] font-black text-sm uppercase tracking-wide text-[#FFE600]">
              🏆 Top Performing Batches
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b-2 border-dashed border-[#FFE600]/30">
                  {[
                    "Batch Name",
                    "Videos",
                    "Platform",
                    "CTR",
                    "Views",
                    "Status",
                  ].map((h, i) => (
                    <th
                      key={h}
                      className="px-6 py-3 text-left text-xs font-black uppercase tracking-widest"
                      style={{ color: ACCENTS[i % 5] }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-dashed divide-[#FFE600]/20">
                {[
                  [
                    "Summer Sale Campaign",
                    "120",
                    "TikTok",
                    "4.2%",
                    "89,241",
                    "done",
                  ],
                  [
                    "Product Launch Series",
                    "50",
                    "YouTube",
                    "3.8%",
                    "45,002",
                    "done",
                  ],
                  [
                    "Flash Deal Reels",
                    "200",
                    "Instagram",
                    "5.1%",
                    "—",
                    "processing",
                  ],
                  [
                    "Brand Story Montage",
                    "30",
                    "YouTube",
                    "2.9%",
                    "—",
                    "processing",
                  ],
                ].map((row, i) => (
                  <tr
                    key={i}
                    className="hover:bg-[#FFE600]/5 transition-colors"
                  >
                    {row.map((cell, j) => (
                      <td
                        key={j}
                        className={`px-6 py-4 text-sm ${
                          j === 0
                            ? "font-bold text-white"
                            : j === row.length - 1
                              ? ""
                              : "text-white/60"
                        }`}
                      >
                        {j === row.length - 1 ? (
                          <StatusBadge status={cell} />
                        ) : (
                          cell
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
