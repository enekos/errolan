
// ===== emoji.jsx =====
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


// ===== host-article.jsx =====
// host-article.jsx — Shared tech-blog substrate for both widget directions.

const ARTICLE = {
  kicker: 'INFRASTRUCTURE',
  title: 'We removed our message queue and nothing exploded',
  dek: 'A long story about read-your-writes consistency, a 600-line consumer that became 12, and the unglamorous engineering of \u201Cobvious in retrospect.\u201D',
  author: 'Hana Okafor',
  authorRole: 'Staff engineer, platform',
  date: 'May 6, 2026',
  readTime: '11 min',
  paragraphs: [
    { id:'p1', kind:'p', text:'Last quarter we removed the message queue from our checkout service. We replaced about six hundred lines of consumer plumbing with a twelve-line function. Nothing exploded. This is the longer version of what we found.' },
    { id:'p2', kind:'p', text:'The history is mundane. Three years ago, when the team was four engineers, we punted on a tricky synchronous call by sticking a queue between two services. It worked. It was also the start of a long story about read-your-writes consistency that ended up costing us more than the original synchronous call would have.' },
    { id:'p3', kind:'p', text:'None of this is an indictment of message queues. It is an indictment of our use of a message queue for a workload it was never the right tool for. The shape of the workload mattered more than the shape of the tool.' },
    { id:'p4', kind:'h2', text:'The original architecture' },
    { id:'p5', kind:'p', text:'Order writes went to the primary database, then a change-data-capture stream fanned them out to a queue, and a consumer materialized them into a read model. The read model was the thing the checkout UI actually read from.' },
    { id:'p6', kind:'code', lang:'ts', text:'// the original consumer, roughly\nasync function onOrderCreated(evt: OrderEvent) {\n  const order = await db.orders.get(evt.id);\n  const lines = await db.lines.where({ orderId: evt.id }).all();\n  const totals = await pricing.totals(order, lines);\n  await readModel.upsert(toReadRow(order, lines, totals));\n}' },
    { id:'p7', kind:'p', text:'On a quiet day, end-to-end latency was around forty milliseconds. On a less quiet day \u2014 a deploy, a slow consumer, a backed-up partition \u2014 it was whatever the consumer\u2019s lag happened to be. The UI did not know which day it was.' },
    { id:'p8', kind:'p', text:'We tried the usual things. Read-after-write polling. A short cache. A "did you mean to refresh?" toast which I am still embarrassed about. Each one papered over the same shape of bug and left a different one behind.' },
    { id:'p9', kind:'h2', text:'What we actually wanted' },
    { id:'p10', kind:'p', text:'What we actually wanted was a strongly-consistent read of a small, bounded view of the data, on the request path, on the same database. We were using a queue because someone had once said the word "decoupling" out loud and nobody had revisited it.' },
  ],
};

const COMMENTS = [
  { id:'c1', anchor:'p2', author:'dthorne', avatar:'D', role:null,
    body:'This matches my experience at $LASTJOB \u2014 the read-after-write story is what kills you in practice. We ended up doing the same trick with a write-through cache and it was an immediate quality-of-life upgrade for the team.',
    time:'2h', reactions:{ 'ship-it':4, 'this':3, 'plus-one':2 },
    replies:[
      { id:'c1r1', author:'hana-okafor', avatar:'H', role:'author', body:'Yeah \u2014 the team morale piece is underrated. We stopped getting paged at 3am about \u201Crefresh fixed it\u201D bugs and that did more for the codebase than the patch did.', time:'2h', reactions:{ 'this':6, 'plus-one':3 } },
    ]},
  { id:'c2', anchor:'p3', author:'miriam-yu', avatar:'M', role:null,
    body:'Hot take: \u201Ceventual consistency\u201D is a synonym for \u201Cwe don\u2019t know when this will be consistent and we\u2019re not going to tell you.\u201D Every time I see it in a design doc I add three weeks to the estimate.',
    time:'3h', reactions:{ 'hot-take':12, 'spicy':6, 'skull':2, 'tea':4 },
    replies:[
      { id:'c2r1', author:'joeyc', avatar:'J', role:null, body:'@miriam-yu galaxy-brain energy. saving this for the next design review.', time:'2h', reactions:{ 'galaxy-brain':3 } },
      { id:'c2r2', author:'alanmoss', avatar:'A', role:null, body:'I want to push back gently. \u201CEventually consistent\u201D is a real, useful guarantee \u2014 the failure mode here is that the team didn\u2019t articulate which reads needed which guarantee. A queue isn\u2019t the villain.', time:'1h', reactions:{ 'this':5, 'wrong':1 } },
    ]},
  { id:'c3', anchor:'p6', author:'kpetrov', avatar:'K', role:null,
    body:'For folks who want this pattern without ripping the queue out, an outbox + same-transaction read of the projection table will give you most of the benefit.',
    code:'-- in the same txn as the order insert\nINSERT INTO read_orders (id, total, status)\n  SELECT id, total(*), status\n  FROM orders WHERE id = $1;',
    time:'4h', reactions:{ 'galaxy-brain':3, 'plus-one':5, 'merge':2 },
    replies:[]},
  { id:'c4', anchor:'p8', author:'ravensong', avatar:'R', role:null,
    body:'I miss reading articles where someone has actually run something in production before writing about it. Thank you for the post.',
    time:'5h', reactions:{ 'this':8, 'plus-one':4 },
    replies:[]},
  { id:'c5', anchor:'p10', author:'nbiswas', avatar:'N', role:null,
    body:'What did you measure for \u201Cnothing exploded\u201D? Curious about the SLO deltas \u2014 p99, error rate, anything around the deploy itself.',
    time:'6h', reactions:{ 'eyes':9, 'this':2 },
    replies:[
      { id:'c5r1', author:'hana-okafor', avatar:'H', role:'author', body:'Full numbers in part two, but the headline: p99 on the read path went from 380ms \u2192 41ms, and the alert that fired most often (consumer lag &gt; 30s) just stopped existing. No new pages in six weeks.', time:'5h', reactions:{ 'ship-it':14, 'plus-one':8, 'rubber-duck':2 } },
    ]},
  { id:'c6', anchor:'p10', author:'tess-lin', avatar:'T', role:null,
    body:'The honesty about the \u201Cdid you mean to refresh?\u201D toast is the best part of this post. We have one of those. I am going to go delete it.',
    time:'1h', reactions:{ 'skull':6, 'this':3 },
    replies:[]},
];

// Renders the article body. `onParagraph` lets a widget host instrument the
// paragraphs (refs, comment counts, hover state) without forking the markup.
const Article = ({ onParagraph = null, density = 'normal' }) => {
  const sz = density === 'dense'
    ? { body: 17, lh: 1.55, h2: 22 }
    : { body: 19, lh: 1.7,  h2: 26 };
  return (
    <article style={{ fontFamily:'var(--serif)', color:'var(--ink)', fontSize:sz.body, lineHeight:sz.lh }}>
      <div style={{ fontFamily:'var(--mono)', fontSize:11, letterSpacing:'.14em', color:'var(--accent)', textTransform:'uppercase', marginBottom:14 }}>
        {ARTICLE.kicker} &nbsp;·&nbsp; {ARTICLE.date} &nbsp;·&nbsp; {ARTICLE.readTime}
      </div>
      <h1 style={{ fontFamily:'var(--display)', fontWeight:400, fontSize:54, lineHeight:1.04, letterSpacing:'-.01em', margin:'0 0 18px', color:'var(--ink)' }}>
        {ARTICLE.title}
      </h1>
      <p style={{ fontFamily:'var(--serif)', fontStyle:'italic', fontSize:22, lineHeight:1.4, color:'var(--mute)', margin:'0 0 24px', fontWeight:300 }}>
        {ARTICLE.dek}
      </p>
      <div style={{ display:'flex', alignItems:'center', gap:12, padding:'14px 0', borderTop:'1px solid var(--rule)', borderBottom:'1px solid var(--rule)', marginBottom:32 }}>
        <div style={{ width:36, height:36, borderRadius:'50%', background:'var(--accent)', color:'var(--paper)', display:'grid', placeItems:'center', fontFamily:'var(--display)', fontSize:18 }}>H</div>
        <div>
          <div style={{ fontSize:15, fontWeight:500 }}>{ARTICLE.author}</div>
          <div style={{ fontSize:12, color:'var(--mute)', fontFamily:'var(--mono)' }}>{ARTICLE.authorRole}</div>
        </div>
      </div>

      {ARTICLE.paragraphs.map((p,i)=>{
        const extras = onParagraph ? onParagraph(p, i) : {};
        if (p.kind === 'h2') return (
          <h2 key={p.id} data-pid={p.id} {...extras} style={{ fontFamily:'var(--display)', fontWeight:400, fontSize:sz.h2 + 8, margin:'34px 0 14px', position:'relative', ...(extras.style||{}) }}>{p.text}</h2>
        );
        if (p.kind === 'code') return (
          <pre key={p.id} data-pid={p.id} {...extras} style={{ fontFamily:'var(--mono)', fontSize:13, lineHeight:1.55, background:'var(--paper-2)', borderLeft:'2px solid var(--ink)', padding:'14px 16px', margin:'22px 0', borderRadius:2, overflow:'auto', position:'relative', ...(extras.style||{}) }}>
            <code dangerouslySetInnerHTML={{__html: highlightTs(p.text)}} />
          </pre>
        );
        return <p key={p.id} data-pid={p.id} {...extras} style={{ margin:'0 0 18px', position:'relative', ...(extras.style||{}) }}>{p.text}</p>;
      })}
    </article>
  );
};

// Tiny syntax highlighter so code blocks aren't gray-on-gray.
function highlightTs(src){
  const esc = (s)=>s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
  return esc(src)
    .replace(/(\/\/[^\n]*)/g, '<span style="color:#8a8170;font-style:italic">$1</span>')
    .replace(/(--[^\n]*)/g, '<span style="color:#8a8170;font-style:italic">$1</span>')
    .replace(/\b(async|function|await|const|let|return|SELECT|FROM|WHERE|INSERT|INTO)\b/g, '<span style="color:#B7351D;font-weight:600">$1</span>')
    .replace(/\b(true|false|null|undefined)\b/g, '<span style="color:#5B2A56">$1</span>')
    .replace(/(['"`])(.*?)\1/g, '<span style="color:#2E6A3E">$1$2$1</span>');
}

Object.assign(window, { ARTICLE, COMMENTS, Article });


// ===== widget-marginalia.jsx =====
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


// ===== widget-cadence.jsx =====
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


// ===== dashboard.jsx =====
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


// ===== app.jsx =====
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

