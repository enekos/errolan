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
