/**
 * MCP Gateway logo — gateway arch with flow arrow, cyan→indigo→purple gradient.
 * The arch represents the proxy/bridge between REST APIs (left) and MCP tools (right).
 */
export default function Logo({ size = 32 }: { size?: number }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      width={size}
      height={size}
      aria-label="MCP Gateway logo"
    >
      <defs>
        <linearGradient id="logo-bg" x1="0" y1="0" x2="64" y2="64" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#0d1b2a"/>
          <stop offset="100%" stopColor="#180d3d"/>
        </linearGradient>
        <linearGradient id="logo-gate" x1="32" y1="8" x2="32" y2="56" gradientUnits="userSpaceOnUse">
          <stop offset="0%"   stopColor="#06b6d4"/>
          <stop offset="55%"  stopColor="#6366f1"/>
          <stop offset="100%" stopColor="#a855f7"/>
        </linearGradient>
        <linearGradient id="logo-arrow" x1="20" y1="39" x2="44" y2="39" gradientUnits="userSpaceOnUse">
          <stop offset="0%"   stopColor="#67e8f9"/>
          <stop offset="100%" stopColor="#e879f9"/>
        </linearGradient>
        <filter id="logo-glow">
          <feGaussianBlur stdDeviation="1.5" result="b"/>
          <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
        </filter>
      </defs>

      {/* Background */}
      <rect width="64" height="64" rx="13" fill="url(#logo-bg)"/>
      <rect x="1" y="1" width="62" height="28" rx="12" fill="white" opacity="0.04"/>

      {/* Gateway arch: two pillars connected by a curved top */}
      <path
        d="M14 56 L14 25 Q14 9 32 9 Q50 9 50 25 L50 56"
        fill="none"
        stroke="url(#logo-gate)"
        strokeWidth="7"
        strokeLinecap="round"
        strokeLinejoin="round"
        filter="url(#logo-glow)"
      />

      {/* Left side: REST endpoint lines */}
      <rect x="3"  y="29" width="8" height="3.5" rx="1.75" fill="#22d3ee" opacity="0.80"/>
      <rect x="3"  y="37" width="6" height="3.5" rx="1.75" fill="#22d3ee" opacity="0.45"/>

      {/* Right side: MCP tool nodes */}
      <circle cx="59" cy="30" r="2.5" fill="#c084fc" opacity="0.80"/>
      <circle cx="59" cy="39" r="2.5" fill="#c084fc" opacity="0.45"/>

      {/* Flow arrow through the arch opening */}
      <line x1="20" y1="39" x2="38" y2="39" stroke="url(#logo-arrow)" strokeWidth="3" strokeLinecap="round"/>
      <polyline
        points="32,33 39.5,39 32,45"
        fill="none"
        stroke="url(#logo-arrow)"
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}
