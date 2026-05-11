// dashboard.jsx — Admin moderation view. Same editorial language.
// Three-pane: queue (left) · focused thread (center) · site/emoji pack (right).

const Dashboard = () => {
  const [active, setActive] = React.useState('q2');

  const queue = [
    { id:'q1', author:'newcommer42', avatar:'N', flag:'spam', preview:'Check out my newsletter where I write about exactly this kind of distributed systems prob…', site:'orderly.dev', when:'4m', score:0.91 },
    { id:'q2', author:'alanmoss',    avatar:'A', flag:'flagged', preview:'I want to push back gently. "Eventually consistent" is a real, useful guarantee — the failu…', site:'orderly.dev', when:'12m', score:0.42, flaggedBy:'@miriam-yu' },
    { id:'q3', author:'h_ranjit',    avatar:'H', flag:'first-time', preview:'First time commenting here. Did you consider sticking a Postgres LISTEN/NOTIFY in the middle?', site:'orderly.dev', when:'31m', score:0.08 },
    { id:'q4', author:'anon-3219',   avatar:'?', flag:'toxicity', preview:'this is the dumbest take ive read all wee…', site:'orderly.dev', when:'58m', score:0.86 },
    { id:'q5', author:'tess-lin',    avatar:'T', flag:'first-time', preview:'The honesty about the "did you mean to refresh?" toast is the best part of this post.', site:'orderly.dev', when:'1h', score:0.04 },
    { id:'q6', author:'kpetrov',     avatar:'K', flag:'code-block', preview:'For folks who want this pattern without ripping the queue out, an outbox + same-txn read…', site:'orderly.dev', when:'2h', score:0.05 },
  ];

  const activeItem = queue.find(q=>q.id===active) || queue[0];

  return (
    <div style={{ background:'var(--paper)', minHeight:'100%', fontFamily:'var(--serif)' }}>
      <DashMasthead />
      <div style={{ display:'grid', gridTemplateColumns:'320px 1fr 320px', gap:0, minHeight:'calc(100% - 70px)' }}>
        {/* QUEUE */}
        <div style={{ borderRight:'1px solid var(--rule)', overflow:'auto' }}>
          <div style={{ padding:'18px 22px', borderBottom:'1px solid var(--rule-soft)', display:'flex', justifyContent:'space-between', alignItems:'baseline' }}>
            <div>
              <div style={{ fontFamily:'var(--display)', fontSize:22 }}>Moderation queue</div>
              <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)', marginTop:2 }}>
                {queue.length} pending · 1 escalated
              </div>
            </div>
            <button style={{ fontFamily:'var(--mono)', fontSize:11, padding:'5px 10px', background:'transparent', border:'1px solid var(--rule)', cursor:'pointer' }}>Filter</button>
          </div>
          <div>
            {queue.map(q=>(
              <button key={q.id} onClick={()=>setActive(q.id)} style={{
                display:'block', width:'100%', textAlign:'left',
                padding:'14px 22px', background: active===q.id ? 'var(--paper-2)' : 'transparent',
                border:'none', borderBottom:'1px solid var(--rule-soft)', cursor:'pointer',
                borderLeft: active===q.id ? '3px solid var(--accent)' : '3px solid transparent',
              }}>
                <div style={{ display:'flex', alignItems:'center', gap:8, marginBottom:6 }}>
                  <Avatar letter={q.avatar} />
                  <span style={{ fontFamily:'var(--mono)', fontSize:12, fontWeight:600 }}>@{q.author}</span>
                  <span style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)', marginLeft:'auto' }}>{q.when}</span>
                </div>
                <div style={{ fontFamily:'var(--serif)', fontSize:14, lineHeight:1.45, color:'var(--ink-2)', marginBottom:8 }}>
                  {q.preview}
                </div>
                <div style={{ display:'flex', alignItems:'center', gap:6, fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.1em', textTransform:'uppercase' }}>
                  <FlagPill flag={q.flag} />
                  <ScoreBar score={q.score} />
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* FOCUSED */}
        <div style={{ padding:'32px 40px', overflow:'auto' }}>
          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)' }}>
            {activeItem.site} · on “We removed our message queue…” · paragraph 3
          </div>
          <div style={{ display:'flex', alignItems:'center', gap:14, marginTop:14, marginBottom:18 }}>
            <Avatar letter={activeItem.avatar} large />
            <div>
              <div style={{ fontFamily:'var(--display)', fontSize:24, color:'var(--ink)' }}>@{activeItem.author}</div>
              <div style={{ fontFamily:'var(--mono)', fontSize:11, color:'var(--mute)' }}>2 prior comments · joined 2024 · trust 0.71</div>
            </div>
            <FlagPill flag={activeItem.flag} large />
          </div>
          <div style={{ borderTop:'1px solid var(--rule)', borderBottom:'1px solid var(--rule)', padding:'24px 0', margin:'8px 0 24px' }}>
            <p style={{ fontFamily:'var(--serif)', fontSize:19, lineHeight:1.65, color:'var(--ink)', margin:0 }}>
              I want to push back gently. <span style={{ background:'var(--highlight-soft)' }}>“Eventually consistent” is a real, useful guarantee</span> — the failure mode here is that the team didn’t articulate which reads needed which guarantee. A queue isn’t the villain.
            </p>
          </div>

          {/* Context */}
          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)', marginBottom:10 }}>
            Flagged by
          </div>
          <div style={{ padding:'12px 16px', background:'var(--paper-2)', borderLeft:'2px solid var(--accent)', marginBottom:24 }}>
            <div style={{ fontFamily:'var(--mono)', fontSize:12, fontWeight:600, marginBottom:4 }}>@miriam-yu · 1 report</div>
            <div style={{ fontFamily:'var(--serif)', fontSize:14, fontStyle:'italic', color:'var(--mute)' }}>
              “Not toxic, but I disagree with the framing and the reply isn’t adding to the thread. Up to you.”
            </div>
          </div>

          {/* Signals */}
          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)', marginBottom:10 }}>
            Signals
          </div>
          <div style={{ display:'grid', gridTemplateColumns:'repeat(4,1fr)', gap:1, background:'var(--rule)', border:'1px solid var(--rule)', marginBottom:32 }}>
            {[
              ['Toxicity', '0.04', 'low'],
              ['Spam', '0.02', 'low'],
              ['Heat', '0.71', 'high'],
              ['On-topic', '0.93', 'high'],
            ].map(([k,v,t])=>(
              <div key={k} style={{ padding:'14px 16px', background:'var(--paper)' }}>
                <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.12em', textTransform:'uppercase', color:'var(--mute)' }}>{k}</div>
                <div style={{ fontFamily:'var(--display)', fontSize:28, color: t==='high' ? 'var(--accent)' : 'var(--ink)', marginTop:4 }}>{v}</div>
              </div>
            ))}
          </div>

          {/* Actions */}
          <div style={{ display:'flex', gap:10, flexWrap:'wrap' }}>
            <Action primary label="Approve" hotkey="A" />
            <Action label="Approve + pin" hotkey="P" />
            <Action label="Hide" hotkey="H" />
            <Action danger label="Remove" hotkey="R" />
            <Action label="Shadow" hotkey="S" />
            <Action label="Mute author" hotkey="M" />
          </div>
        </div>

        {/* RIGHT — SITE / EMOJI PACK */}
        <div style={{ borderLeft:'1px solid var(--rule)', padding:'24px 22px', background:'var(--paper)' }}>
          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)', marginBottom:6 }}>Site</div>
          <div style={{ fontFamily:'var(--display)', fontSize:22, marginBottom:14 }}>orderly.dev</div>
          <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:1, background:'var(--rule)', border:'1px solid var(--rule)', marginBottom:24 }}>
            {[
              ['Today','148','comments'],
              ['Threads','42','active'],
              ['Pack','24','emojis'],
              ['Mods','3','active'],
            ].map(([a,b,c])=>(
              <div key={a} style={{ padding:'12px 14px', background:'var(--paper)' }}>
                <div style={{ fontFamily:'var(--mono)', fontSize:9, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)' }}>{a}</div>
                <div style={{ fontFamily:'var(--display)', fontSize:24, color:'var(--ink)' }}>{b}</div>
                <div style={{ fontFamily:'var(--mono)', fontSize:9, color:'var(--mute)' }}>{c}</div>
              </div>
            ))}
          </div>

          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)', marginBottom:10 }}>Custom emoji pack</div>
          <div style={{ display:'grid', gridTemplateColumns:'repeat(4, 1fr)', gap:6, marginBottom:14 }}>
            {EMOJI_PACK.map(e=>(
              <div key={e.code} style={{ aspectRatio:'1', display:'grid', placeItems:'center', background:'var(--paper-2)', position:'relative', borderRadius:2 }}>
                <EmojiSVG name={e.code} size={24} />
              </div>
            ))}
            <div style={{ aspectRatio:'1', display:'grid', placeItems:'center', background:'transparent', border:'1px dashed var(--rule)', borderRadius:2, fontFamily:'var(--mono)', fontSize:18, color:'var(--mute)' }}>+</div>
          </div>
          <div style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)', marginBottom:18 }}>Drop a .png or .svg to add — max 64KB</div>

          <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)', marginBottom:10 }}>Top reactions today</div>
          {[['ship-it', 142],['this', 98],['hot-take', 71],['skull', 38]].map(([code, count], i)=>(
            <div key={code} style={{ display:'flex', alignItems:'center', gap:10, padding:'6px 0', borderBottom: i<3 ? '1px solid var(--rule-soft)' : 'none' }}>
              <EmojiSVG name={code} size={18} />
              <span style={{ fontFamily:'var(--mono)', fontSize:12, color:'var(--ink-2)' }}>:{code}:</span>
              <span style={{ flex:1, marginLeft:8, height:1, background:'var(--rule-soft)' }} />
              <span style={{ fontFamily:'var(--mono)', fontSize:12, color:'var(--mute)' }}>{count}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

const DashMasthead = () => (
  <header style={{ borderBottom:'1px solid var(--rule)', padding:'14px 28px', display:'flex', alignItems:'center', justifyContent:'space-between', background:'var(--paper)' }}>
    <div style={{ display:'flex', alignItems:'baseline', gap:18 }}>
      <div style={{ fontFamily:'var(--display)', fontSize:24 }}>Errolan<span style={{ color:'var(--accent)' }}>.</span></div>
      <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)' }}>Studio</div>
    </div>
    <nav style={{ display:'flex', gap:22, fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.1em', textTransform:'uppercase' }}>
      {['Queue','Threads','Readers','Pack','Settings'].map((n,i)=>(
        <a key={n} style={{ color: i===0 ? 'var(--ink)' : 'var(--mute)', textDecoration:'none', borderBottom: i===0 ? '1px solid var(--accent)' : 'none', paddingBottom:2 }}>{n}</a>
      ))}
    </nav>
    <div style={{ display:'flex', alignItems:'center', gap:10 }}>
      <span style={{ fontFamily:'var(--mono)', fontSize:11, color:'var(--mute)' }}>⌘K</span>
      <div style={{ width:30, height:30, borderRadius:'50%', background:'var(--ink)', color:'var(--paper)', display:'grid', placeItems:'center', fontFamily:'var(--display)', fontSize:14 }}>H</div>
    </div>
  </header>
);

const FlagPill = ({ flag, large }) => {
  const map = {
    'spam':       ['Spam',         'var(--accent)'],
    'toxicity':   ['Toxicity',     'var(--accent)'],
    'flagged':    ['User-flagged', 'var(--plum)'],
    'first-time': ['First-time',   'var(--blue)'],
    'code-block': ['Has code',     'var(--green)'],
  };
  const [label, color] = map[flag] || [flag, 'var(--mute)'];
  return (
    <span style={{
      display:'inline-flex', alignItems:'center', gap:5,
      padding: large ? '4px 10px' : '2px 7px',
      fontFamily:'var(--mono)', fontSize: large ? 10 : 9, letterSpacing:'.12em', textTransform:'uppercase',
      color, border:`1px solid ${color}`, borderRadius:2,
    }}>
      <span style={{ width:5, height:5, borderRadius:'50%', background:color }} />
      {label}
    </span>
  );
};

const ScoreBar = ({ score }) => {
  const color = score > 0.7 ? 'var(--accent)' : score > 0.4 ? 'var(--plum)' : 'var(--green)';
  return (
    <span style={{ marginLeft:'auto', display:'flex', alignItems:'center', gap:6, fontFamily:'var(--mono)', fontSize:10, color }}>
      <span style={{ width:36, height:4, background:'var(--rule-soft)', position:'relative' }}>
        <span style={{ position:'absolute', left:0, top:0, height:'100%', width:`${score*100}%`, background:color }} />
      </span>
      {score.toFixed(2)}
    </span>
  );
};

const Action = ({ label, hotkey, primary, danger }) => (
  <button style={{
    display:'inline-flex', alignItems:'center', gap:8,
    padding:'10px 16px', cursor:'pointer',
    background: primary ? 'var(--ink)' : danger ? 'var(--paper)' : 'var(--paper)',
    color: primary ? 'var(--paper)' : danger ? 'var(--accent)' : 'var(--ink)',
    border: primary ? 'none' : `1px solid ${danger ? 'var(--accent)' : 'var(--rule)'}`,
    fontFamily:'var(--mono)', fontSize:12, letterSpacing:'.08em',
  }}>
    {label}
    <span style={{ fontFamily:'var(--mono)', fontSize:10, padding:'1px 5px', background: primary ? 'rgba(255,255,255,.18)' : 'var(--paper-2)', color: primary ? 'var(--paper)' : 'var(--mute)', borderRadius:2 }}>{hotkey}</span>
  </button>
);

Object.assign(window, { Dashboard });
