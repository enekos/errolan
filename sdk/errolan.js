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
 *   </article>
 *
 *   <div data-errolan-thread="post-slug"
 *        data-errolan-site="erl_yourkey"
 *        data-errolan-api="https://your-host"
 *        data-errolan-mode="marginalia"
 *        data-errolan-article="#post"></div>
 *
 * On the host page, each anchored element should carry a STABLE id
 * (`data-errolan-anchor`) — usually derived from heading/paragraph slugs.
 * Reuse the same id every render so comments stick to the right text.
 *
 * Programmatic mount:
 *   Errolan.mount(el, { api, site, thread, mode, article, ...});
 */
(function (global) {
  "use strict";

  const TOKEN_KEY = "errolan.token";
  const NAME_KEY  = "errolan.anonName";

  // ---------- DOM helpers ----------

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

  function escapeHTML(s) {
    return String(s).replace(/[&<>"']/g, c => ({ "&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
  }

  // Minimal body formatter: escape HTML, expand :emoji: tokens against the
  // site's pack, linkify URLs, and break double newlines into paragraphs.
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

  // ---------- API client ----------

  function Client(opts) {
    this.api = opts.api.replace(/\/$/, "");
    this.site = opts.site;
  }
  Client.prototype.token = function () { return localStorage.getItem(TOKEN_KEY) || ""; };
  Client.prototype.setToken = function (t) { t ? localStorage.setItem(TOKEN_KEY, t) : localStorage.removeItem(TOKEN_KEY); };

  Client.prototype.request = async function (method, path, body, extraHeaders) {
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
  };

  Client.prototype.getThread = function (slug, params, ifNoneMatch) {
    const qs = new URLSearchParams();
    if (params) {
      if (params.title) qs.set("title", params.title);
      if (params.url) qs.set("url", params.url);
      if (params.sort) qs.set("sort", params.sort);
      if (params.limit) qs.set("limit", params.limit);
      if (params.before_id) qs.set("before_id", params.before_id);
    }
    const suffix = qs.toString() ? "?" + qs.toString() : "";
    return this.request("GET", "/api/threads/" + encodeURIComponent(slug) + suffix,
      undefined, ifNoneMatch ? { "If-None-Match": ifNoneMatch } : null);
  };
  Client.prototype.postComment = function (slug, body, parentID, authorName, anchor) {
    return this.request("POST", "/api/threads/" + encodeURIComponent(slug) + "/comments", {
      body, parent_id: parentID || null, author_name: authorName || "",
      website: "", anchor: anchor || "",
    });
  };
  Client.prototype.editComment   = function (id, body)    { return this.request("PATCH", "/api/comments/" + id, { body }); };
  Client.prototype.deleteComment = function (id)          { return this.request("DELETE", "/api/comments/" + id); };
  Client.prototype.react         = function (id, code)    { return this.request("POST", "/api/comments/" + id + "/reactions", { code }); };
  Client.prototype.flag          = function (id, reason)  { return this.request("POST", "/api/comments/" + id + "/flag", { reason: reason || "" }); };
  Client.prototype.pin           = function (id, pinned)  { return this.request("POST", "/api/comments/" + id + "/pin", { pinned: !!pinned }); };
  Client.prototype.login         = function (email, pw)   { return this.request("POST", "/api/auth/login", { email, password: pw }); };
  Client.prototype.register      = function (email, n, pw){ return this.request("POST", "/api/auth/register", { email, name: n, password: pw }); };

  // ---------- Widget ----------

  function Widget(root, opts) {
    this.root = root;
    this.opts = opts;
    this.client = new Client(opts);
    this.mode = opts.mode === "marginalia" ? "marginalia" : "cadence";
    this.sort = opts.sort || "best";
    this.live = opts.live !== false;
    this.viewer = null;
    this.state = null;
    this.packByCode = {};
    this.replyTo = null;
    this.editing = null;
    this.etag = null;
    this.es = null;
    this.poller = null;
    this.refreshing = false;

    // Marginalia-only state
    this.articleEl = null;
    this.rail = null;
    this.panel = null;
    this.activeAnchor = null;
    this._escHandler = null;
    this._resizeHandler = null;
  }

  Widget.prototype.mount = async function () {
    this.root.classList.add("erl");
    this.root.classList.add("erl-" + this.mode);
    this.root.innerHTML = "";
    this.root.appendChild(el("div", { class: "erl-loading", text: "Loading conversation…" }));

    if (this.mode === "marginalia") {
      const sel = this.opts.article;
      this.articleEl = sel ? document.querySelector(sel) : null;
      if (!this.articleEl) {
        this.root.innerHTML = "";
        this.root.appendChild(el("div", { class: "erl-error",
          text: 'Marginalia mode requires data-errolan-article="<selector>" pointing at the article element.' }));
        return;
      }
    }

    try {
      await this.refresh();
      if (this.live) this.startLive();
      if (this.mode === "marginalia") {
        this._resizeHandler = () => this.layoutRail();
        window.addEventListener("resize", this._resizeHandler);
        window.addEventListener("load", this._resizeHandler);
      }
    } catch (e) {
      this.root.innerHTML = "";
      this.root.appendChild(el("div", { class: "erl-error", text: "Failed to load: " + e.message }));
    }
  };

  Widget.prototype.destroy = function () {
    if (this.es) { this.es.close(); this.es = null; }
    if (this.poller) { clearInterval(this.poller); this.poller = null; }
    if (this._resizeHandler) {
      window.removeEventListener("resize", this._resizeHandler);
      window.removeEventListener("load", this._resizeHandler);
    }
    if (this.rail && this.rail.parentNode) this.rail.parentNode.removeChild(this.rail);
    this.closeParagraphPanel();
  };

  Widget.prototype.refresh = async function (opts) {
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
      this.etag = (data && data.__etag) || null;
      this.state = data;
      this.viewer = data.viewer || null;
      this.packByCode = {};
      (data.emojis || []).forEach(e => { this.packByCode[e.code] = e; });
      this.render();
    } finally { this.refreshing = false; }
  };

  Widget.prototype.startLive = function () {
    if (typeof EventSource !== "undefined" && this.state && this.state.thread) {
      const url = this.client.api + "/api/threads/" + encodeURIComponent(this.opts.thread)
        + "/events?site=" + encodeURIComponent(this.opts.site);
      try {
        this.es = new EventSource(url, { withCredentials: false });
        this.es.addEventListener("update", () => this.refresh({ useETag: true }));
        this.es.addEventListener("error", () => { if (!this.poller) this.startPolling(); });
        return;
      } catch (e) {}
    }
    this.startPolling();
  };
  Widget.prototype.startPolling = function () {
    if (this.poller) return;
    this.poller = setInterval(() => this.refresh({ useETag: true }), 15000);
  };

  Widget.prototype.render = function () {
    this.root.innerHTML = "";
    if (this.mode === "marginalia") this.renderMarginalia();
    else this.renderCadence();
  };

  // ---------- Cadence mode ----------

  Widget.prototype.renderCadence = function () {
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
      this.root.appendChild(this.renderComposer(null, ""));
    }

    const list = el("ul", { class: "erl-list erl-river" });
    for (const c of this.state.comments || []) list.appendChild(this.renderComment(c));
    this.root.appendChild(list);

    if (this.state.has_more) {
      const w = this;
      this.root.appendChild(el("div", { class: "erl-more" }, [
        el("button", { class: "erl-link", text: "Load more", onClick: () => w.loadMore() }),
      ]));
    }
    this.root.appendChild(el("div", { class: "erl-footer", html: 'Powered by <a href="#">Errolan</a>' }));
  };

  // ---------- Marginalia mode ----------

  Widget.prototype.renderMarginalia = function () {
    const total = (this.state.thread && this.state.thread.comment_count) || 0;
    const card = el("div", { class: "erl-marginalia-summary" }, [
      el("div", { class: "erl-eyebrow", text: "Marginalia · " + total + " " + (total === 1 ? "note" : "notes") }),
      el("div", { class: "erl-summary-row" }, [
        el("span", { class: "erl-summary-text",
          text: "Hover the right margin to see what readers wrote next to each paragraph." }),
        this.renderAuthBar(),
      ]),
    ]);
    this.root.appendChild(card);

    // Un-anchored (thread-level) notes still render below the article.
    const orphans = (this.state.comments || []).filter(c => !c.anchor);
    if (orphans.length) {
      this.root.appendChild(el("div", { class: "erl-divider" }, [
        el("span", { class: "erl-rule" }),
        el("span", { class: "erl-divider-label", text: "General notes" }),
        el("span", { class: "erl-rule" }),
      ]));
      const list = el("ul", { class: "erl-list" });
      orphans.forEach(c => list.appendChild(this.renderComment(c)));
      this.root.appendChild(list);
    }

    this.buildRail();
    this.layoutRail();
  };

  Widget.prototype.buildRail = function () {
    if (this.rail && this.rail.parentNode) this.rail.parentNode.removeChild(this.rail);
    const rail = el("div", { class: "erl-rail" });
    rail.style.position = "absolute";
    rail.style.pointerEvents = "none";
    document.body.appendChild(rail);
    this.rail = rail;

    const byAnchor = {};
    (this.state.comments || []).forEach(c => {
      if (!c.anchor) return;
      (byAnchor[c.anchor] ||= []).push(c);
    });

    const widget = this;
    this.articleEl.querySelectorAll("[data-errolan-anchor]").forEach(elt => {
      const id = elt.getAttribute("data-errolan-anchor");
      const stamp = widget.renderStamp(id, byAnchor[id] || [], elt);
      stamp.style.pointerEvents = "auto";
      rail.appendChild(stamp);
    });
  };

  Widget.prototype.layoutRail = function () {
    if (!this.rail || !this.articleEl) return;
    const rect = this.articleEl.getBoundingClientRect();
    const scrollY = window.scrollY || window.pageYOffset;
    const scrollX = window.scrollX || window.pageXOffset;
    const railWidth = 280;
    const gap = 32;
    this.rail.style.left = (rect.right + scrollX + gap) + "px";
    this.rail.style.top  = (rect.top + scrollY) + "px";
    this.rail.style.width = railWidth + "px";
    this.rail.style.height = rect.height + "px";

    const map = {};
    this.articleEl.querySelectorAll("[data-errolan-anchor]").forEach(elt => {
      map[elt.getAttribute("data-errolan-anchor")] = elt;
    });
    Array.from(this.rail.children).forEach(stamp => {
      const id = stamp.getAttribute("data-anchor");
      const elt = map[id];
      if (!elt) return;
      const top = elt.getBoundingClientRect().top - rect.top;
      stamp.style.position = "absolute";
      stamp.style.top = top + "px";
      stamp.style.left = "0";
      stamp.style.right = "0";
    });
  };

  Widget.prototype.renderStamp = function (anchorID, comments, paragraphEl) {
    const widget = this;
    const total = comments.reduce((s, c) => s + 1 + ((c.replies || []).length), 0);
    const tally = {};
    comments.forEach(c => {
      Object.entries(c.reactions || {}).forEach(([k, v]) => tally[k] = (tally[k] || 0) + v);
      (c.replies || []).forEach(r => Object.entries(r.reactions || {}).forEach(([k, v]) => tally[k] = (tally[k] || 0) + v));
    });
    const top = Object.entries(tally).sort((a, b) => b[1] - a[1]).slice(0, 3);

    const stamp = el("button", {
      class: "erl-stamp" + (total === 0 ? " erl-stamp-empty" : "")
        + (this.activeAnchor === anchorID ? " erl-stamp-active" : ""),
      "data-anchor": anchorID,
      onClick: () => widget.openParagraphPanel(anchorID, paragraphEl),
      onMouseEnter: () => paragraphEl.classList.add("erl-anchor-hot"),
      onMouseLeave: () => paragraphEl.classList.remove("erl-anchor-hot"),
    });
    if (total === 0) {
      stamp.appendChild(el("span", { class: "erl-stamp-add", text: "+ Add a note" }));
    } else {
      const rxRow = el("span", { class: "erl-stamp-rx" });
      top.forEach(([code, count]) => {
        const e = widget.packByCode[code];
        if (e) rxRow.insertAdjacentHTML("beforeend",
          emojiHTML(e, 14) + '<span class="erl-stamp-rx-count">' + count + "</span>");
      });
      stamp.appendChild(rxRow);
      stamp.appendChild(el("span", { class: "erl-stamp-meta",
        text: total + " " + (total === 1 ? "note" : "notes") }));
      stamp.appendChild(el("span", { class: "erl-stamp-preview",
        text: '"' + comments[0].body.slice(0, 90) + (comments[0].body.length > 90 ? "…" : "") + '"' }));
      stamp.appendChild(el("span", { class: "erl-stamp-author", text: "— @" + comments[0].author_name }));
    }
    return stamp;
  };

  Widget.prototype.openParagraphPanel = function (anchorID, paragraphEl) {
    this.activeAnchor = anchorID;
    if (paragraphEl) {
      this.articleEl.querySelectorAll("[data-errolan-anchor]")
        .forEach(p => p.classList.remove("erl-anchor-active"));
      paragraphEl.classList.add("erl-anchor-active");
    }
    if (this.panel && this.panel.parentNode) this.panel.parentNode.removeChild(this.panel);

    const widget = this;
    const overlay = el("div", { class: "erl-panel-overlay",
      onClick: (ev) => { if (ev.target === overlay) widget.closeParagraphPanel(); }});
    const panel = el("aside", { class: "erl-panel" });
    panel.appendChild(el("button", {
      class: "erl-panel-close", text: "×", title: "Close (Esc)",
      onClick: () => widget.closeParagraphPanel(),
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
      comments.forEach(c => list.appendChild(this.renderComment(c)));
    }
    panel.appendChild(list);

    if (this.state.thread && !this.state.thread.locked) {
      panel.appendChild(this.renderComposer(null, anchorID));
    } else if (this.state.thread && this.state.thread.locked) {
      panel.appendChild(el("div", { class: "erl-locked", text: "This thread is locked." }));
    }

    overlay.appendChild(panel);
    document.body.appendChild(overlay);
    this.panel = overlay;
    this._escHandler = (ev) => { if (ev.key === "Escape") this.closeParagraphPanel(); };
    document.addEventListener("keydown", this._escHandler);
  };
  Widget.prototype.closeParagraphPanel = function () {
    if (this.panel && this.panel.parentNode) this.panel.parentNode.removeChild(this.panel);
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
    if (this.mode === "marginalia") { this.buildRail(); this.layoutRail(); }
  };

  // ---------- Shared bits ----------

  Widget.prototype.renderSortControl = function () {
    const widget = this;
    const sel = el("select", { class: "erl-sort" });
    [["best","Best"],["newest","Newest"],["oldest","Oldest"]].forEach(([v, label]) => {
      const opt = el("option", { value: v, text: label });
      if (v === widget.sort) opt.selected = "selected";
      sel.appendChild(opt);
    });
    sel.addEventListener("change", () => { widget.sort = sel.value; widget.etag = null; widget.refresh(); });
    return sel;
  };

  Widget.prototype.loadMore = async function () {
    if (!this.state || !this.state.next_id) return;
    const params = { title: this.opts.title, url: this.opts.url,
      sort: this.sort, limit: 100, before_id: this.state.next_id };
    try {
      const more = await this.client.getThread(this.opts.thread, params);
      this.state.comments = (this.state.comments || []).concat(more.comments || []);
      this.state.has_more = more.has_more;
      this.state.next_id = more.next_id || null;
      this.render();
    } catch (e) { alert(e.message); }
  };

  Widget.prototype.renderAuthBar = function () {
    const widget = this;
    if (this.viewer) {
      return el("div", { class: "erl-auth" }, [
        el("span", { class: "erl-user", text: "@" + this.viewer.name }),
        this.viewer.is_admin ? el("span", { class: "erl-badge", text: "admin" }) : null,
        el("button", { class: "erl-link", text: "sign out", onClick: () => {
          widget.client.setToken(""); widget.etag = null; widget.refresh();
        }}),
      ]);
    }
    return el("div", { class: "erl-auth" }, [
      el("button", { class: "erl-link", text: "sign in", onClick: () => widget.openAuthDialog("login") }),
      el("span", { class: "erl-divider-dot", text: "·" }),
      el("button", { class: "erl-link", text: "register", onClick: () => widget.openAuthDialog("register") }),
    ]);
  };

  Widget.prototype.openAuthDialog = function (mode) {
    const widget = this;
    const overlay = el("div", { class: "erl-overlay" });
    const dialog = el("div", { class: "erl-dialog" });
    dialog.appendChild(el("h4", { text: mode === "login" ? "Sign in" : "Create an account" }));
    const email = el("input", { type: "email", placeholder: "email", required: true });
    const name  = mode === "register" ? el("input", { type: "text", placeholder: "display name", required: true }) : null;
    const pw    = el("input", { type: "password", placeholder: "password (≥8 chars)", required: true });
    const err   = el("div", { class: "erl-error" });
    const submit = el("button", { class: "erl-primary", text: mode === "login" ? "Sign in" : "Create" });
    const cancel = el("button", { class: "erl-link", text: "cancel",
      onClick: () => document.body.removeChild(overlay) });
    const switcher = el("button", { class: "erl-link",
      text: mode === "login" ? "need an account?" : "have an account?",
      onClick: () => { document.body.removeChild(overlay); widget.openAuthDialog(mode === "login" ? "register" : "login"); }});
    submit.addEventListener("click", async () => {
      err.textContent = "";
      try {
        const res = mode === "login"
          ? await widget.client.login(email.value, pw.value)
          : await widget.client.register(email.value, name.value, pw.value);
        widget.client.setToken(res.token);
        document.body.removeChild(overlay);
        widget.etag = null;
        widget.refresh();
      } catch (e) { err.textContent = e.message; }
    });
    overlay.addEventListener("click", (ev) => { if (ev.target === overlay) document.body.removeChild(overlay); });
    dialog.appendChild(email);
    if (name) dialog.appendChild(name);
    dialog.appendChild(pw);
    dialog.appendChild(err);
    dialog.appendChild(el("div", { class: "erl-actions" }, [cancel, submit]));
    dialog.appendChild(switcher);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    email.focus();
  };

  Widget.prototype.renderComposer = function (parentID, anchor) {
    const widget = this;
    const wrap = el("form", { class: "erl-composer" });
    const requireAuth = this.state.site && this.state.site.require_auth;
    if (requireAuth && !this.viewer) {
      wrap.appendChild(el("div", { class: "erl-muted", text: "Please sign in to comment." }));
      return wrap;
    }

    if (anchor) {
      wrap.appendChild(el("div", { class: "erl-eyebrow",
        text: "Add a note to paragraph " + anchor }));
    }

    let nameInput = null;
    if (!this.viewer) {
      nameInput = el("input", { type: "text", class: "erl-name", placeholder: "Your name",
        value: localStorage.getItem(NAME_KEY) || "" });
      wrap.appendChild(nameInput);
    }
    const textarea = el("textarea", {
      class: "erl-body",
      placeholder: parentID ? "Write a reply…" : (anchor ? "Write in the margin…" : "Write a thoughtful note. Use :code: for emoji."),
      rows: parentID ? 2 : 3,
    });
    wrap.appendChild(textarea);
    const honeypot = el("input", { type: "text", name: "website", class: "erl-hp",
      autocomplete: "off", tabindex: "-1" });
    wrap.appendChild(honeypot);

    if (Object.keys(this.packByCode).length > 0) {
      const picker = el("div", { class: "erl-picker" });
      Object.values(this.packByCode).forEach(em => {
        const b = el("button", { type: "button", class: "erl-picker-btn",
          title: ":" + em.code + ":", html: emojiHTML(em, 18),
          onClick: () => {
            const before = textarea.value.slice(0, textarea.selectionStart);
            const after  = textarea.value.slice(textarea.selectionEnd);
            textarea.value = before + ":" + em.code + ":" + after;
            textarea.focus();
          }});
        picker.appendChild(b);
      });
      wrap.appendChild(picker);
    }

    const err = el("div", { class: "erl-error" });
    const submit = el("button", { type: "submit", class: "erl-primary",
      text: parentID ? "Reply" : (anchor ? "Post note" : "Post") });
    wrap.appendChild(err);
    wrap.appendChild(el("div", { class: "erl-actions" }, [submit]));

    wrap.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      err.textContent = "";
      const body = textarea.value.trim();
      if (!body) return;
      if (honeypot.value) return;
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
            '[data-errolan-anchor="' + (window.CSS && CSS.escape ? CSS.escape(widget.activeAnchor) : widget.activeAnchor) + '"]'
          );
          widget.openParagraphPanel(widget.activeAnchor, elNode);
        }
      } catch (e) { err.textContent = e.message; }
      finally { submit.disabled = false; }
    });
    return wrap;
  };

  Widget.prototype.renderComment = function (c) {
    const widget = this;
    const li = el("li", { class: "erl-c" + (c.pinned ? " erl-c-pinned" : ""), "data-id": c.id });

    const avatar = el("div", { class: "erl-avatar",
      style: c.avatar_url ? ("background-image:url(" + c.avatar_url + ")") : "",
      text: c.avatar_url ? "" : (c.author_name || "?").charAt(0).toUpperCase() });
    const meta = el("div", { class: "erl-c-meta" }, [
      el("span", { class: "erl-c-author", text: "@" + c.author_name }),
      c.pinned ? el("span", { class: "erl-pin-badge", text: "★ pinned" }) : null,
      el("span", { class: "erl-c-time",
        text: relTime(c.created_at) + (c.edit_count ? " · edited" : "") }),
    ]);
    li.appendChild(el("div", { class: "erl-c-head" }, [avatar, meta]));

    const bodyNode = this.editing === c.id
      ? this.renderEditor(c)
      : el("div", { class: "erl-c-body", html: formatBody(c.body, this.packByCode) });
    li.appendChild(bodyNode);

    // Reactions
    const rxStrip = el("div", { class: "erl-rx" });
    const myReacts = new Set(c.my_reacts || []);
    const counts = c.reactions || {};
    Object.values(this.packByCode).forEach(em => {
      const count = counts[em.code] || 0;
      if (count === 0 && !myReacts.has(em.code)) return;
      const b = el("button", {
        class: "erl-rx-btn" + (myReacts.has(em.code) ? " erl-rx-active" : ""),
        title: ":" + em.code + ":",
        html: emojiHTML(em, 14) + '<span class="erl-rx-count">' + count + "</span>",
        onClick: () => widget.handleReact(c, em.code),
      });
      rxStrip.appendChild(b);
    });
    rxStrip.appendChild(el("button", { class: "erl-rx-add", text: "+ react",
      onClick: (ev) => widget.openReactionPicker(c, ev.currentTarget) }));
    li.appendChild(rxStrip);

    const actions = el("div", { class: "erl-c-actions" }, [
      !this.state.thread.locked
        ? el("button", { class: "erl-link", text: "reply",
            onClick: () => { widget.replyTo = c.id; widget.render(); } })
        : null,
      this.canEdit(c)
        ? el("button", { class: "erl-link", text: "edit",
            onClick: () => { widget.editing = c.id; widget.render(); } })
        : null,
      this.canEdit(c) && c.status !== "deleted"
        ? el("button", { class: "erl-link", text: "delete",
            onClick: () => widget.handleDelete(c) })
        : null,
      this.viewer && this.viewer.is_admin
        ? el("button", { class: "erl-link", text: c.pinned ? "unpin" : "pin",
            onClick: () => widget.handlePin(c) })
        : null,
      el("button", { class: "erl-link", text: "flag", onClick: () => widget.handleFlag(c) }),
    ]);
    li.appendChild(actions);

    if (this.replyTo === c.id && !this.state.thread.locked) {
      li.appendChild(this.renderComposer(c.id, c.anchor || ""));
    }
    if (c.replies && c.replies.length) {
      const ul = el("ul", { class: "erl-list erl-replies" });
      for (const r of c.replies) ul.appendChild(this.renderComment(r));
      li.appendChild(ul);
    }
    return li;
  };

  Widget.prototype.renderEditor = function (c) {
    const widget = this;
    const wrap = el("form", { class: "erl-composer" });
    const ta = el("textarea", { class: "erl-body", rows: 3 });
    ta.value = c.body;
    const err = el("div", { class: "erl-error" });
    wrap.appendChild(ta);
    wrap.appendChild(err);
    wrap.appendChild(el("div", { class: "erl-actions" }, [
      el("button", { type: "submit", class: "erl-primary", text: "Save" }),
      el("button", { type: "button", class: "erl-link", text: "cancel",
        onClick: () => { widget.editing = null; widget.render(); }}),
    ]));
    wrap.addEventListener("submit", async (ev) => {
      ev.preventDefault();
      err.textContent = "";
      try {
        await widget.client.editComment(c.id, ta.value.trim());
        widget.editing = null; widget.etag = null;
        await widget.refresh();
      } catch (e) { err.textContent = e.message; }
    });
    return wrap;
  };

  Widget.prototype.canEdit = function (c) {
    if (!this.viewer) return false;
    if (this.viewer.is_admin) return true;
    return c.user_id === this.viewer.id;
  };

  Widget.prototype.handleReact = async function (c, code) {
    if (!this.viewer) { this.openAuthDialog("login"); return; }
    try { await this.client.react(c.id, code); this.etag = null; await this.refresh(); }
    catch (e) { alert(e.message); }
  };
  Widget.prototype.handleDelete = async function (c) {
    if (!confirm("Delete this comment?")) return;
    try { await this.client.deleteComment(c.id); this.etag = null; await this.refresh(); }
    catch (e) { alert(e.message); }
  };
  Widget.prototype.handlePin = async function (c) {
    try { await this.client.pin(c.id, !c.pinned); this.etag = null; await this.refresh(); }
    catch (e) { alert(e.message); }
  };
  Widget.prototype.handleFlag = async function (c) {
    const reason = prompt("Flag this comment — reason (optional):");
    if (reason === null) return;
    try { await this.client.flag(c.id, reason); alert("Thanks — a moderator will review it."); }
    catch (e) { alert(e.message); }
  };

  Widget.prototype.openReactionPicker = function (c, anchorBtn) {
    const widget = this;
    if (!this.viewer) { this.openAuthDialog("login"); return; }
    const old = document.querySelector(".erl-rx-popover");
    if (old) old.remove();
    const rect = anchorBtn.getBoundingClientRect();
    const pop = el("div", { class: "erl-rx-popover" });
    pop.style.position = "absolute";
    pop.style.top  = (rect.bottom + window.scrollY + 6) + "px";
    pop.style.left = (rect.left + window.scrollX) + "px";
    Object.values(this.packByCode).forEach(em => {
      pop.appendChild(el("button", { type: "button", class: "erl-rx-popover-btn",
        title: ":" + em.code + ":", html: emojiHTML(em, 18),
        onClick: () => { pop.remove(); widget.handleReact(c, em.code); }}));
    });
    document.body.appendChild(pop);
    setTimeout(() => {
      const close = (e) => { if (!pop.contains(e.target)) { pop.remove(); document.removeEventListener("click", close); } };
      document.addEventListener("click", close);
    }, 0);
  };

  // ---------- Mounting ----------

  function mount(element, opts) {
    if (!opts || !opts.api || !opts.site || !opts.thread) {
      throw new Error("Errolan: api, site, and thread are required");
    }
    const w = new Widget(element, opts);
    w.mount();
    return w;
  }

  function autoMount() {
    document.querySelectorAll("[data-errolan-thread]").forEach(n => {
      if (n.__erlMounted) return;
      n.__erlMounted = true;
      mount(n, {
        api: n.getAttribute("data-errolan-api") || global.ERROLAN_API || "",
        site: n.getAttribute("data-errolan-site") || global.ERROLAN_SITE || "",
        thread: n.getAttribute("data-errolan-thread"),
        title: n.getAttribute("data-errolan-title") || document.title,
        url: n.getAttribute("data-errolan-url") || location.href,
        sort: n.getAttribute("data-errolan-sort") || "best",
        mode: n.getAttribute("data-errolan-mode") || "cadence",
        article: n.getAttribute("data-errolan-article") || "",
        live: n.getAttribute("data-errolan-live") !== "false",
      });
    });
  }

  global.Errolan = { mount, Client };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})(window);
