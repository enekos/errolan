/*
 * Errolan SDK — embeddable comment widget with two render modes.
 *
 * --- Cadence (flat conversation, default) -----------------------------
 *   <link rel="stylesheet" href="https://your-host/sdk/errolan.css">
 *   <div data-errolan-thread="post-slug"
 *        data-errolan-site="erl_yourkey"
 *        data-errolan-api="https://your-host"></div>
 *   <script src="https://your-host/sdk/errolan.js"></script>
 *
 * --- Marginalia (paragraph-anchored) ----------------------------------
 *   <article id="post">
 *     <p data-errolan-anchor="p1">First paragraph…</p>
 *     <p data-errolan-anchor="p2">Second paragraph…</p>
 *
 *     <div data-errolan-thread="post-slug"
 *          data-errolan-site="erl_yourkey"
 *          data-errolan-api="https://your-host"
 *          data-errolan-mode="marginalia"></div>
 *   </article>
 *
 * The widget auto-discovers the nearest <article> ancestor when no
 * data-errolan-article selector is given. Anchors can be either
 * `data-errolan-anchor="…"` (canonical) or any element matched by
 * `data-errolan-anchor-selector` whose `id` becomes the anchor.
 *
 * Compatibility extras (all optional):
 *   data-errolan-anchor-selector  CSS selector for anchor elements
 *                                 (default: "[data-errolan-anchor]").
 *                                 e.g. "h2[id], h3[id]" to anchor existing
 *                                 heading ids.
 *   data-errolan-inline-breakpoint Width (px) below which marginalia
 *                                 downgrades to inline stamps (default 900).
 *   data-errolan-inline           "true" forces inline rendering always.
 *   data-errolan-lazy             "true" defers mounting until the widget
 *                                 enters the viewport.
 *   data-errolan-manual           "true" skips auto-mount; call mount()
 *                                 yourself when you're ready.
 *
 * Programmatic mount:
 *   Errolan.mount(el, { api, site, thread, mode, article, ...});
 *
 * Internal structure (top→bottom):
 *   - Storage keys + small utilities (DOM, Format helpers)
 *   - resolveArticle, findAnchors: site-shape adapters
 *   - Client: a thin fetch wrapper around the Errolan HTTP API
 *   - LiveStream: SSE with polling fallback
 *   - AuthDialog / ReactionPicker / Composer: focused UI factories
 *   - CommentView: pure-function comment renderer broken into parts
 *   - MarginaliaRail: owns the rail-or-inline stamps and layout sync
 *   - Widget: top-level orchestrator
 *   - mount / autoMount: public surface
 */
(function (global) {
  "use strict";

  // =========================================================================
  // Storage keys & constants
  // =========================================================================

  const TOKEN_KEY = "errolan.token";
  const NAME_KEY  = "errolan.anonName";
  const POLL_MS   = 15000;
  const BODY_MAX  = 8000;
  const RAIL_W    = 280;
  const RAIL_GAP  = 32;
  const INLINE_BP = 900;  // default width (px) below which marginalia → inline

  // =========================================================================
  // DOM helpers
  // =========================================================================

  function el(tag, attrs, children) {
    const node = document.createElement(tag);
    if (attrs) for (const k in attrs) {
      const v = attrs[k];
      if (v == null || v === false) continue;
      if (k === "class") node.className = v;
      else if (k === "text") node.textContent = v;
      else if (k === "html") node.innerHTML = v;
      else if (k.startsWith("on") && typeof v === "function") node.addEventListener(k.slice(2).toLowerCase(), v);
      else node.setAttribute(k, v);
    }
    if (children) for (const c of [].concat(children)) {
      if (c == null) continue;
      node.appendChild(typeof c === "string" ? document.createTextNode(c) : c);
    }
    return node;
  }

  function removeNode(n) {
    if (n && n.parentNode) n.parentNode.removeChild(n);
  }

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
  }

  function cssEscape(s) {
    return (window.CSS && CSS.escape) ? CSS.escape(s) : s;
  }

  // isRTL walks up the ancestor chain looking for an explicit dir="rtl" /
  // "ltr" — falls back to the documentElement's computed direction. Used so
  // the rail flips sides for right-to-left articles automatically.
  function isRTL(node) {
    let n = node;
    while (n && n.getAttribute) {
      const dir = n.getAttribute("dir");
      if (dir) return dir.toLowerCase() === "rtl";
      n = n.parentElement;
    }
    try {
      return getComputedStyle(document.documentElement).direction === "rtl";
    } catch (_) { return false; }
  }

  // =========================================================================
  // Site-shape adapters: turn whatever DOM the host gave us into the bits
  // marginalia mode needs (article element, anchor list). These are written
  // to be tolerant: no <article>? no anchors? — return null/empty and let the
  // caller surface a useful error.
  // =========================================================================

  // resolveArticle accepts a CSS selector, a live element, or "" / undefined.
  // When nothing is supplied, walk up the widget's DOM ancestors to find the
  // nearest <article> — this lets users drop a marginalia widget inside their
  // article element without configuring the selector twice.
  function resolveArticle(opt, widgetRoot) {
    if (opt && typeof opt !== "string") return opt; // already a DOM element
    if (opt) return document.querySelector(opt);
    let n = widgetRoot && widgetRoot.parentElement;
    while (n) {
      if (n.tagName === "ARTICLE") return n;
      n = n.parentElement;
    }
    return null;
  }

  // findAnchors returns [{ id, elt }] for every element in articleEl that
  // matches the selector. If an element doesn't already carry
  // `data-errolan-anchor`, the helper stamps one — derived from the
  // element's `id` — so the existing CSS-driven highlight rules
  // ([data-errolan-anchor].erl-anchor-hot, …) keep working unchanged.
  function findAnchors(articleEl, selector) {
    if (!articleEl) return [];
    const out  = [];
    const seen = new Set();
    articleEl.querySelectorAll(selector).forEach(elt => {
      let id = elt.getAttribute("data-errolan-anchor") || elt.id || "";
      if (!id) return;
      if (seen.has(id)) return; // dedupe: first occurrence wins
      seen.add(id);
      if (!elt.getAttribute("data-errolan-anchor")) {
        elt.setAttribute("data-errolan-anchor", id);
      }
      out.push({ id, elt });
    });
    return out;
  }

  // =========================================================================
  // Format helpers (pure)
  // =========================================================================

  function formatBody(s, packByCode) {
    const escaped = escapeHTML(s);
    const withEmoji = escaped.replace(/:([a-z0-9_-]+):/g, (full, code) => {
      const e = packByCode && packByCode[code];
      return e ? emojiHTML(e, 16) : full;
    });
    const withLinks = withEmoji.replace(/(https?:\/\/[^\s<]+)/g,
      '<a href="$1" target="_blank" rel="nofollow noopener">$1</a>');
    return withLinks.split(/\n{2,}/).map(p => "<p>" + p.replace(/\n/g, "<br>") + "</p>").join("");
  }

  function emojiHTML(emoji, size) {
    if (!emoji || !emoji.svg) return "";
    const s = String(size || 16);
    if (emoji.svg.startsWith("https://")) {
      return `<img class="erl-emoji" src="${escapeHTML(emoji.svg)}" alt=":${escapeHTML(emoji.code)}:" width="${s}" height="${s}">`;
    }
    return `<span class="erl-emoji" style="width:${s}px;height:${s}px;" title=":${escapeHTML(emoji.code)}:">${emoji.svg}</span>`;
  }

  function relTime(unixSec) {
    const d = Math.floor(Date.now() / 1000 - unixSec);
    if (d < 60) return d + "s";
    if (d < 3600) return Math.floor(d / 60) + "m";
    if (d < 86400) return Math.floor(d / 3600) + "h";
    if (d < 2592000) return Math.floor(d / 86400) + "d";
    return new Date(unixSec * 1000).toLocaleDateString();
  }

  // =========================================================================
  // API client
  // =========================================================================

  class Client {
    constructor(opts) {
      this.api  = opts.api.replace(/\/$/, "");
      this.site = opts.site;
    }

    token()     { return localStorage.getItem(TOKEN_KEY) || ""; }
    setToken(t) { t ? localStorage.setItem(TOKEN_KEY, t) : localStorage.removeItem(TOKEN_KEY); }

    async request(method, path, body, extraHeaders) {
      const headers = { "X-Errolan-Site": this.site };
      if (body !== undefined) headers["Content-Type"] = "application/json";
      const tok = this.token();
      if (tok) headers["Authorization"] = "Bearer " + tok;
      if (extraHeaders) Object.assign(headers, extraHeaders);

      const res = await fetch(this.api + path, {
        method, headers,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      });
      if (res.status === 304) return { __notModified: true };
      if (res.status === 204) return null;

      const etag = res.headers.get("ETag");
      const text = await res.text();
      let data = null;
      try { data = text ? JSON.parse(text) : null; } catch (e) { data = { error: text }; }
      if (!res.ok) {
        const err = new Error((data && data.error) || ("HTTP " + res.status));
        err.status = res.status;
        throw err;
      }
      if (data && typeof data === "object") data.__etag = etag;
      return data;
    }

    getThread(slug, params, ifNoneMatch) {
      const qs = new URLSearchParams();
      if (params) {
        if (params.title)     qs.set("title", params.title);
        if (params.url)       qs.set("url", params.url);
        if (params.sort)      qs.set("sort", params.sort);
        if (params.limit)     qs.set("limit", params.limit);
        if (params.before_id) qs.set("before_id", params.before_id);
      }
      const suffix = qs.toString() ? "?" + qs.toString() : "";
      return this.request("GET", "/api/threads/" + encodeURIComponent(slug) + suffix,
        undefined, ifNoneMatch ? { "If-None-Match": ifNoneMatch } : null);
    }

    postComment(slug, body, parentID, authorName, anchor) {
      return this.request("POST", "/api/threads/" + encodeURIComponent(slug) + "/comments", {
        body, parent_id: parentID || null, author_name: authorName || "",
        website: "", anchor: anchor || "",
      });
    }

    editComment(id, body)   { return this.request("PATCH",  "/api/comments/" + id, { body }); }
    deleteComment(id)       { return this.request("DELETE", "/api/comments/" + id); }
    react(id, code)         { return this.request("POST",   "/api/comments/" + id + "/reactions", { code }); }
    flag(id, reason)        { return this.request("POST",   "/api/comments/" + id + "/flag", { reason: reason || "" }); }
    pin(id, pinned)         { return this.request("POST",   "/api/comments/" + id + "/pin", { pinned: !!pinned }); }
    login(email, pw)        { return this.request("POST",   "/api/auth/login", { email, password: pw }); }
    register(email, n, pw)  { return this.request("POST",   "/api/auth/register", { email, name: n, password: pw }); }
  }

  // =========================================================================
  // LiveStream: SSE with polling fallback
  // =========================================================================

  class LiveStream {
    constructor({ api, site, thread, onUpdate }) {
      this.api      = api;
      this.site     = site;
      this.thread   = thread;
      this.onUpdate = onUpdate;
      this.es       = null;
      this.poller   = null;
    }

    start() {
      if (typeof EventSource === "undefined") {
        this.startPolling();
        return;
      }
      const url = this.api + "/api/threads/" + encodeURIComponent(this.thread)
        + "/events?site=" + encodeURIComponent(this.site);
      try {
        this.es = new EventSource(url, { withCredentials: false });
        this.es.addEventListener("update", () => this.onUpdate());
        this.es.addEventListener("error",  () => { if (!this.poller) this.startPolling(); });
      } catch (_) {
        this.startPolling();
      }
    }

    startPolling() {
      if (this.poller) return;
      this.poller = setInterval(() => this.onUpdate(), POLL_MS);
    }

    stop() {
      if (this.es)     { this.es.close(); this.es = null; }
      if (this.poller) { clearInterval(this.poller); this.poller = null; }
    }
  }

  // =========================================================================
  // AuthDialog: sign-in / register modal
  // =========================================================================

  function openAuthDialog(widget, mode) {
    const overlay = el("div", { class: "erl-overlay" });
    const dialog  = el("div", { class: "erl-dialog" });
    const close   = () => removeNode(overlay);

    const email  = el("input",  { type: "email", placeholder: "email", required: true });
    const name   = mode === "register"
      ? el("input", { type: "text", placeholder: "display name", required: true })
      : null;
    const pw     = el("input",  { type: "password", placeholder: "password (≥8 chars)", required: true });
    const errBox = el("div",    { class: "erl-error" });
    const submit = el("button", { class: "erl-primary", text: mode === "login" ? "Sign in" : "Create" });
    const cancel = el("button", { class: "erl-link", text: "cancel", onClick: close });
    const switcher = el("button", {
      class: "erl-link",
      text: mode === "login" ? "need an account?" : "have an account?",
      onClick: () => { close(); openAuthDialog(widget, mode === "login" ? "register" : "login"); },
    });

    submit.addEventListener("click", async () => {
      errBox.textContent = "";
      try {
        const res = mode === "login"
          ? await widget.client.login(email.value, pw.value)
          : await widget.client.register(email.value, name.value, pw.value);
        widget.client.setToken(res.token);
        close();
        widget.etag = null;
        widget.refresh();
      } catch (e) { errBox.textContent = e.message; }
    });

    overlay.addEventListener("click", (ev) => { if (ev.target === overlay) close(); });

    dialog.appendChild(el("h4", { text: mode === "login" ? "Sign in" : "Create an account" }));
    dialog.appendChild(email);
    if (name) dialog.appendChild(name);
    dialog.appendChild(pw);
    dialog.appendChild(errBox);
    dialog.appendChild(el("div", { class: "erl-actions" }, [cancel, submit]));
    dialog.appendChild(switcher);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    email.focus();
  }

  // =========================================================================
  // ReactionPicker: floating popover with the site's emoji pack
  // =========================================================================

  function openReactionPicker(widget, comment, anchorBtn) {
    if (!widget.viewer) { openAuthDialog(widget, "login"); return; }

    removeNode(document.querySelector(".erl-rx-popover"));

    const rect = anchorBtn.getBoundingClientRect();
    const pop  = el("div", { class: "erl-rx-popover" });
    pop.style.position = "absolute";
    pop.style.top  = (rect.bottom + window.scrollY + 6) + "px";
    pop.style.left = (rect.left   + window.scrollX) + "px";

    Object.values(widget.packByCode).forEach(em => {
      pop.appendChild(el("button", {
        type: "button", class: "erl-rx-popover-btn",
        title: ":" + em.code + ":", html: emojiHTML(em, 18),
        onClick: () => { removeNode(pop); widget.handleReact(comment, em.code); },
      }));
    });
    document.body.appendChild(pop);

    setTimeout(() => {
      const onDocClick = (e) => {
        if (!pop.contains(e.target)) {
          removeNode(pop);
          document.removeEventListener("click", onDocClick);
        }
      };
      document.addEventListener("click", onDocClick);
    }, 0);
  }

  // =========================================================================
  // Composer: factory that returns a <form> element ready to insert
  // =========================================================================

  function renderComposer(widget, parentID, anchor) {
    const wrap = el("form", { class: "erl-composer" });
    const requireAuth = widget.state.site && widget.state.site.require_auth;
    if (requireAuth && !widget.viewer) {
      wrap.appendChild(el("div", { class: "erl-muted", text: "Please sign in to comment." }));
      return wrap;
    }

    if (anchor) {
      wrap.appendChild(el("div", { class: "erl-eyebrow", text: "Add a note to paragraph " + anchor }));
    }

    let nameInput = null;
    if (!widget.viewer) {
      nameInput = el("input", {
        type: "text", class: "erl-name", placeholder: "Your name",
        value: localStorage.getItem(NAME_KEY) || "",
      });
      wrap.appendChild(nameInput);
    }

    const textarea = el("textarea", {
      class: "erl-body",
      placeholder: parentID ? "Write a reply…"
        : (anchor ? "Write in the margin…" : "Write a thoughtful note. Use :code: for emoji."),
      rows: parentID ? 2 : 3,
    });
    wrap.appendChild(textarea);

    const honeypot = el("input", {
      type: "text", name: "website", class: "erl-hp",
      autocomplete: "off", tabindex: "-1",
    });
    wrap.appendChild(honeypot);

    if (Object.keys(widget.packByCode).length > 0) {
      const picker = el("div", { class: "erl-picker" });
      Object.values(widget.packByCode).forEach(em => {
        picker.appendChild(el("button", {
          type: "button", class: "erl-picker-btn",
          title: ":" + em.code + ":", html: emojiHTML(em, 18),
          onClick: () => {
            const before = textarea.value.slice(0, textarea.selectionStart);
            const after  = textarea.value.slice(textarea.selectionEnd);
            textarea.value = before + ":" + em.code + ":" + after;
            textarea.focus();
          },
        }));
      });
      wrap.appendChild(picker);
    }

    const errBox = el("div", { class: "erl-error" });
    const submit = el("button", {
      type: "submit", class: "erl-primary",
      text: parentID ? "Reply" : (anchor ? "Post note" : "Post"),
    });
    wrap.appendChild(errBox);
    wrap.appendChild(el("div", { class: "erl-actions" }, [submit]));

    wrap.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      errBox.textContent = "";
      const body = textarea.value.trim();
      if (!body) return;
      if (honeypot.value) return;
      if (body.length > BODY_MAX) {
        errBox.textContent = "Comment is too long.";
        return;
      }
      const an = nameInput ? nameInput.value.trim() : "";
      if (nameInput && an) localStorage.setItem(NAME_KEY, an);
      submit.disabled = true;
      try {
        await widget.client.postComment(widget.opts.thread, body, parentID, an, anchor);
        widget.replyTo = null;
        widget.etag = null;
        await widget.refresh();
        if (widget.activeAnchor && widget.mode === "marginalia") {
          const elNode = widget.articleEl.querySelector(
            '[data-errolan-anchor="' + cssEscape(widget.activeAnchor) + '"]');
          widget.openParagraphPanel(widget.activeAnchor, elNode);
        }
      } catch (e) { errBox.textContent = e.message; }
      finally { submit.disabled = false; }
    });

    return wrap;
  }

  // =========================================================================
  // CommentView: render a single comment into a <li>
  // =========================================================================

  function renderCommentHead(c) {
    const avatar = el("div", {
      class: "erl-avatar",
      style: c.avatar_url ? ("background-image:url(" + c.avatar_url + ")") : "",
      text:  c.avatar_url ? "" : (c.author_name || "?").charAt(0).toUpperCase(),
    });
    const meta = el("div", { class: "erl-c-meta" }, [
      el("span", { class: "erl-c-author", text: "@" + c.author_name }),
      c.pinned ? el("span", { class: "erl-pin-badge", text: "★ pinned" }) : null,
      el("span", { class: "erl-c-time",
        text: relTime(c.created_at) + (c.edit_count ? " · edited" : "") }),
    ]);
    return el("div", { class: "erl-c-head" }, [avatar, meta]);
  }

  function renderReactions(widget, c) {
    const strip    = el("div", { class: "erl-rx" });
    const myReacts = new Set(c.my_reacts || []);
    const counts   = c.reactions || {};

    Object.values(widget.packByCode).forEach(em => {
      const count = counts[em.code] || 0;
      if (count === 0 && !myReacts.has(em.code)) return;
      strip.appendChild(el("button", {
        class: "erl-rx-btn" + (myReacts.has(em.code) ? " erl-rx-active" : ""),
        title: ":" + em.code + ":",
        html:  emojiHTML(em, 14) + '<span class="erl-rx-count">' + count + "</span>",
        onClick: () => widget.handleReact(c, em.code),
      }));
    });

    strip.appendChild(el("button", {
      class: "erl-rx-add", text: "+ react",
      onClick: (ev) => openReactionPicker(widget, c, ev.currentTarget),
    }));
    return strip;
  }

  function renderCommentActions(widget, c) {
    const locked = widget.state.thread.locked;
    return el("div", { class: "erl-c-actions" }, [
      !locked ? el("button", {
        class: "erl-link", text: "reply",
        onClick: () => { widget.replyTo = c.id; widget.render(); },
      }) : null,
      widget.canEdit(c) ? el("button", {
        class: "erl-link", text: "edit",
        onClick: () => { widget.editing = c.id; widget.render(); },
      }) : null,
      widget.canEdit(c) && c.status !== "deleted" ? el("button", {
        class: "erl-link", text: "delete",
        onClick: () => widget.handleDelete(c),
      }) : null,
      widget.viewer && widget.viewer.is_admin ? el("button", {
        class: "erl-link", text: c.pinned ? "unpin" : "pin",
        onClick: () => widget.handlePin(c),
      }) : null,
      el("button", { class: "erl-link", text: "flag",
        onClick: () => widget.handleFlag(c) }),
    ]);
  }

  function renderEditor(widget, c) {
    const wrap = el("form", { class: "erl-composer" });
    const ta   = el("textarea", { class: "erl-body", rows: 3 });
    ta.value = c.body;
    const errBox = el("div", { class: "erl-error" });

    wrap.appendChild(ta);
    wrap.appendChild(errBox);
    wrap.appendChild(el("div", { class: "erl-actions" }, [
      el("button", { type: "submit", class: "erl-primary", text: "Save" }),
      el("button", { type: "button", class: "erl-link",    text: "cancel",
        onClick: () => { widget.editing = null; widget.render(); } }),
    ]));
    wrap.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      errBox.textContent = "";
      try {
        await widget.client.editComment(c.id, ta.value.trim());
        widget.editing = null;
        widget.etag    = null;
        await widget.refresh();
      } catch (e) { errBox.textContent = e.message; }
    });
    return wrap;
  }

  function renderComment(widget, c) {
    const li = el("li", { class: "erl-c" + (c.pinned ? " erl-c-pinned" : ""), "data-id": c.id });
    li.appendChild(renderCommentHead(c));

    const bodyNode = widget.editing === c.id
      ? renderEditor(widget, c)
      : el("div", { class: "erl-c-body", html: formatBody(c.body, widget.packByCode) });
    li.appendChild(bodyNode);

    li.appendChild(renderReactions(widget, c));
    li.appendChild(renderCommentActions(widget, c));

    if (widget.replyTo === c.id && !widget.state.thread.locked) {
      li.appendChild(renderComposer(widget, c.id, c.anchor || ""));
    }
    if (c.replies && c.replies.length) {
      const ul = el("ul", { class: "erl-list erl-replies" });
      for (const r of c.replies) ul.appendChild(renderComment(widget, r));
      li.appendChild(ul);
    }
    return li;
  }

  // =========================================================================
  // MarginaliaRail: stamps next to anchored elements.
  //
  // Two rendering layouts share the same stamp shape:
  //   rail    – absolute-positioned column inside the article element
  //   inline  – stamp inserted as a sibling right after each anchor
  //
  // Picking between the two happens on every build() and on viewport resize:
  //   - forceInline option → always inline
  //   - viewport width < inlineBreakpoint → inline (mobile fallback)
  //   - otherwise → rail
  //
  // Lives entirely within the article element when in rail mode, so the
  // widget composes cleanly inside any scroll container, stacking context,
  // or alongside other widgets on the same page. The previous body-level
  // implementation broke in all of those cases.
  // =========================================================================

  class MarginaliaRail {
    constructor(widget) {
      this.widget   = widget;
      this.rail     = null;
      this.inlineStamps = [];
      this.ro       = null;          // ResizeObserver — relayout on article-size changes
      this._savedArticlePos = null;  // restore inline style on detach
      this._onResize = () => this.layout();
      this._onResizeRebuild = () => this.maybeRebuild();
    }

    attach() {
      const article = this.widget.articleEl;
      // We anchor the rail inside the article with position: absolute; force the
      // article to be a positioned ancestor if it isn't already. We capture the
      // pre-existing inline style so detach() can restore it exactly.
      this._savedArticlePos = article.style.position;
      if (getComputedStyle(article).position === "static") {
        article.style.position = "relative";
      }
      if (typeof ResizeObserver !== "undefined") {
        this.ro = new ResizeObserver(this._onResize);
        this.ro.observe(article);
      }
      window.addEventListener("resize", this._onResizeRebuild);
      window.addEventListener("load",   this._onResize);
    }

    detach() {
      if (this.ro) { this.ro.disconnect(); this.ro = null; }
      window.removeEventListener("resize", this._onResizeRebuild);
      window.removeEventListener("load",   this._onResize);
      if (this.widget.articleEl) {
        this.widget.articleEl.style.position = this._savedArticlePos || "";
      }
      this.clear();
    }

    clear() {
      removeNode(this.rail);
      this.rail = null;
      this.inlineStamps.forEach(removeNode);
      this.inlineStamps = [];
    }

    // shouldInline picks the rendering layout. We collapse to inline when:
    //   1. forceInline is set,
    //   2. the viewport is narrower than the configured breakpoint,
    //   3. the article (or an ancestor we'd be positioned against) clips
    //      horizontally — the rail would be invisible inside that container,
    //   4. there isn't enough lateral room next to the article for the rail —
    //      a common case on full-width article designs.
    // These three lateral guards are what makes the widget self-correct across
    // many host layouts instead of demanding a magazine-style two-column page.
    shouldInline() {
      if (this.widget.opts.forceInline) return true;
      const bp = this.widget.opts.inlineBreakpoint || INLINE_BP;
      if (window.innerWidth < bp) return true;

      const article = this.widget.articleEl;
      if (!article) return true;

      // Walk the offset ancestor chain: any clipping ancestor between the
      // article and the viewport means the rail would be hidden.
      let n = article;
      while (n && n !== document.body && n !== document.documentElement) {
        const cs = getComputedStyle(n);
        if (cs.overflow === "hidden" || cs.overflowX === "hidden") return true;
        n = n.parentElement;
      }

      // Make sure there's room for the rail next to the article on the side
      // we'd put it (right for LTR, left for RTL).
      const rect = article.getBoundingClientRect();
      const lateral = isRTL(article)
        ? rect.left
        : (window.innerWidth - rect.right);
      if (lateral < (RAIL_W + RAIL_GAP)) return true;

      return false;
    }

    // maybeRebuild is the debounced resize handler. We only rebuild when the
    // layout type actually flips so a slow drag-resize doesn't churn the DOM
    // every pixel.
    maybeRebuild() {
      const wantInline = this.shouldInline();
      const isInline   = this.inlineStamps.length > 0;
      if (wantInline === isInline && this.rail) {
        this.layout();
        return;
      }
      this.build();
    }

    // collectStampSources turns the configured anchor selector into a list of
    // (id, anchor element, comment list) tuples, ready for rendering.
    collectStampSources() {
      const byAnchor = {};
      (this.widget.state.comments || []).forEach(c => {
        if (c.anchor) (byAnchor[c.anchor] ||= []).push(c);
      });
      const selector = this.widget.anchorSelector;
      return findAnchors(this.widget.articleEl, selector).map(({ id, elt }) => ({
        id, elt, comments: byAnchor[id] || [],
      }));
    }

    build() {
      this.clear();
      const sources = this.collectStampSources();
      if (sources.length === 0) return;

      const inline = this.shouldInline();
      if (inline) {
        sources.forEach(({ id, elt, comments }) => {
          const stamp = this.renderStamp(id, comments, elt);
          stamp.classList.add("erl-stamp-inline");
          elt.insertAdjacentElement("afterend", stamp);
          this.inlineStamps.push(stamp);
        });
      } else {
        const rail = el("div", { class: "erl-rail" });
        rail.style.position = "absolute";
        rail.style.top      = "0";
        rail.style.width    = RAIL_W + "px";
        rail.style.pointerEvents = "none";
        if (isRTL(this.widget.articleEl)) {
          rail.style.left  = "-" + (RAIL_W + RAIL_GAP) + "px";
          rail.style.right = "auto";
        } else {
          rail.style.left  = "auto";
          rail.style.right = "-" + (RAIL_W + RAIL_GAP) + "px";
        }
        this.widget.articleEl.appendChild(rail);
        this.rail = rail;
        sources.forEach(({ id, elt, comments }) => {
          const stamp = this.renderStamp(id, comments, elt);
          stamp.style.pointerEvents = "auto";
          rail.appendChild(stamp);
        });
      }
      this.layout();
    }

    // layout only matters in rail mode — inline stamps flow with the document
    // and don't need explicit positioning.
    layout() {
      if (!this.rail || !this.widget.articleEl) return;
      const article = this.widget.articleEl;
      const articleRect = article.getBoundingClientRect();
      this.rail.style.height = articleRect.height + "px";

      Array.from(this.rail.children).forEach(stamp => {
        const id  = stamp.getAttribute("data-anchor");
        const elt = article.querySelector('[data-errolan-anchor="' + cssEscape(id) + '"]');
        if (!elt) return;
        const top = elt.getBoundingClientRect().top - articleRect.top;
        stamp.style.position = "absolute";
        stamp.style.top      = top + "px";
        stamp.style.left     = "0";
        stamp.style.right    = "0";
      });
    }

    renderStamp(anchorID, comments, paragraphEl) {
      const widget = this.widget;
      const total  = comments.reduce((s, c) => s + 1 + ((c.replies || []).length), 0);
      const tally  = {};
      comments.forEach(c => {
        Object.entries(c.reactions || {}).forEach(([k, v]) => tally[k] = (tally[k] || 0) + v);
        (c.replies || []).forEach(r =>
          Object.entries(r.reactions || {}).forEach(([k, v]) => tally[k] = (tally[k] || 0) + v));
      });
      const top = Object.entries(tally).sort((a, b) => b[1] - a[1]).slice(0, 3);

      const stamp = el("button", {
        class: "erl-stamp"
          + (total === 0 ? " erl-stamp-empty" : "")
          + (widget.activeAnchor === anchorID ? " erl-stamp-active" : ""),
        "data-anchor": anchorID,
        onClick:      () => widget.openParagraphPanel(anchorID, paragraphEl),
        onMouseEnter: () => paragraphEl.classList.add("erl-anchor-hot"),
        onMouseLeave: () => paragraphEl.classList.remove("erl-anchor-hot"),
      });

      if (total === 0) {
        stamp.appendChild(el("span", { class: "erl-stamp-add", text: "+ Add a note" }));
        return stamp;
      }

      const rxRow = el("span", { class: "erl-stamp-rx" });
      top.forEach(([code, count]) => {
        const e = widget.packByCode[code];
        if (e) rxRow.insertAdjacentHTML("beforeend",
          emojiHTML(e, 14) + '<span class="erl-stamp-rx-count">' + count + "</span>");
      });
      stamp.appendChild(rxRow);
      stamp.appendChild(el("span", { class: "erl-stamp-meta",
        text: total + " " + (total === 1 ? "note" : "notes") }));
      const preview = comments[0].body.slice(0, 90);
      stamp.appendChild(el("span", { class: "erl-stamp-preview",
        text: '"' + preview + (comments[0].body.length > 90 ? "…" : "") + '"' }));
      stamp.appendChild(el("span", { class: "erl-stamp-author",
        text: "— @" + comments[0].author_name }));
      return stamp;
    }
  }

  // =========================================================================
  // Widget — top-level orchestrator
  // =========================================================================

  class Widget {
    constructor(root, opts) {
      this.root   = root;
      this.opts   = opts;
      this.client = new Client(opts);
      this.mode   = opts.mode === "marginalia" ? "marginalia" : "cadence";
      this.sort   = opts.sort || "best";
      this.live   = opts.live !== false;

      // Anchor selector: default to data-attribute, but accept any CSS
      // selector so hosts can use existing semantic ids (e.g. heading anchors).
      this.anchorSelector = opts.anchorSelector || "[data-errolan-anchor]";

      this.viewer       = null;
      this.state        = null;
      this.packByCode   = {};
      this.replyTo      = null;
      this.editing      = null;
      this.etag         = null;
      this.refreshing   = false;

      this.stream       = null;
      this.articleEl    = null;
      this.rail         = null;
      this.panel        = null;
      this.activeAnchor = null;
      this._escHandler  = null;
      this._lazyObserver = null;
    }

    // mount is the public entry point. If `lazy` is set, the actual data fetch
    // and DOM render is deferred until the widget enters the viewport — useful
    // when the comments block sits far below the fold on long articles.
    async mount() {
      this.root.classList.add("erl");
      this.root.classList.add("erl-" + this.mode);
      this.root.innerHTML = "";
      this.root.appendChild(el("div", { class: "erl-loading", text: "Loading conversation…" }));

      if (this.mode === "marginalia") {
        this.articleEl = resolveArticle(this.opts.article, this.root);
        if (!this.articleEl) {
          this.root.innerHTML = "";
          this.root.appendChild(el("div", { class: "erl-error", text:
            "Marginalia mode needs an article element. Set data-errolan-article=\"<selector>\" or place the widget inside <article>." }));
          return;
        }
        if (findAnchors(this.articleEl, this.anchorSelector).length === 0) {
          // Soft-warn: we keep the widget alive (general notes still work) but
          // surface the situation so hosts notice the article has no anchors.
          this.root.appendChild(el("div", { class: "erl-muted", text:
            "No anchored paragraphs found (selector: " + this.anchorSelector + "). General notes only." }));
        }
        this.rail = new MarginaliaRail(this);
      }

      if (this.opts.lazy && typeof IntersectionObserver !== "undefined") {
        await this._waitForViewport();
      }

      try {
        await this.refresh();
        if (this.live) this.startLive();
        if (this.rail) this.rail.attach();
      } catch (e) {
        this.root.innerHTML = "";
        this.root.appendChild(el("div", { class: "erl-error", text: "Failed to load: " + e.message }));
      }
    }

    // _waitForViewport returns once the widget root scrolls within rootMargin
    // of the viewport. The promise also resolves on the next animation frame
    // when the widget is already on-screen — avoids waiting forever for an
    // observer that won't fire.
    _waitForViewport() {
      return new Promise((resolve) => {
        const io = new IntersectionObserver((entries) => {
          if (entries.some(e => e.isIntersecting)) {
            io.disconnect();
            this._lazyObserver = null;
            resolve();
          }
        }, { rootMargin: "200px" });
        this._lazyObserver = io;
        io.observe(this.root);
      });
    }

    destroy() {
      if (this._lazyObserver) { this._lazyObserver.disconnect(); this._lazyObserver = null; }
      if (this.stream) { this.stream.stop(); this.stream = null; }
      if (this.rail)   { this.rail.detach(); this.rail = null; }
      this.closeParagraphPanel();
      delete this.root.__erlMounted;
    }

    async refresh(opts) {
      if (this.refreshing) return;
      this.refreshing = true;
      try {
        const params = {
          title: this.opts.title, url: this.opts.url,
          sort: this.sort, limit: 100,
        };
        const ifMatch = opts && opts.useETag ? this.etag : null;
        const data = await this.client.getThread(this.opts.thread, params, ifMatch);
        if (data && data.__notModified) return;

        this.etag       = (data && data.__etag) || null;
        this.state      = data;
        this.viewer     = data.viewer || null;
        this.packByCode = {};
        (data.emojis || []).forEach(e => { this.packByCode[e.code] = e; });
        this.render();
      } finally { this.refreshing = false; }
    }

    startLive() {
      if (!this.state || !this.state.thread) return;
      this.stream = new LiveStream({
        api:    this.client.api,
        site:   this.opts.site,
        thread: this.opts.thread,
        onUpdate: () => this.refresh({ useETag: true }),
      });
      this.stream.start();
    }

    render() {
      this.root.innerHTML = "";
      if (this.mode === "marginalia") this.renderMarginalia();
      else this.renderCadence();
    }

    // ----- cadence -----

    renderCadence() {
      const total = (this.state.thread && this.state.thread.comment_count) || 0;
      this.root.appendChild(el("div", { class: "erl-divider" }, [
        el("span", { class: "erl-rule" }),
        el("span", { class: "erl-divider-label", text: "The conversation" }),
        el("span", { class: "erl-rule" }),
      ]));
      this.root.appendChild(el("div", { class: "erl-header" }, [
        el("span", { class: "erl-count", text: total + " " + (total === 1 ? "note" : "notes") }),
        this.renderSortControl(),
        this.renderAuthBar(),
      ]));

      if (this.state.thread && this.state.thread.locked) {
        this.root.appendChild(el("div", { class: "erl-locked", text: "This thread is locked." }));
      } else {
        this.root.appendChild(renderComposer(this, null, ""));
      }

      const list = el("ul", { class: "erl-list erl-river" });
      for (const c of this.state.comments || []) list.appendChild(renderComment(this, c));
      this.root.appendChild(list);

      if (this.state.has_more) {
        this.root.appendChild(el("div", { class: "erl-more" }, [
          el("button", { class: "erl-link", text: "Load more", onClick: () => this.loadMore() }),
        ]));
      }
      this.root.appendChild(el("div", { class: "erl-footer", html: 'Powered by <a href="#">Errolan</a>' }));
    }

    // ----- marginalia -----

    renderMarginalia() {
      const total = (this.state.thread && this.state.thread.comment_count) || 0;
      this.root.appendChild(el("div", { class: "erl-marginalia-summary" }, [
        el("div", { class: "erl-eyebrow",
          text: "Marginalia · " + total + " " + (total === 1 ? "note" : "notes") }),
        el("div", { class: "erl-summary-row" }, [
          el("span", { class: "erl-summary-text",
            text: "Hover the right margin to see what readers wrote next to each paragraph." }),
          this.renderAuthBar(),
        ]),
      ]));

      const orphans = (this.state.comments || []).filter(c => !c.anchor);
      if (orphans.length) {
        this.root.appendChild(el("div", { class: "erl-divider" }, [
          el("span", { class: "erl-rule" }),
          el("span", { class: "erl-divider-label", text: "General notes" }),
          el("span", { class: "erl-rule" }),
        ]));
        const list = el("ul", { class: "erl-list" });
        orphans.forEach(c => list.appendChild(renderComment(this, c)));
        this.root.appendChild(list);
      }

      this.rail.build();
    }

    openParagraphPanel(anchorID, paragraphEl) {
      this.activeAnchor = anchorID;
      if (paragraphEl) {
        this.articleEl.querySelectorAll("[data-errolan-anchor]")
          .forEach(p => p.classList.remove("erl-anchor-active"));
        paragraphEl.classList.add("erl-anchor-active");
      }
      removeNode(this.panel);

      const overlay = el("div", { class: "erl-panel-overlay",
        onClick: (ev) => { if (ev.target === overlay) this.closeParagraphPanel(); }});
      const panel = el("aside", { class: "erl-panel" });
      panel.appendChild(el("button", {
        class: "erl-panel-close", text: "×", title: "Close (Esc)",
        onClick: () => this.closeParagraphPanel(),
      }));
      panel.appendChild(el("div", { class: "erl-eyebrow", text: "Marginalia · paragraph " + anchorID }));

      if (paragraphEl) {
        const txt = paragraphEl.textContent || "";
        panel.appendChild(el("blockquote", { class: "erl-quoted",
          text: txt.slice(0, 240) + (txt.length > 240 ? "…" : "") }));
      }

      const list = el("ul", { class: "erl-list erl-panel-thread" });
      const comments = (this.state.comments || []).filter(c => c.anchor === anchorID);
      if (comments.length === 0) {
        list.appendChild(el("li", { class: "erl-muted", text: "Be the first to write here." }));
      } else {
        comments.forEach(c => list.appendChild(renderComment(this, c)));
      }
      panel.appendChild(list);

      if (this.state.thread && !this.state.thread.locked) {
        panel.appendChild(renderComposer(this, null, anchorID));
      } else if (this.state.thread && this.state.thread.locked) {
        panel.appendChild(el("div", { class: "erl-locked", text: "This thread is locked." }));
      }

      overlay.appendChild(panel);
      document.body.appendChild(overlay);
      this.panel = overlay;
      this._escHandler = (ev) => { if (ev.key === "Escape") this.closeParagraphPanel(); };
      document.addEventListener("keydown", this._escHandler);
    }

    closeParagraphPanel() {
      removeNode(this.panel);
      this.panel = null;
      this.activeAnchor = null;
      if (this._escHandler) {
        document.removeEventListener("keydown", this._escHandler);
        this._escHandler = null;
      }
      if (this.articleEl) {
        this.articleEl.querySelectorAll("[data-errolan-anchor]")
          .forEach(p => p.classList.remove("erl-anchor-active"));
      }
      if (this.mode === "marginalia" && this.rail) this.rail.build();
    }

    // ----- shared -----

    renderSortControl() {
      const sel = el("select", { class: "erl-sort" });
      [["best","Best"],["newest","Newest"],["oldest","Oldest"]].forEach(([v, label]) => {
        const opt = el("option", { value: v, text: label });
        if (v === this.sort) opt.selected = "selected";
        sel.appendChild(opt);
      });
      sel.addEventListener("change", () => {
        this.sort = sel.value;
        this.etag = null;
        this.refresh();
      });
      return sel;
    }

    async loadMore() {
      if (!this.state || !this.state.next_id) return;
      const params = {
        title: this.opts.title, url: this.opts.url,
        sort: this.sort, limit: 100, before_id: this.state.next_id,
      };
      try {
        const more = await this.client.getThread(this.opts.thread, params);
        this.state.comments = (this.state.comments || []).concat(more.comments || []);
        this.state.has_more = more.has_more;
        this.state.next_id  = more.next_id || null;
        this.render();
      } catch (e) { alert(e.message); }
    }

    renderAuthBar() {
      if (this.viewer) {
        return el("div", { class: "erl-auth" }, [
          el("span", { class: "erl-user", text: "@" + this.viewer.name }),
          this.viewer.is_admin ? el("span", { class: "erl-badge", text: "admin" }) : null,
          el("button", { class: "erl-link", text: "sign out",
            onClick: () => {
              this.client.setToken("");
              this.etag = null;
              this.refresh();
            }}),
        ]);
      }
      return el("div", { class: "erl-auth" }, [
        el("button", { class: "erl-link", text: "sign in",
          onClick: () => openAuthDialog(this, "login") }),
        el("span", { class: "erl-divider-dot", text: "·" }),
        el("button", { class: "erl-link", text: "register",
          onClick: () => openAuthDialog(this, "register") }),
      ]);
    }

    canEdit(c) {
      if (!this.viewer) return false;
      if (this.viewer.is_admin) return true;
      return c.user_id === this.viewer.id;
    }

    // ----- per-comment actions -----

    async handleReact(c, code) {
      if (!this.viewer) { openAuthDialog(this, "login"); return; }
      try { await this.client.react(c.id, code); this.etag = null; await this.refresh(); }
      catch (e) { alert(e.message); }
    }
    async handleDelete(c) {
      if (!confirm("Delete this comment?")) return;
      try { await this.client.deleteComment(c.id); this.etag = null; await this.refresh(); }
      catch (e) { alert(e.message); }
    }
    async handlePin(c) {
      try { await this.client.pin(c.id, !c.pinned); this.etag = null; await this.refresh(); }
      catch (e) { alert(e.message); }
    }
    async handleFlag(c) {
      const reason = prompt("Flag this comment — reason (optional):");
      if (reason === null) return;
      try { await this.client.flag(c.id, reason); alert("Thanks — a moderator will review it."); }
      catch (e) { alert(e.message); }
    }
  }

  // =========================================================================
  // Mounting
  // =========================================================================

  function mount(element, opts) {
    if (!opts || !opts.api || !opts.site || !opts.thread) {
      throw new Error("Errolan: api, site, and thread are required");
    }
    if (element.__erlMounted) return element.__erlMounted;
    const w = new Widget(element, opts);
    element.__erlMounted = w;
    w.mount();
    return w;
  }

  // readDataOpts pulls every supported attribute off an element. Booleans use
  // the "true" literal so omitting the attribute is unambiguously "off".
  function readDataOpts(n) {
    const bp = parseInt(n.getAttribute("data-errolan-inline-breakpoint"), 10);
    return {
      api:               n.getAttribute("data-errolan-api")   || global.ERROLAN_API  || "",
      site:              n.getAttribute("data-errolan-site")  || global.ERROLAN_SITE || "",
      thread:            n.getAttribute("data-errolan-thread"),
      title:             n.getAttribute("data-errolan-title") || document.title,
      url:               n.getAttribute("data-errolan-url")   || location.href,
      sort:              n.getAttribute("data-errolan-sort")  || "best",
      mode:              n.getAttribute("data-errolan-mode")  || "cadence",
      article:           n.getAttribute("data-errolan-article") || "",
      anchorSelector:    n.getAttribute("data-errolan-anchor-selector") || "",
      inlineBreakpoint:  isNaN(bp) ? undefined : bp,
      forceInline:       n.getAttribute("data-errolan-inline") === "true",
      lazy:              n.getAttribute("data-errolan-lazy")   === "true",
      live:              n.getAttribute("data-errolan-live")   !== "false",
    };
  }

  function autoMount() {
    document.querySelectorAll("[data-errolan-thread]").forEach(n => {
      if (n.__erlMounted) return;
      if (n.getAttribute("data-errolan-manual") === "true") return;
      mount(n, readDataOpts(n));
    });
  }

  // Public surface. autoMount is exposed so manual-mode hosts can trigger
  // discovery later (e.g. after SPA route changes inject new widgets).
  global.Errolan = { mount, autoMount, Client };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})(window);
