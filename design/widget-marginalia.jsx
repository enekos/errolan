// widget-marginalia.jsx — Direction A.
// Comments live in the right margin, anchored to specific paragraphs.
// Library / annotated-book feel. Dense. The thing nobody else does.

const Marginalia = ({ tweaks = {} }) => {
  const [active, setActive] = React.useState('c2');
  const [hoverPid, setHoverPid] = React.useState(null);
  const articleRef = React.useRef(null);
  const [pidPositions, setPidPositions] = React.useState({});

  // After render, measure each paragraph's top offset so we can pin
  // marginalia stamps next to the right paragraph.
  React.useLayoutEffect(()=>{
    if (!articleRef.current) return;
    const root = articleRef.current;
    const containerTop = root.getBoundingClientRect().top;
    const pos = {};
    root.querySelectorAll('[data-pid]').forEach(el=>{
      pos[el.dataset.pid] = el.getBoundingClientRect().top - containerTop;
    });
    setPidPositions(pos);
  }, []);

  // Group comments by anchor paragraph
  const byAnchor = {};
  COMMENTS.forEach(c => { (byAnchor[c.anchor] ||= []).push(c); });

  const activeComment = COMMENTS.find(c => c.id === active) || COMMENTS[0];

  return (
    <div style={{ background:'var(--paper)', minHeight:'100%', fontFamily:'var(--serif)', color:'var(--ink)' }}>
      <Masthead />
      <div style={{ display:'grid', gridTemplateColumns:'1fr 320px 380px', gap:0, maxWidth:1280, margin:'0 auto', padding:'40px 56px 80px', position:'relative' }}>

        {/* ARTICLE COLUMN */}
        <div ref={articleRef} style={{ paddingRight:36, borderRight:'1px solid var(--rule-soft)', position:'relative' }}>
          <Article
            onParagraph={(p)=>({
              onMouseEnter:()=>setHoverPid(p.id),
              onMouseLeave:()=>setHoverPid(prev => prev === p.id ? null : prev),
              style: (byAnchor[p.id] && (hoverPid === p.id || byAnchor[p.id].some(c=>c.id===active))) ? { background:'linear-gradient(to right, var(--highlight-soft) 0%, var(--highlight-soft) 70%, transparent 100%)', boxShadow:'-12px 0 0 var(--highlight-soft)' } : {},
            })}
          />
        </div>

        {/* MARGINALIA STAMP COLUMN — anchors + reaction sigils next to paragraphs */}
        <div style={{ position:'relative', paddingLeft:18, paddingRight:18 }}>
          {Object.entries(byAnchor).map(([pid, comments])=>{
            const top = (pidPositions[pid] ?? 0);
            const totalReplies = comments.reduce((s,c)=>s + 1 + (c.replies?.length||0), 0);
            const allRx = {};
            comments.forEach(c=>{
              Object.entries(c.reactions||{}).forEach(([k,v])=>{ allRx[k]=(allRx[k]||0)+v; });
              (c.replies||[]).forEach(r=>{
                Object.entries(r.reactions||{}).forEach(([k,v])=>{ allRx[k]=(allRx[k]||0)+v; });
              });
            });
            const topRx = Object.entries(allRx).sort((a,b)=>b[1]-a[1]).slice(0,3);
            const isActive = comments.some(c=>c.id===active);
            return (
              <button
                key={pid}
                onClick={()=>setActive(comments[0].id)}
                onMouseEnter={()=>setHoverPid(pid)}
                onMouseLeave={()=>setHoverPid(null)}
                style={{
                  position:'absolute', top, left:18, right:18,
                  display:'flex', flexDirection:'column', alignItems:'flex-start', gap:6,
                  padding:'8px 10px', textAlign:'left', cursor:'pointer',
                  background: isActive ? 'var(--paper-2)' : 'transparent',
                  border:'none', borderLeft: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                  borderRadius:0, color:'var(--ink)', fontFamily:'var(--serif)',
                  transition:'background .15s',
                }}
              >
                <div style={{ display:'flex', alignItems:'center', gap:6 }}>
                  {topRx.map(([code,count])=>(
                    <span key={code} style={{ display:'inline-flex', alignItems:'center', gap:3, fontFamily:'var(--mono)', fontSize:11, color:'var(--mute)' }}>
                      <EmojiSVG name={code} size={15} />
                      <span>{count}</span>
                    </span>
                  ))}
                </div>
                <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.12em', textTransform:'uppercase', color:'var(--mute)' }}>
                  {totalReplies} {totalReplies===1?'note':'notes'} · {comments.length===1?'1 thread':comments.length+' threads'}
                </div>
                <div style={{ fontFamily:'var(--serif)', fontSize:13, fontStyle:'italic', color:'var(--ink-2)', lineHeight:1.4, overflow:'hidden', display:'-webkit-box', WebkitLineClamp:2, WebkitBoxOrient:'vertical' }}>
                  “{comments[0].body.slice(0, 80)}…”
                </div>
                <div style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute-2)' }}>
                  @{comments[0].author}{comments.length>1?` +${comments.length-1}`:''}
                </div>
              </button>
            );
          })}
          {/* anchor line indicating which paragraph is selected */}
          {pidPositions[activeComment.anchor]!=null && (
            <div style={{ position:'absolute', top:pidPositions[activeComment.anchor]+10, left:-1, width:18, height:1, background:'var(--accent)' }} />
          )}
        </div>

        {/* COMMENT READER COLUMN — focused thread */}
        <div style={{ paddingLeft:28, borderLeft:'1px solid var(--rule-soft)', position:'sticky', top:24, alignSelf:'start', maxHeight:'calc(100vh - 100px)', overflow:'auto' }}>
          <FocusedThread comment={activeComment} />
          <Composer />
        </div>
      </div>
    </div>
  );
};

const Masthead = () => (
  <header style={{ borderBottom:'1px solid var(--rule)', padding:'18px 56px', display:'flex', alignItems:'center', justifyContent:'space-between', background:'var(--paper)' }}>
    <div style={{ display:'flex', alignItems:'baseline', gap:18 }}>
      <div style={{ fontFamily:'var(--display)', fontSize:28, lineHeight:1, color:'var(--ink)' }}>
        Errolan<span style={{ color:'var(--accent)' }}>.</span>
      </div>
      <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.18em', textTransform:'uppercase', color:'var(--mute)' }}>
        Comments, with character
      </div>
    </div>
    <nav style={{ display:'flex', gap:24, fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.1em', textTransform:'uppercase', color:'var(--mute)' }}>
      <a style={{ color:'var(--ink)', textDecoration:'none' }}>Read</a>
      <a style={{ color:'var(--mute)', textDecoration:'none' }}>About</a>
      <a style={{ color:'var(--mute)', textDecoration:'none' }}>Archive</a>
    </nav>
  </header>
);

const FocusedThread = ({ comment }) => {
  return (
    <div>
      <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--accent)', marginBottom:18 }}>
        Marginalia · paragraph {comment.anchor}
      </div>
      <CommentBlock c={comment} top />
      {comment.replies?.map(r => (
        <div key={r.id} style={{ marginLeft:14, paddingLeft:14, borderLeft:'1px solid var(--rule)' }}>
          <CommentBlock c={r} />
        </div>
      ))}
    </div>
  );
};

const CommentBlock = ({ c, top = false }) => {
  return (
    <div style={{ marginBottom:22, paddingBottom:18, borderBottom: top ? '1px solid var(--rule-soft)':'none' }}>
      <div style={{ display:'flex', alignItems:'center', gap:8, marginBottom:8 }}>
        <Avatar letter={c.avatar} role={c.role} />
        <div style={{ flex:1 }}>
          <div style={{ fontFamily:'var(--mono)', fontSize:12, fontWeight:600, color:'var(--ink)' }}>
            @{c.author}
            {c.role==='author' && <span style={{ marginLeft:6, padding:'1px 6px', background:'var(--accent)', color:'var(--paper)', fontSize:9, letterSpacing:'.1em', textTransform:'uppercase', borderRadius:2 }}>Author</span>}
          </div>
          <div style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)' }}>{c.time} ago</div>
        </div>
      </div>
      <div style={{ fontFamily:'var(--serif)', fontSize:16, lineHeight:1.6, color:'var(--ink-2)', marginBottom:10 }}>
        <ParseEmoji text={c.body} size={16} />
      </div>
      {c.code && (
        <pre style={{ fontFamily:'var(--mono)', fontSize:11.5, lineHeight:1.5, background:'var(--paper-2)', borderLeft:'2px solid var(--ink)', padding:'10px 12px', margin:'8px 0 12px', overflow:'auto', borderRadius:2 }}>
          <code dangerouslySetInnerHTML={{__html: window.highlightTs ? window.highlightTs(c.code) : c.code}} />
        </pre>
      )}
      <ReactionStrip reactions={c.reactions} />
    </div>
  );
};

const Avatar = ({ letter, role }) => (
  <div style={{
    width:30, height:30, borderRadius:'50%',
    background: role==='author' ? 'var(--accent)' : 'var(--paper-2)',
    color: role==='author' ? 'var(--paper)' : 'var(--ink)',
    border:'1px solid var(--rule)',
    display:'grid', placeItems:'center',
    fontFamily:'var(--display)', fontSize:15
  }}>{letter}</div>
);

const ReactionStrip = ({ reactions }) => {
  const entries = Object.entries(reactions || {});
  if (!entries.length) return null;
  return (
    <div style={{ display:'flex', flexWrap:'wrap', gap:4, marginTop:6 }}>
      {entries.map(([code, count])=>(
        <button key={code} style={{
          display:'inline-flex', alignItems:'center', gap:4,
          background:'var(--paper-2)', border:'1px solid var(--rule)', borderRadius:2,
          padding:'2px 8px', cursor:'pointer', fontFamily:'var(--mono)', fontSize:11, color:'var(--ink-2)',
        }}>
          <EmojiSVG name={code} size={14} />
          <span>{count}</span>
        </button>
      ))}
      <button style={{
        display:'inline-flex', alignItems:'center', gap:2, background:'transparent', border:'1px dashed var(--rule)',
        borderRadius:2, padding:'2px 8px', cursor:'pointer', fontFamily:'var(--mono)', fontSize:11, color:'var(--mute)',
      }}>+</button>
    </div>
  );
};

const Composer = () => (
  <div style={{ marginTop:24, padding:14, border:'1px solid var(--rule)', background:'var(--paper)' }}>
    <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)', marginBottom:8 }}>
      Add a note to paragraph 3
    </div>
    <div style={{ minHeight:60, fontFamily:'var(--serif)', fontStyle:'italic', color:'var(--mute-2)', fontSize:15 }}>
      Write in the margin…
    </div>
    <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', marginTop:8, paddingTop:8, borderTop:'1px solid var(--rule-soft)' }}>
      <div style={{ display:'flex', gap:6 }}>
        {EMOJI_PACK.slice(0,5).map(e=>(
          <button key={e.code} style={{ background:'transparent', border:'none', cursor:'pointer', padding:2 }} title={e.label}>
            <EmojiSVG name={e.code} size={18} />
          </button>
        ))}
        <button style={{ background:'transparent', border:'none', cursor:'pointer', padding:2, fontFamily:'var(--mono)', fontSize:11, color:'var(--mute)' }}>+24</button>
      </div>
      <button style={{ fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.1em', textTransform:'uppercase', padding:'6px 14px', background:'var(--ink)', color:'var(--paper)', border:'none', cursor:'pointer' }}>
        Post note
      </button>
    </div>
  </div>
);

Object.assign(window, { Marginalia });
