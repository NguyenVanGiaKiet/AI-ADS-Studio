import { useEffect, useState } from "react"
import type { ReactNode } from "react"

import AnalyticsPage from "./pages/AnalyticsPage"
import RemixVideoPage from "./pages/RemixVideoPage"
import DashboardPage from "./pages/DashboardPage"
import ProjectsPage from "./pages/ProjectsPage"
import SettingsPage from "./pages/SettingsPage"
import TemplatesPage from "./pages/TemplatesPage"
import CreatedVideosPage from "./pages/CreatedVideosPage"
import { ACCENTS, apiFetch } from "./pages/shared"

type Tab = "dashboard" | "create" | "videos" | "projects" | "templates" | "analytics" | "settings"
const ACTIVE_TAB_KEY = "ai-ads-studio:active-tab"

const NAV_ITEMS: { id: Tab; label: string; icon: string }[] = [
  { id: "dashboard", label: "Tổng quan", icon: "⚡" },
  { id: "create", label: "Remix Video", icon: "🎬" },
  { id: "videos", label: "Video Đã Tạo", icon: "🎞️" },
  { id: "projects", label: "Lịch sử tác vụ", icon: "⚙️" },
  { id: "templates", label: "Hướng dẫn", icon: "📖" },
  { id: "analytics", label: "Thống kê", icon: "📊" },
  { id: "settings", label: "Hệ thống", icon: "🛠️" },
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

function Sidebar({
  active,
  onNav,
  onLogout,
  logoutError,
}: {
  active: Tab
  onNav: (tab: Tab) => void
  onLogout: () => void
  logoutError: string
}) {
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
              VIETNAMESE TTS
            </div>
          </div>
        </div>
        <div className="mt-4 bg-[#00F5D4]/10 border-2 border-dashed border-[#00F5D4] rounded-2xl px-4 py-2">
          <div className="text-[#00F5D4] text-xs font-black uppercase tracking-widest">
            Piper chạy cục bộ
          </div>
          <div className="text-white/60 text-xs mt-0.5">
            Groq dùng riêng để tạo kịch bản.
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
        {logoutError && (
          <p role="alert" className="mb-2 text-xs text-[#FF8B75]">{logoutError}</p>
        )}
        <div className="rounded-3xl border-4 border-[#FFE600] bg-[#2D1B4E] p-4 shadow-multi-sm">
          <div className="text-[#FFE600] text-xs font-black uppercase tracking-widest animate-wiggle inline-block">
            🎬 TẠO VIDEO
          </div>
          <div className="font-['Unbounded'] font-black text-sm mt-1 text-white">
            Remix từ video nguồn
          </div>
          <div className="text-white/60 text-xs mt-1">
            Cắt ghép, lồng giọng Việt và xuất video quảng cáo.
          </div>
          <button
            onClick={() => onNav("create")}
            className="mt-3 w-full rounded-full border-2 border-[#FFE600] text-[#FFE600] text-xs font-black uppercase tracking-widest py-2 hover:bg-[#FFE600] hover:text-[#0D0D1A] transition-all duration-200"
          >
            Bắt đầu
          </button>
        </div>
        <button
          type="button"
          onClick={onLogout}
          className="mt-3 w-full rounded-full border-2 border-white/20 px-4 py-2 text-xs font-black uppercase tracking-widest text-white/70 transition hover:border-[#FF6B35] hover:text-[#FF6B35]"
        >
          Đăng xuất
        </button>
      </div>
    </aside>
  )
}

function LoginScreen({
  onAuthenticated,
  error,
  onRetry,
}: {
  onAuthenticated: () => void
  error?: string
  onRetry?: () => void
}) {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [loginError, setLoginError] = useState("")
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoginError("")
    setSubmitting(true)
    try {
      const response = await apiFetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })
      if (!response.ok) {
        throw new Error((await response.text()) || "Đăng nhập thất bại.")
      }
      setPassword("")
      onAuthenticated()
    } catch (requestError) {
      setLoginError(
        requestError instanceof Error
          ? requestError.message
          : "Không thể đăng nhập.",
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#0D0D1A] px-4 py-10 text-white">
      <div className="pointer-events-none fixed inset-0 pattern-dots opacity-[0.03]" aria-hidden="true" />
      <div className="pointer-events-none fixed inset-0 pattern-mesh opacity-50" aria-hidden="true" />
      <Floaties />
      <section className="relative z-10 w-full max-w-md rounded-3xl border-4 border-[#FF3AF2] bg-[#2D1B4E] p-7 shadow-[10px_10px_0_#FFE600] sm:p-9">
        <div className="mb-6 text-center">
          <div className="mx-auto mb-4 flex size-14 items-center justify-center rounded-2xl border-4 border-[#FFE600] bg-[#FF3AF2] text-2xl">
            🎬
          </div>
          <p className="text-xs font-black uppercase tracking-[0.2em] text-[#00F5D4]">
            AI ADS STUDIO
          </p>
          <h1 className="mt-2 font-['Unbounded'] text-2xl font-black">
            Đăng nhập
          </h1>
          <p className="mt-2 text-sm text-white/55">
            Đăng nhập bằng tài khoản quản trị để tiếp tục.
          </p>
        </div>
        {error ? (
          <div className="space-y-4 text-center">
            <p role="alert" className="text-sm text-[#FFB08E]">{error}</p>
            {onRetry && (
              <button
                type="button"
                onClick={onRetry}
                className="rounded-full border-2 border-[#00F5D4] px-5 py-2 text-xs font-black uppercase tracking-widest text-[#00F5D4] hover:bg-[#00F5D4] hover:text-[#0D0D1A]"
              >
                Thử kết nối lại
              </button>
            )}
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <label className="block">
              <span className="mb-2 block text-xs font-black uppercase tracking-widest text-[#FFE600]">
                Tên đăng nhập
              </span>
              <input
                autoComplete="username"
                required
                value={username}
                onChange={event => setUsername(event.target.value)}
                className="w-full rounded-xl border-2 border-[#7B2FFF] bg-[#0D0D1A] px-4 py-3 text-sm text-white outline-none focus:border-[#00F5D4]"
              />
            </label>
            <label className="block">
              <span className="mb-2 block text-xs font-black uppercase tracking-widest text-[#FFE600]">
                Mật khẩu
              </span>
              <input
                autoComplete="current-password"
                required
                type="password"
                value={password}
                onChange={event => setPassword(event.target.value)}
                className="w-full rounded-xl border-2 border-[#7B2FFF] bg-[#0D0D1A] px-4 py-3 text-sm text-white outline-none focus:border-[#00F5D4]"
              />
            </label>
            {loginError && (
              <p role="alert" className="text-sm text-[#FF8B75]">{loginError}</p>
            )}
            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-full border-4 border-[#FFE600] bg-gradient-to-r from-[#FF3AF2] via-[#7B2FFF] to-[#00F5D4] px-6 py-3 text-xs font-black uppercase tracking-widest text-white transition hover:scale-[1.02] disabled:cursor-wait disabled:opacity-60"
            >
              {submitting ? "Đang xác thực…" : "Đăng nhập"}
            </button>
          </form>
        )}
      </section>
    </main>
  )
}

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>(() => {
    const savedTab = window.localStorage.getItem(ACTIVE_TAB_KEY)
    return NAV_ITEMS.find(item => item.id === savedTab)?.id ?? "dashboard"
  })
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [authState, setAuthState] = useState<"checking" | "authenticated" | "unauthenticated" | "error">("checking")
  const [authError, setAuthError] = useState("")
  const [logoutError, setLogoutError] = useState("")

  async function checkAuthentication() {
    setAuthState("checking")
    setAuthError("")
    try {
      const response = await apiFetch("/api/auth/session")
      if (!response.ok) throw new Error(`Không thể kiểm tra đăng nhập (HTTP ${response.status})`)
      const payload: { data?: { authenticated?: boolean } } = await response.json()
      setAuthState(payload.data?.authenticated ? "authenticated" : "unauthenticated")
    } catch (requestError) {
      setAuthError(
        requestError instanceof Error
          ? requestError.message
          : "Không thể kết nối đến backend.",
      )
      setAuthState("error")
    }
  }

  useEffect(() => {
    void checkAuthentication()
  }, [])

  useEffect(() => {
    window.localStorage.setItem(ACTIVE_TAB_KEY, activeTab)
  }, [activeTab])

  async function logout() {
    setLogoutError("")
    try {
      const response = await apiFetch("/api/auth/logout", { method: "POST" })
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      setAuthState("unauthenticated")
    } catch {
      setLogoutError("Không thể đăng xuất. Vui lòng thử lại.")
    }
  }

  if (authState === "checking") {
    return (
      <main className="flex min-h-screen items-center justify-center bg-[#0D0D1A] text-sm font-bold text-[#00F5D4]">
        Đang kiểm tra phiên đăng nhập…
      </main>
    )
  }
  if (authState === "error" || authState === "unauthenticated") {
    return (
      <LoginScreen
        error={authState === "error" ? authError : undefined}
        onRetry={authState === "error" ? () => void checkAuthentication() : undefined}
        onAuthenticated={() => setAuthState("authenticated")}
      />
    )
  }

  const renderActivePage = () => {
    switch (activeTab) {
      case "dashboard":
        return <DashboardPage onNavigate={setActiveTab} />
      case "create":
        return null
      case "videos":
        return <CreatedVideosPage onNavigateToCreate={() => setActiveTab("create")} />
      case "projects":
        return <ProjectsPage onNavigateToCreate={() => setActiveTab("create")} />
      case "templates":
        return <TemplatesPage onNavigateToCreate={() => setActiveTab("create")} />
      case "analytics":
        return <AnalyticsPage />
      case "settings":
        return <SettingsPage />
      default:
        return <DashboardPage onNavigate={setActiveTab} />
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
            logoutError={logoutError}
            onLogout={() => void logout()}
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
          {activeTab !== "create" && renderActivePage()}
          <div className={activeTab === "create" ? "block" : "hidden"}>
            <RemixVideoPage />
          </div>
        </div>
      </main>
    </div>
  )
}
