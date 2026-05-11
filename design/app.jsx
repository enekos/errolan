// app.jsx — Mounts the design canvas with both widget directions + dashboard,
// and wires the Tweaks panel (light theme only — variations live as artboards).

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "accent":  "#B7351D",
  "serif":   "Newsreader",
  "density": "normal",
  "emojiSize": 16,
  "showSpeakerNotes": false
}/*EDITMODE-END*/;

const ACCENT_OPTIONS = ['#B7351D','#1C4F88','#2E6A3E','#5B2A56','#16140F'];
const SERIF_OPTIONS  = ['Newsreader','Source Serif 4','Cormorant Garamond','EB Garamond'];

const App = () => {
  const [tweaks, setTweak] = useTweaks(TWEAK_DEFAULTS);

  // Apply the few tweaks that propagate visually.
  React.useEffect(()=>{
    document.documentElement.style.setProperty('--accent', tweaks.accent);
    document.documentElement.style.setProperty('--accent-ink', shade(tweaks.accent, -25));
    document.documentElement.style.setProperty('--serif', `'${tweaks.serif}', Georgia, serif`);
  }, [tweaks.accent, tweaks.serif]);

  return (
    <>
      <DesignCanvas
        title="Errolan — comments, with character"
        subtitle="Self-hosted Disqus alt · refined-editorial direction · two widget concepts + studio"
      >
        <DCSection id="widgets" title="The widget" subtitle="What readers see on a blog post — two directions">
          <DCArtboard id="marginalia" label="A · Marginalia — paragraph-anchored comments in the margin" width={1280} height={1500}>
            <Marginalia tweaks={tweaks} />
          </DCArtboard>
          <DCArtboard id="cadence" label="B · Cadence — flat conversation river with time spine" width={920} height={1900}>
            <Cadence tweaks={tweaks} />
          </DCArtboard>
        </DCSection>

        <DCSection id="studio" title="The studio" subtitle="Admin moderation — site owners install Errolan and live here">
          <DCArtboard id="dashboard" label="Moderation queue · signals · emoji pack" width={1440} height={900}>
            <Dashboard />
          </DCArtboard>
        </DCSection>

        <DCSection id="pack" title="The emoji pack" subtitle="Per-site custom emoji — drawn here as small editorial woodcuts">
          <DCArtboard id="emoji-display" label="Pack at full size" width={760} height={420}>
            <EmojiPackShowcase />
          </DCArtboard>
        </DCSection>
      </DesignCanvas>

      <TweaksPanel title="Tweaks">
        <TweakSection title="Type">
          <TweakSelect label="Serif" value={tweaks.serif} onChange={v=>setTweak('serif', v)} options={SERIF_OPTIONS} />
          <TweakRadio  label="Density" value={tweaks.density} onChange={v=>setTweak('density', v)} options={['cozy','normal','dense']} />
        </TweakSection>
        <TweakSection title="Color">
          <TweakColor label="Accent" value={tweaks.accent} onChange={v=>setTweak('accent', v)} options={ACCENT_OPTIONS} />
        </TweakSection>
        <TweakSection title="Emoji">
          <TweakSlider label="Inline size" value={tweaks.emojiSize} onChange={v=>setTweak('emojiSize', v)} min={12} max={22} step={1} />
        </TweakSection>
      </TweaksPanel>
    </>
  );
};

// Lightly darken or lighten a hex color so the focus ring under the accent
// stays distinct without the user having to pick two.
function shade(hex, percent){
  const n = parseInt(hex.slice(1), 16);
  let r = (n >> 16) & 0xff, g = (n >> 8) & 0xff, b = n & 0xff;
  const t = percent < 0 ? 0 : 255;
  const p = Math.abs(percent) / 100;
  r = Math.round((t - r) * p) + r;
  g = Math.round((t - g) * p) + g;
  b = Math.round((t - b) * p) + b;
  return `#${((1 << 24) + (r<<16) + (g<<8) + b).toString(16).slice(1)}`;
}

const EmojiPackShowcase = () => (
  <div style={{ background:'var(--paper)', padding:'40px 48px', height:'100%', display:'flex', flexDirection:'column', justifyContent:'space-between' }}>
    <div>
      <div style={{ fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.18em', textTransform:'uppercase', color:'var(--accent)', marginBottom:14 }}>
        Site emoji · infra-team@orderly
      </div>
      <div style={{ fontFamily:'var(--display)', fontSize:36, color:'var(--ink)', marginBottom:6, fontStyle:'italic' }}>
        A custom set, drawn for this site.
      </div>
      <div style={{ fontFamily:'var(--serif)', fontSize:15, color:'var(--mute)', marginBottom:24, maxWidth:520 }}>
        Bring your own .svg or .png. Errolan renders them at any size, in line with prose, in reactions, and in the moderation studio. Up to 96 per site.
      </div>
    </div>
    <div style={{ display:'grid', gridTemplateColumns:'repeat(12, 1fr)', gap:16, alignItems:'center' }}>
      {EMOJI_PACK.map(e=>(
        <div key={e.code} style={{ display:'flex', flexDirection:'column', alignItems:'center', gap:6 }}>
          <EmojiSVG name={e.code} size={36} />
          <div style={{ fontFamily:'var(--mono)', fontSize:9, color:'var(--mute)', letterSpacing:'.04em' }}>:{e.code}:</div>
        </div>
      ))}
    </div>
  </div>
);

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
