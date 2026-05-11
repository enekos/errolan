// widget-cadence.jsx — Direction B.
// Flat-ish conversation "river" with a left-side time spine, AI thread
// summary at the top, generous editorial whitespace, and a docked emoji
// shelf that drags onto comments to react. Replies appear as quote-cards
// (no deep indentation) so the conversation reads top-to-bottom like a
// proper editorial transcript.

const Cadence = ({ tweaks = {} }) => {
  const [pickerOpen, setPickerOpen] = React.useState(false);
  const [sortBy, setSortBy] = React.useState('best');

  // Flatten replies into the same vertical river, but tagged with parent.
  const flat = [];
  COMMENTS.forEach(c => {
    flat.push({ ...c, _kind:'top' });
    (c.replies||[]).forEach(r => flat.push({ ...r, _kind:'reply', _parent: c }));
  });

  return (
    <div style={{ background:'var(--paper)', minHeight:'100%', fontFamily:'var(--serif)' }}>
      <Masthead />
      <div style={{ maxWidth:780, margin:'0 auto', padding:'48px 24px 80px' }}>
        <Article />

        {/* Divider between article and comments */}
        <div style={{ margin:'56px 0 0', display:'flex', alignItems:'center', gap:14 }}>
          <div style={{ height:1, flex:1, background:'var(--rule)' }} />
          <div style={{ fontFamily:'var(--display)', fontSize:24, fontStyle:'italic', color:'var(--ink)' }}>
            The conversation
          </div>
          <div style={{ height:1, flex:1, background:'var(--rule)' }} />
        </div>

        {/* Thread weather — sentiment summary */}
        <ThreadWeather />

        {/* Sort + count */}
        <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between', margin:'24px 0 12px', paddingBottom:12, borderBottom:'1px solid var(--rule-soft)' }}>
          <div style={{ fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.14em', textTransform:'uppercase', color:'var(--mute)' }}>
            {flat.length} notes · 7 readers writing
          </div>
          <div style={{ display:'flex', gap:0, fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.1em', textTransform:'uppercase' }}>
            {['best','newest','oldest'].map(s=>(
              <button key={s} onClick={()=>setSortBy(s)} style={{
                background:'transparent', border:'none', padding:'4px 10px', cursor:'pointer',
                color: sortBy===s ? 'var(--ink)' : 'var(--mute)',
                borderBottom: sortBy===s ? '1px solid var(--accent)' : '1px solid transparent',
              }}>{s}</button>
            ))}
          </div>
        </div>

        {/* Top-of-thread composer */}
        <TopComposer />

        {/* The river */}
        <div style={{ position:'relative', paddingLeft:48, marginTop:32 }}>
          {/* Spine */}
          <div style={{ position:'absolute', left:18, top:8, bottom:60, width:1, background:'linear-gradient(to bottom, var(--rule) 0%, var(--rule) 90%, transparent 100%)' }} />
          <div style={{ position:'absolute', left:11, top:0, fontFamily:'var(--mono)', fontSize:9, color:'var(--mute)', letterSpacing:'.14em', textTransform:'uppercase', writingMode:'vertical-rl', transform:'rotate(180deg)' }}>
            NOW →
          </div>

          {flat.map((c, i)=>(
            <CadenceComment key={c.id} c={c} index={i} total={flat.length} />
          ))}

          {/* "writing now" pulse */}
          <div style={{ position:'relative', marginLeft:-30, marginTop:18, display:'flex', alignItems:'center', gap:14 }}>
            <span style={{ width:9, height:9, borderRadius:'50%', background:'var(--accent)', boxShadow:'0 0 0 4px rgba(183,53,29,.15)', animation:'pulse 1.6s ease-in-out infinite' }} />
            <span style={{ fontFamily:'var(--serif)', fontStyle:'italic', fontSize:14, color:'var(--mute)' }}>
              @alanmoss is writing a reply…
            </span>
          </div>
        </div>
      </div>

      {/* Floating emoji shelf */}
      <EmojiShelf open={pickerOpen} onToggle={()=>setPickerOpen(v=>!v)} />

      <style>{`
        @keyframes pulse{0%,100%{transform:scale(1);opacity:1}50%{transform:scale(1.4);opacity:.6}}
      `}</style>
    </div>
  );
};

const ThreadWeather = () => {
  // Aggregate the dominant reactions across the whole thread.
  const tally = {};
  COMMENTS.forEach(c=>{
    Object.entries(c.reactions||{}).forEach(([k,v])=>tally[k]=(tally[k]||0)+v);
    (c.replies||[]).forEach(r=>{
      Object.entries(r.reactions||{}).forEach(([k,v])=>tally[k]=(tally[k]||0)+v);
    });
  });
  const top = Object.entries(tally).sort((a,b)=>b[1]-a[1]).slice(0,5);
  const total = top.reduce((s,[,v])=>s+v,0);

  return (
    <div style={{ marginTop:32, padding:'22px 28px', background:'var(--paper-2)', borderTop:'2px solid var(--ink)', borderBottom:'1px solid var(--rule)', position:'relative' }}>
      <div style={{ position:'absolute', top:-9, left:24, padding:'0 8px', background:'var(--paper-2)', fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.18em', textTransform:'uppercase', color:'var(--mute)' }}>
        Thread weather
      </div>
      <div style={{ display:'grid', gridTemplateColumns:'1fr auto', gap:32, alignItems:'center' }}>
        <p style={{ margin:0, fontFamily:'var(--display)', fontSize:21, lineHeight:1.4, fontStyle:'italic', color:'var(--ink)' }}>
          “Mostly enthusiastic agreement, one substantive disagreement from <span style={{ fontStyle:'normal', fontFamily:'var(--mono)', fontSize:14, color:'var(--accent)' }}>@alanmoss</span>, and unusual consensus that the &lsquo;refresh fixed it&rsquo; toast was the funniest part.”
        </p>
        <div style={{ display:'flex', alignItems:'flex-end', gap:14 }}>
          {top.map(([code, count])=>{
            const h = 30 + (count/total)*60;
            return (
              <div key={code} style={{ display:'flex', flexDirection:'column', alignItems:'center', gap:6 }}>
                <div style={{ height:h, width:24, background:'linear-gradient(to top, var(--accent) 0%, var(--accent) 60%, var(--highlight) 100%)', opacity:.18, borderTop:'2px solid var(--accent)' }} />
                <EmojiSVG name={code} size={18} />
                <span style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)' }}>{count}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

const TopComposer = () => {
  const [text, setText] = React.useState('');
  return (
    <div style={{ padding:'18px 0', borderBottom:'1px solid var(--rule-soft)', display:'flex', gap:14, alignItems:'flex-start' }}>
      <div style={{ width:38, height:38, borderRadius:'50%', background:'var(--ink)', color:'var(--paper)', display:'grid', placeItems:'center', fontFamily:'var(--display)', fontSize:18 }}>Y</div>
      <div style={{ flex:1 }}>
        <textarea
          value={text}
          onChange={e=>setText(e.target.value)}
          placeholder="Write a thoughtful note. Markdown welcome. Type : to add an emoji."
          style={{
            width:'100%', minHeight:46, padding:'8px 0',
            border:'none', background:'transparent', resize:'none',
            fontFamily:'var(--serif)', fontSize:17, lineHeight:1.5, color:'var(--ink)',
            outline:'none',
          }}
        />
        <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', paddingTop:8, borderTop:'1px solid var(--rule-soft)' }}>
          <div style={{ display:'flex', gap:14, fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.12em', textTransform:'uppercase', color:'var(--mute)' }}>
            <span>B</span><span style={{ fontStyle:'italic' }}>I</span><span>“ ”</span><span>{ '</>' }</span><span>:emoji:</span><span>@</span>
          </div>
          <div style={{ display:'flex', gap:8 }}>
            <button style={{ fontFamily:'var(--mono)', fontSize:11, padding:'6px 12px', background:'transparent', border:'1px solid var(--rule)', cursor:'pointer', color:'var(--mute)' }}>Preview</button>
            <button style={{ fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.1em', textTransform:'uppercase', padding:'6px 14px', background:'var(--ink)', color:'var(--paper)', border:'none', cursor:'pointer' }}>
              Post note
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

const CadenceComment = ({ c, index, total }) => {
  const isReply = c._kind === 'reply';
  return (
    <div style={{ position:'relative', marginBottom:isReply ? 18 : 28, opacity: 1 - (index/total)*0.05 }}>
      {/* spine dot */}
      <div style={{
        position:'absolute', left:-37, top:14,
        width: isReply ? 6 : 10, height: isReply ? 6 : 10,
        borderRadius:'50%',
        background: c.role==='author' ? 'var(--accent)' : 'var(--ink)',
        border:'2px solid var(--paper)',
        boxShadow:'0 0 0 1px var(--rule)',
      }} />
      {/* time marker */}
      <div style={{ position:'absolute', left:-78, top:13, fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)', width:34, textAlign:'right' }}>
        {c.time}
      </div>

      {/* quoted parent for replies */}
      {isReply && c._parent && (
        <div style={{ marginBottom:8, padding:'6px 12px', borderLeft:'2px solid var(--rule)', background:'var(--paper-2)', fontFamily:'var(--serif)', fontStyle:'italic', fontSize:13, color:'var(--mute)', lineHeight:1.45 }}>
          <span style={{ fontFamily:'var(--mono)', fontStyle:'normal', fontSize:11, color:'var(--ink-2)' }}>@{c._parent.author}</span>
          &nbsp;· {c._parent.body.slice(0,90)}…
        </div>
      )}

      <div style={{ display:'flex', alignItems:'center', gap:10, marginBottom:6 }}>
        <Avatar letter={c.avatar} role={c.role} />
        <span style={{ fontFamily:'var(--mono)', fontSize:12, fontWeight:600, color:'var(--ink)' }}>
          @{c.author}
        </span>
        {c.role==='author' && (
          <span style={{ padding:'1px 6px', background:'var(--accent)', color:'var(--paper)', fontFamily:'var(--mono)', fontSize:9, letterSpacing:'.1em', textTransform:'uppercase' }}>Author</span>
        )}
      </div>

      <div style={{ fontFamily:'var(--serif)', fontSize: isReply ? 15 : 17, lineHeight:1.6, color:'var(--ink-2)' }}>
        <ParseEmoji text={c.body} size={isReply ? 14 : 16} />
      </div>

      {c.code && (
        <pre style={{ fontFamily:'var(--mono)', fontSize:12, lineHeight:1.5, background:'var(--paper-2)', borderLeft:'2px solid var(--ink)', padding:'10px 14px', margin:'10px 0', overflow:'auto' }}>
          <code dangerouslySetInnerHTML={{__html: window.highlightTs ? window.highlightTs(c.code) : c.code}} />
        </pre>
      )}

      <div style={{ display:'flex', alignItems:'center', gap:10, marginTop:10 }}>
        <ReactionStripCadence reactions={c.reactions} />
        <div style={{ display:'flex', gap:14, fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.12em', textTransform:'uppercase', color:'var(--mute)' }}>
          <button style={{ background:'none', border:'none', color:'inherit', cursor:'pointer', fontFamily:'inherit', fontSize:'inherit', letterSpacing:'inherit', textTransform:'inherit', padding:0 }}>Reply</button>
          <button style={{ background:'none', border:'none', color:'inherit', cursor:'pointer', fontFamily:'inherit', fontSize:'inherit', letterSpacing:'inherit', textTransform:'inherit', padding:0 }}>Quote</button>
        </div>
      </div>
    </div>
  );
};

const ReactionStripCadence = ({ reactions }) => {
  const entries = Object.entries(reactions || {});
  if (!entries.length) return (
    <button style={{ background:'transparent', border:'none', padding:'2px 6px', cursor:'pointer', color:'var(--mute)', fontFamily:'var(--mono)', fontSize:11 }}>+ react</button>
  );
  return (
    <div style={{ display:'flex', alignItems:'center', gap:8 }}>
      {entries.map(([code, count])=>(
        <button key={code} style={{
          display:'inline-flex', alignItems:'center', gap:4,
          background:'transparent', border:'none', cursor:'pointer',
          padding:'2px 4px', fontFamily:'var(--mono)', fontSize:11, color:'var(--ink-2)',
          borderBottom:'1px dotted var(--rule)',
        }}>
          <EmojiSVG name={code} size={15} />
          <span>{count}</span>
        </button>
      ))}
    </div>
  );
};

const EmojiShelf = ({ open, onToggle }) => {
  return (
    <div style={{ position:'fixed', right:24, bottom:24, zIndex:10 }}>
      {open && (
        <div style={{
          position:'absolute', right:0, bottom:60,
          width:300, padding:14, background:'var(--paper)',
          border:'1px solid var(--rule)', boxShadow:'0 12px 40px rgba(20,15,5,.12)',
        }}>
          <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', marginBottom:10 }}>
            <div style={{ fontFamily:'var(--mono)', fontSize:10, letterSpacing:'.16em', textTransform:'uppercase', color:'var(--mute)' }}>
              Site emoji pack · drag to react
            </div>
            <div style={{ fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)' }}>12/24</div>
          </div>
          <div style={{ display:'grid', gridTemplateColumns:'repeat(6,1fr)', gap:8 }}>
            {EMOJI_PACK.map(e=>(
              <div key={e.code} title={`:${e.code}:`} style={{
                aspectRatio:'1', display:'grid', placeItems:'center',
                background:'var(--paper-2)', cursor:'grab', borderRadius:2,
              }}>
                <EmojiSVG name={e.code} size={22} />
              </div>
            ))}
          </div>
          <div style={{ marginTop:10, paddingTop:10, borderTop:'1px solid var(--rule-soft)', fontFamily:'var(--mono)', fontSize:10, color:'var(--mute)' }}>
            Custom pack: <span style={{ color:'var(--accent)' }}>infra-team@orderly</span>
          </div>
        </div>
      )}
      <button onClick={onToggle} style={{
        display:'flex', alignItems:'center', gap:8,
        padding:'10px 14px', background:'var(--ink)', color:'var(--paper)',
        border:'none', cursor:'pointer', boxShadow:'0 6px 20px rgba(20,15,5,.18)',
        fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.12em', textTransform:'uppercase',
      }}>
        <EmojiSVG name="hot-take" size={16} />
        Emoji pack
      </button>
    </div>
  );
};

Object.assign(window, { Cadence });
