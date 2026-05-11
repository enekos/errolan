// emoji.jsx — Custom emoji pack: small editorial SVG illustrations.
// Each emoji has a :shortcode:, a label, and a renderer at any size.

const EmojiSVG = ({ name, size = 18, style = {} }) => {
  const s = size;
  const common = { width: s, height: s, viewBox: '0 0 24 24', style: { display:'inline-block', verticalAlign:'-0.18em', ...style } };
  switch(name){
    case 'ship-it': return (
      <svg {...common}><g fill="none" stroke="#1A3A6E" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
        <path d="M3 16h18l-1.6 4H4.6L3 16Z" fill="#1A3A6E" stroke="none"/>
        <path d="M5.5 16V11h13v5"/>
        <path d="M9 11V7h4"/>
        <path d="M13 7l3 4"/>
        <circle cx="12" cy="14" r="0.9" fill="#1A3A6E" stroke="none"/>
        <path d="M2 14c1.5-1 3-1 4 0M18 14c1.5-1 3-1 4 0" />
      </g></svg>);
    case 'hot-take': return (
      <svg {...common}><g fill="#B7351D" stroke="#7A2412" strokeWidth="1.1" strokeLinejoin="round">
        <path d="M12 3c1 3 4 4 4 8a4 4 0 1 1-8 0c0-2 1-2 1-4 1 1 2 1 3-4Z"/>
        <path d="M12 13c.6 1.5 2 2 2 4a2 2 0 1 1-4 0c0-1 .6-1.2 1-2 .4.5 1 .5 1-2Z" fill="#FBE89A" stroke="#7A2412"/>
      </g></svg>);
    case 'this': return (
      <svg {...common}><g fill="#16140F" stroke="#16140F" strokeWidth="0.8" strokeLinejoin="round">
        <path d="M5 13c0-1 .8-1.5 1.5-1.4l3 .4V6.5a1.3 1.3 0 0 1 2.6 0v4.2l4.2.6c1 .1 1.7.9 1.7 1.9V17a3 3 0 0 1-3 3h-4.5c-.9 0-1.7-.4-2.3-1.1L5.8 16C5.3 15.5 5 14.8 5 14v-1Z" fill="#FAF7EF"/>
        <path d="M9.5 12V7"/>
      </g></svg>);
    case 'rubber-duck': return (
      <svg {...common}><g fill="#F1B12A" stroke="#7A4A0A" strokeWidth="1" strokeLinejoin="round">
        <ellipse cx="11" cy="16" rx="7" ry="3.2"/>
        <circle cx="15" cy="10" r="3.4"/>
        <path d="M18 9.6l3 .2c.4 0 .5.5.1.7l-2.6 1.2" fill="#B7351D"/>
        <circle cx="16" cy="9.4" r=".7" fill="#16140F" stroke="none"/>
        <path d="M4 16c1-1.4 2.5-1.6 4-1" fill="none"/>
      </g></svg>);
    case 'galaxy-brain': return (
      <svg {...common}><g fill="none" stroke="#5B2A56" strokeWidth="1.1" strokeLinecap="round">
        <path d="M12 12m-7 0a7 7 0 1 0 14 0a7 7 0 1 0-14 0" fill="#F3E6F1"/>
        <path d="M12 5c-3 1.5-4 4-3 6s4 2 5 0 0-4 3-4M9 11c1 2 4 2 5 0M8 14c2 2 5 2 7 0"/>
        <circle cx="9" cy="9" r=".7" fill="#5B2A56"/>
        <circle cx="15" cy="14" r=".7" fill="#5B2A56"/>
      </g></svg>);
    case 'wrong': return (
      <svg {...common}><g stroke="#B7351D" strokeWidth="2.2" strokeLinecap="round" fill="none">
        <circle cx="12" cy="12" r="8" fill="#FAE0DA" stroke="#B7351D" strokeWidth="1.2"/>
        <path d="M8 8l8 8M16 8l-8 8"/>
      </g></svg>);
    case 'skull': return (
      <svg {...common}><g fill="#F0EBDC" stroke="#16140F" strokeWidth="1.1" strokeLinejoin="round">
        <path d="M5 11a7 7 0 0 1 14 0v3.5c0 .7-.4 1.3-1 1.6l-1 .5v2H7v-2l-1-.5c-.6-.3-1-.9-1-1.6V11Z"/>
        <circle cx="9" cy="12.5" r="1.6" fill="#16140F"/>
        <circle cx="15" cy="12.5" r="1.6" fill="#16140F"/>
        <path d="M11 16h2M10.5 18v1M13.5 18v1M12 16v1"/>
      </g></svg>);
    case 'plus-one': return (
      <svg {...common}><g fill="#2E6A3E" stroke="#1A4A2A" strokeWidth="1" strokeLinejoin="round">
        <path d="M6 12c0-.8.6-1.4 1.4-1.4h2.6V7a1.6 1.6 0 0 1 3.2 0v3.6h3.4A1.4 1.4 0 0 1 18 12v5a3 3 0 0 1-3 3h-5a3 3 0 0 1-3-3v-5Z" fill="#D6E8DA"/>
      </g></svg>);
    case 'eyes': return (
      <svg {...common}><g>
        <ellipse cx="8" cy="13" rx="3.6" ry="3.2" fill="#FAF7EF" stroke="#16140F" strokeWidth="1"/>
        <ellipse cx="16" cy="13" rx="3.6" ry="3.2" fill="#FAF7EF" stroke="#16140F" strokeWidth="1"/>
        <circle cx="9" cy="13.5" r="1.6" fill="#16140F"/>
        <circle cx="17" cy="13.5" r="1.6" fill="#16140F"/>
        <path d="M4 11c1-3 5-4 7-2M13 9c2-2 6-1 7 2" fill="none" stroke="#16140F" strokeWidth="1" strokeLinecap="round"/>
      </g></svg>);
    case 'tea': return (
      <svg {...common}><g fill="#FAF7EF" stroke="#16140F" strokeWidth="1.1" strokeLinejoin="round">
        <path d="M5 11h11v5a4 4 0 0 1-4 4H9a4 4 0 0 1-4-4v-5Z"/>
        <path d="M16 13h2a2 2 0 0 1 0 4h-2" fill="none"/>
        <path d="M8 8c0-1 1-1.5 1-3M11 8c0-1 1-1.5 1-3M14 8c0-1 1-1.5 1-3" stroke="#B7351D" fill="none"/>
      </g></svg>);
    case 'merge': return (
      <svg {...common}><g fill="none" stroke="#5B2A56" strokeWidth="1.5" strokeLinecap="round">
        <circle cx="7" cy="6" r="2" fill="#F3E6F1"/>
        <circle cx="7" cy="18" r="2" fill="#F3E6F1"/>
        <circle cx="17" cy="14" r="2" fill="#F3E6F1"/>
        <path d="M7 8v8"/>
        <path d="M7 11c0 2 2 3 4 3h4"/>
      </g></svg>);
    case 'spicy': return (
      <svg {...common}><g fill="#B7351D" stroke="#7A2412" strokeWidth="1" strokeLinejoin="round">
        <path d="M8 17c2 1 4 1 7-.5 3-1.6 5-4 4.5-5.5-.5-1.4-3-1.2-6 .4-3 1.7-5.4 4-5.5 5.6Z"/>
        <path d="M6.5 18c-.5.5-2 1.2-3 .8 0-1 .7-2.5 1.5-3" fill="#2E6A3E" stroke="#1A4A2A"/>
      </g></svg>);
    default: return <span style={{fontFamily:'var(--mono)', fontSize:s*0.8}}>:{name}:</span>;
  }
};

const EMOJI_PACK = [
  { code:'ship-it',     label:'Ship it'        },
  { code:'hot-take',    label:'Hot take'       },
  { code:'this',        label:'This'           },
  { code:'plus-one',    label:'+1'             },
  { code:'rubber-duck', label:'Rubber duck'    },
  { code:'galaxy-brain',label:'Galaxy brain'   },
  { code:'wrong',       label:'Disagree'       },
  { code:'skull',       label:'Dead'           },
  { code:'eyes',        label:'Watching'       },
  { code:'tea',         label:'Tea'            },
  { code:'merge',       label:'Merged'         },
  { code:'spicy',       label:'Spicy'          },
];

// Inline emoji parser — converts ":code:" tokens inside a string into <EmojiSVG>.
const ParseEmoji = ({ text, size = 16 }) => {
  const parts = text.split(/(:[a-z-]+:)/g);
  return <>{parts.map((p,i)=>{
    const m = p.match(/^:([a-z-]+):$/);
    if (m && EMOJI_PACK.find(e=>e.code===m[1])) return <EmojiSVG key={i} name={m[1]} size={size} />;
    return <span key={i}>{p}</span>;
  })}</>;
};

Object.assign(window, { EmojiSVG, EMOJI_PACK, ParseEmoji });
