import { useState } from "react"
import type { ReactNode } from "react"

import AnalyticsPage from "./pages/AnalyticsPage"
import RemixVideoPage from "./pages/RemixVideoPage"
import DashboardPage from "./pages/DashboardPage"
import ProjectsPage from "./pages/ProjectsPage"
import SettingsPage from "./pages/SettingsPage"
import TemplatesPage from "./pages/TemplatesPage"
import CreatedVideosPage from "./pages/CreatedVideosPage"
import { ACCENTS } from "./pages/shared"

type Tab = "dashboard" | "create" | "videos" | "projects" | "templates" | "analytics" | "settings"

const NAV_ITEMS: { id: Tab; label: string; icon: string }[] = [
  { id: "dashboard", label: "Dashboard", icon: "⚡" },
  { id: "create", label: "Remix Video", icon: "🎬" },
  { id: "videos", label: "Video Đã Tạo", icon: "🎞️" },
  { id: "projects", label: "Projects", icon: "📁" },
  { id: "templates", label: "Templates", icon: "✨" },
  { id: "analytics", label: "Analytics", icon: "📊" },
  { id: "settings", label: "Settings", icon: "⚙️" },
]

function Floaties() {
  return (
    <div
      className="pointer-events-none fixed inset-0 overflow-hidden z-0"
      aria-hidden="true"
    >
      <div className="animate-float absolute top-[8%] left-[3%] text-[#FF3AF2] text-5xl opacity-40">
        ★
      </div>
      <div className="animate-float-rev absolute top-[15%] right-[5%] text-[#00F5D4] text-6xl opacity-30">
        ✦
      </div>
      <div className="animate-float-slow absolute top-[40%] left-[1%] text-[#FFE600] text-4xl opacity-35">
        ◆
      </div>
      <div className="animate-wiggle absolute top-[60%] right-[3%] text-[#FF6B35] text-5xl opacity-30">
        ●
      </div>
      <div className="animate-float absolute top-[80%] left-[6%] text-[#7B2FFF] text-3xl opacity-40">
        ▲
      </div>
      <div className="animate-bounce-up absolute top-[25%] left-[45%] text-[#FF3AF2] text-2xl opacity-20">
        ✸
      </div>
      <div className="animate-float-rev absolute top-[70%] right-[8%] text-[#00F5D4] text-7xl opacity-15">
        ○
      </div>
      <div className="animate-spin-slow absolute top-[50%] left-[48%] text-[#FFE600] text-2xl opacity-15">
        ✚
      </div>
    </div>
  )
}

function Sidebar({ active, onNav }: { active: Tab; onNav: (tab: Tab) => void }) {
  return (
    <aside
      className="relative w-72 h-full min-h-screen flex flex-col border-r-4 border-[#FF3AF2] bg-[#0D0D1A] z-20"
      style={{ boxShadow: "8px 0 32px rgba(255,58,242,0.2)" }}
    >
      <div
        className="pointer-events-none absolute inset-0 pattern-dots opacity-[0.04]"
        aria-hidden="true"
      />
      <div
        className="pointer-events-none absolute inset-0 pattern-stripes opacity-[0.05]"
        aria-hidden="true"
      />
      <div className="px-6 pt-8 pb-6 border-b-4 border-dashed border-[#7B2FFF]">
        <div className="flex items-center gap-3">
          <div className="w-12 h-12 rounded-2xl border-4 border-[#FFE600] bg-[#FF3AF2] flex items-center justify-center text-2xl animate-pulse-glow">
            🎬
          </div>
          <div>
            <div className="font-['Unbounded'] font-black text-lg leading-tight gradient-text">
              AI ADS Studio
            </div>
            <div className="text-[#00F5D4] text-xs font-bold uppercase tracking-widest">
              Pro · FREE
            </div>
          </div>
        </div>
        <div className="mt-4 bg-[#00F5D4]/10 border-2 border-dashed border-[#00F5D4] rounded-2xl px-4 py-2">
          <div className="text-[#00F5D4] text-xs font-black uppercase tracking-widest">
            ∞ Unlimited Credits
          </div>
          <div className="text-white/60 text-xs mt-0.5">
            Always free. No catch.
          </div>
        </div>
      </div>
      <nav className="flex-1 px-4 py-6 space-y-1 custom-scroll overflow-y-auto">
        {NAV_ITEMS.map((item, i) => {
          const accent = ACCENTS[i % ACCENTS.length]
          const isActive = active === item.id

          return (
            <button
              key={item.id}
              onClick={() => onNav(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-2xl transition-all duration-200 group text-left ${
                isActive
                  ? "text-[#0D0D1A] font-black"
                  : "text-white/70 hover:text-white hover:bg-[#2D1B4E]"
              }`}
              style={
                isActive
                  ? {
                      background: accent,
                      boxShadow: `4px 4px 0 ${ACCENTS[(i + 1) % ACCENTS.length]}`,
                    }
                  : {}
              }
            >
              <span
                className={`text-xl transition-transform duration-200 ${
                  isActive ? "" : "group-hover:scale-125"
                }`}
                aria-hidden="true"
              >
                {item.icon}
              </span>
              <span className="font-bold text-sm uppercase tracking-wide">
                {item.label}
              </span>
              {isActive && (
                <span className="ml-auto text-xs font-black">▶</span>
              )}
            </button>
          )
        })}
      </nav>
      <div className="px-4 pb-6">
        <div className="rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E] p-4 shadow-multi-sm">
          <div className="text-[#FFE600] text-xs font-black uppercase tracking-widest animate-wiggle inline-block">
            🔥 TRENDING
          </div>
          <div className="font-['Unbounded'] font-black text-sm mt-1 text-white">
            Bulk from CSV
          </div>
          <div className="text-white/60 text-xs mt-1">
            Generate 500 ads in one click using spreadsheet data.
          </div>
          <button className="mt-3 w-full rounded-full border-2 border-[#FFE600] text-[#FFE600] text-xs font-black uppercase tracking-widest py-2 hover:bg-[#FFE600] hover:text-[#0D0D1A] transition-all duration-200">
            Try Now
          </button>
        </div>
      </div>
    </aside>
  )
}

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>("dashboard")
  const [sidebarOpen, setSidebarOpen] = useState(false)

  const renderActivePage = () => {
    switch (activeTab) {
      case "dashboard":
        return <DashboardPage />
      case "create":
        return <RemixVideoPage />
      case "videos":
        return <CreatedVideosPage onNavigateToCreate={() => setActiveTab("create")} />
      case "projects":
        return <ProjectsPage />
      case "templates":
        return <TemplatesPage />
      case "analytics":
        return <AnalyticsPage />
      case "settings":
        return <SettingsPage />
      default:
        return <DashboardPage />
    }
  }

  return (
    <div className="min-h-screen bg-[#0D0D1A] flex relative">
      <div
        className="pointer-events-none fixed inset-0 pattern-dots opacity-[0.03] z-0"
        aria-hidden="true"
      />
      <div
        className="pointer-events-none fixed inset-0 pattern-stripes opacity-[0.04] z-0"
        aria-hidden="true"
      />
      <div
        className="pointer-events-none fixed inset-0 pattern-mesh opacity-50 z-0"
        aria-hidden="true"
      />
      <Floaties />
      <div
        className={`fixed inset-0 z-40 lg:static lg:z-auto lg:w-72 lg:shrink-0 transition-all duration-300 ${
          sidebarOpen
            ? "pointer-events-auto"
            : "pointer-events-none lg:pointer-events-auto"
        }`}
      >
        <div
          className={`absolute inset-0 bg-[#0D0D1A]/80 lg:hidden transition-opacity duration-300 ${
            sidebarOpen ? "opacity-100" : "opacity-0"
          }`}
          onClick={() => setSidebarOpen(false)}
        />
        <div
          className={`absolute left-0 top-0 h-full lg:fixed lg:inset-y-0 lg:left-0 transition-transform duration-300 ${
            sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
          }`}
        >
          <Sidebar
            active={activeTab}
            onNav={(tab) => {
              setActiveTab(tab)
              setSidebarOpen(false)
            }}
          />
        </div>
      </div>
      <button
        className="fixed right-4 top-4 z-30 flex size-10 items-center justify-center rounded-xl border-4 border-[#FF3AF2] bg-[#0D0D1A] text-[#FF3AF2] transition-all hover:bg-[#FF3AF2] hover:text-[#0D0D1A] lg:hidden"
        onClick={() => setSidebarOpen(true)}
        aria-label="Open menu"
      >
        ☰
      </button>
      <main className="flex-1 min-w-0 flex flex-col z-10">
        <div className="flex-1 p-6 pt-16 md:p-8 md:pt-8 lg:p-10 overflow-y-auto custom-scroll page-scroll">
          {renderActivePage()}
        </div>
      </main>
    </div>
  )
}
