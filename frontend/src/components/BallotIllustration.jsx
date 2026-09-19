export default function BallotIllustration() {
  return (
    <svg viewBox="0 0 400 460" fill="none" xmlns="http://www.w3.org/2000/svg" style={{ width: '100%', maxWidth: 320 }}>
      {/* ballot box */}
      <rect x="70" y="220" width="260" height="180" rx="4" fill="#0f1829" stroke="#c98a4b" strokeWidth="2" />
      <rect x="70" y="220" width="260" height="34" rx="4" fill="#16233d" stroke="#c98a4b" strokeWidth="2" />
      <rect x="150" y="228" width="100" height="16" rx="8" fill="#0b1220" />

      {/* seal on front of box */}
      <circle cx="200" cy="330" r="34" fill="none" stroke="#c98a4b" strokeWidth="2.5" />
      <path d="M186 330 L196 340 L216 316" stroke="#c98a4b" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none" />

      {/* falling ballot papers */}
      <g>
        <rect x="150" y="70" width="70" height="90" rx="3" fill="#fbf9f3" stroke="#16233d" strokeWidth="2" transform="rotate(-8 150 70)" />
        <line x1="163" y1="95" x2="205" y2="90" stroke="#b9ae8e" strokeWidth="2" transform="rotate(-8 150 70)" />
        <line x1="163" y1="110" x2="205" y2="105" stroke="#b9ae8e" strokeWidth="2" transform="rotate(-8 150 70)" />
        <circle cx="172" cy="128" r="6" fill="none" stroke="#a13d3d" strokeWidth="2" transform="rotate(-8 150 70)" />
      </g>
      <g>
        <rect x="210" y="110" width="70" height="90" rx="3" fill="#fbf9f3" stroke="#16233d" strokeWidth="2" transform="rotate(10 210 110)" />
        <line x1="223" y1="135" x2="265" y2="130" stroke="#b9ae8e" strokeWidth="2" transform="rotate(10 210 110)" />
        <line x1="223" y1="150" x2="265" y2="145" stroke="#b9ae8e" strokeWidth="2" transform="rotate(10 210 110)" />
        <circle cx="232" cy="168" r="6" fill="none" stroke="#a13d3d" strokeWidth="2" transform="rotate(10 210 110)" />
      </g>

      {/* slot arrow */}
      <path d="M200 190 L200 216" stroke="#c98a4b" strokeWidth="2.5" strokeLinecap="round" />
      <path d="M191 207 L200 218 L209 207" stroke="#c98a4b" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" fill="none" />
    </svg>
  )
}
