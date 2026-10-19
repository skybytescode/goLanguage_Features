// Căutarea în tot conținutul și butoanele de copiere.
//
// Indexul (search/<limbă>.json) se încarcă la prima căutare. Textul e
// normalizat caracter cu caracter (litere mici, fără diacritice), așa că
// pozițiile din textul normalizat corespund celor din original și fragmentul
// afișat poate fi decupat direct.
(() => {
  "use strict";

  const MIN_QUERY = 2;
  const SNIPPET_RADIUS = 70;
  const MAX_RESULTS = 30;

  const body = document.body;
  const input = document.querySelector(".search");
  const results = document.querySelector(".results");
  const count = document.querySelector(".results-count");
  const list = document.querySelector(".results-list");
  const page = document.querySelector(".page");

  // --- copiere ---------------------------------------------------------------

  document.addEventListener("click", async (ev) => {
    const btn = ev.target.closest(".copy");
    if (!btn) return;
    const code = btn.closest(".code").querySelector("pre").innerText;
    try {
      await navigator.clipboard.writeText(code);
      const label = btn.textContent;
      btn.textContent = btn.dataset.copied;
      setTimeout(() => (btn.textContent = label), 1500);
    } catch {
      /* fără acces la clipboard: nu facem nimic */
    }
  });

  // --- căutare ---------------------------------------------------------------

  // Un caracter normalizat pentru fiecare caracter din original.
  const fold = (s) =>
    Array.from(s, (c) => {
      const base = c.normalize("NFD")[0].toLowerCase();
      return base.length === 1 ? base : c;
    });

  const find = (hay, needle, from = 0) => {
    outer: for (let i = from; i <= hay.length - needle.length; i++) {
      for (let j = 0; j < needle.length; j++) {
        if (hay[i + j] !== needle[j]) continue outer;
      }
      return i;
    }
    return -1;
  };

  const countIn = (hay, needle, max) => {
    let n = 0;
    for (let i = find(hay, needle); i >= 0 && n < max; i = find(hay, needle, i + needle.length)) n++;
    return n;
  };

  let index = null;
  let loading = null;

  function loadIndex() {
    loading ??= fetch(`${body.dataset.base}search/${body.dataset.lang}.json`)
      .then((r) => r.json())
      .then((entries) => {
        index = entries.map((e, order) => ({
          ...e,
          order,
          title: fold(e.t),
          headings: e.h.map(fold),
          chars: Array.from(e.x),
          folded: fold(e.x),
        }));
      });
    return loading;
  }

  function search(query) {
    const terms = query.split(/\s+/).filter(Boolean).map(fold);
    if (!terms.length) return [];
    const scored = [];
    for (const e of index) {
      let titleTerms = 0;
      let score = 0;
      let ok = true;
      for (const t of terms) {
        const inTitle = find(e.title, t) >= 0;
        const inHeadings = e.headings.filter((h) => find(h, t) >= 0).length;
        const inText = countIn(e.folded, t, 10);
        if (!inTitle && !inHeadings && !inText) {
          ok = false; // toate cuvintele trebuie să apară
          break;
        }
        titleTerms += inTitle ? 1 : 0;
        score += Math.min(inHeadings, 3) * 5 + inText;
      }
      if (ok) scored.push({ e, titleTerms, score });
    }
    // Întâi subiectele al căror titlu se potrivește, în ordinea nivelurilor;
    // apoi restul, după scor; la egalitate, ordinea din cuprins.
    scored.sort((a, b) => {
      if (a.titleTerms !== b.titleTerms) return b.titleTerms - a.titleTerms;
      if (a.titleTerms === 0 && a.score !== b.score) return b.score - a.score;
      return a.e.order - b.e.order;
    });
    return scored.slice(0, MAX_RESULTS).map(({ e }) => ({ e, snippet: snippet(e, terms) }));
  }

  const isSpace = (c) => c !== undefined && /\s/.test(c);
  const collapse = (chars) => chars.join("").split(/\s+/).filter(Boolean).join(" ");

  function snippet(e, terms) {
    let pos = -1;
    let len = 0;
    for (const t of terms) {
      pos = find(e.folded, t);
      if (pos >= 0) {
        len = t.length;
        break;
      }
    }
    const text = e.chars;
    if (pos < 0) return [collapse(text.slice(0, SNIPPET_RADIUS * 2)), "", "…"];

    let from = Math.max(0, pos - SNIPPET_RADIUS);
    let to = Math.min(text.length, pos + len + SNIPPET_RADIUS);
    // nu tăia cuvintele la margini
    while (from > 0 && from < pos && !isSpace(text[from - 1])) from++;
    while (to < text.length && to > pos + len && !isSpace(text[to])) to--;

    let before = collapse(text.slice(from, pos));
    const matched = text.slice(pos, pos + len).join("");
    let after = collapse(text.slice(pos + len, to));
    if (from > 0) before = "…" + before;
    if (to < text.length) after += "…";
    // păstrează spațiile din jurul potrivirii, eliminate de `collapse`
    if (before && isSpace(text[pos - 1])) before += " ";
    if (after && isSpace(text[pos + len])) after = " " + after;
    return [before, matched, after];
  }

  function countLabel(n, query) {
    const d = body.dataset;
    if (n === 0) return d.noResults.replace("%s", query);
    if (n === 1) return d.oneResult;
    // română: „19 rezultate”, dar „20 de rezultate”
    const rem = n % 100;
    return (rem >= 1 && rem < 20 ? d.manyResults : d.manyResultsDe).replace("%d", n);
  }

  function el(tag, cls, text) {
    const node = document.createElement(tag);
    if (cls) node.className = cls;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function render() {
    const query = input.value.trim();
    const active = Array.from(query).length >= MIN_QUERY;
    results.hidden = !active;
    page.hidden = active;
    if (!active) return;
    if (!index) {
      loadIndex().then(render);
      return;
    }
    const hits = search(query);
    count.textContent = countLabel(hits.length, query);
    list.replaceChildren(
      ...hits.map(({ e, snippet: [before, matched, after] }) => {
        const a = el("a", "result");
        a.href = e.u;
        const title = el("span", "result-title", e.t);
        const level = el("span", `result-level lvl-${e.l}`);
        level.append(el("span", "level-num", String(e.l)), e.n);
        title.append(level);
        const snip = el("span", "result-snippet");
        snip.append(before, el("mark", "", matched), after);
        a.append(title, snip);
        return a;
      }),
    );
  }

  input.addEventListener("input", render);
  input.addEventListener("keydown", (ev) => {
    if (ev.key === "Escape") {
      input.value = "";
      render();
    }
  });
  // `/` mută cursorul în căutare, dacă nu scrii deja într-un câmp
  document.addEventListener("keydown", (ev) => {
    const tag = document.activeElement?.tagName;
    if (ev.key === "/" && tag !== "INPUT" && tag !== "TEXTAREA") {
      ev.preventDefault();
      input.focus();
    }
  });
  // la revenirea cu „Back”, browserul poate păstra textul din câmp
  if (input.value) render();

  // --- aplicație instalabilă și offline ----------------------------------------

  if (!("serviceWorker" in navigator)) return;
  navigator.serviceWorker.register(`${body.dataset.base}sw.js`).catch(() => {
    /* fără service worker, site-ul funcționează normal, doar online */
  });

  const btn = document.querySelector(".offline-btn");
  const status = document.querySelector(".offline-status");
  if (!btn) return;
  btn.hidden = false;
  btn.addEventListener("click", async () => {
    const reg = await navigator.serviceWorker.ready;
    if (!reg.active) return;
    btn.disabled = true;
    reg.active.postMessage({ type: "download", lang: body.dataset.lang });
  });
  navigator.serviceWorker.addEventListener("message", (ev) => {
    const d = body.dataset;
    const msg = ev.data || {};
    if (msg.type === "progress") {
      status.textContent = d.offlineProgress.replace("%d", msg.done).replace("%d", msg.total);
    } else if (msg.type === "downloaded") {
      status.textContent = msg.failed ? d.offlineFailed.replace("%d", msg.failed) : d.offlineDone;
      btn.disabled = false;
    }
  });
})();
