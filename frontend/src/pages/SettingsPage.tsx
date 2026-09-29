import { ACCENTS, BORDER_COLORS } from "./shared"

export default function SettingsPage() {
  return (
    <div className="space-y-8">
      <div>
        <div className="text-[#7B2FFF] text-sm font-black uppercase tracking-widest mb-1">
          ⚙️ Config
        </div>
        <h1 className="font-['Unbounded'] font-black text-3xl text-shadow-lg">
          <span className="gradient-text">Settings</span>
        </h1>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {[
          {
            title: "Account",
            icon: "👤",
            accent: ACCENTS[0],
            fields: ["Display Name", "Email Address", "API Key"],
          },

          {
            title: "Brand Kit",
            icon: "🎨",
            accent: ACCENTS[1],
            fields: ["Primary Color", "Logo URL", "Font Choice"],
          },

          {
            title: "Export Defaults",
            icon: "📤",
            accent: ACCENTS[2],
            fields: ["Default Format", "Default Resolution", "Watermark"],
          },

          {
            title: "Notifications",
            icon: "🔔",
            accent: ACCENTS[3],
            fields: ["Email on Complete", "Slack Webhook", "Browser Push"],
          },
        ].map((section, i) => (
          <div
            key={section.title}
            className={`rounded-3xl border-4 ${BORDER_COLORS[i % 5]} bg-[#2D1B4E] p-6`}
            style={{ boxShadow: `8px 8px 0 ${ACCENTS[(i + 1) % 5]}` }}
          >
            <div className="flex items-center gap-3 mb-5">
              <span
                className="text-2xl animate-wiggle inline-block"
                aria-hidden="true"
              >
                {section.icon}
              </span>
              <div
                className="font-['Unbounded'] font-black text-sm uppercase tracking-wide"
                style={{ color: section.accent }}
              >
                {section.title}
              </div>
            </div>
            <div className="space-y-3">
              {section.fields.map((field) => (
                <div key={field}>
                  <label
                    className="text-xs font-black uppercase tracking-widest mb-1.5 block"
                    style={{ color: section.accent }}
                  >
                    {field}
                  </label>
                  <input
                    type="text"
                    placeholder={`Enter ${field.toLowerCase()}…`}
                    className="w-full rounded-full border-4 bg-[#0D0D1A]/60 px-5 py-3 text-sm font-bold text-white placeholder-white/25 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2"
                    style={{ borderColor: section.accent + "66" }}
                    onFocus={(e) => {
                      e.currentTarget.style.borderColor = section.accent
                    }}
                    onBlur={(e) => {
                      e.currentTarget.style.borderColor = section.accent + "66"
                    }}
                  />
                </div>
              ))}
            </div>
            <button
              className="mt-5 rounded-full border-2 font-black uppercase tracking-widest text-xs px-6 py-2.5 transition-all duration-200 hover:scale-105"
              style={{ borderColor: section.accent, color: section.accent }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = section.accent
                e.currentTarget.style.color = "#0D0D1A"
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "transparent"
                e.currentTarget.style.color = section.accent
              }}
            >
              Save Changes
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
