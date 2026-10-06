"use strict";
const $ = (id) => document.getElementById(id);
let pages = [];
let busy = false;
function status(message) { $("status").textContent = message; }
async function api(path, options = {}) {
  const response = await fetch(path, { credentials: "same-origin", ...options });
  if (!response.ok) {
    let data = {}; try { data = await response.json(); } catch (_) {}
    throw new Error(data.error || `HTTP ${response.status}`);
  }
  return response.status === 204 ? null : response.json();
}
function render() {
  const query = $("search").value.toLocaleLowerCase("de");
  const selected = pages.filter(p => p.name.toLocaleLowerCase("de").includes(query));
  $("pages").replaceChildren(); $("empty").hidden = selected.length !== 0;
  for (const p of selected) {
    const row = document.createElement("tr");
    for (const value of [p.name, `${(p.size / 1024).toFixed(1)} KiB`, new Date(p.created).toLocaleString("de-DE")]) {
      const cell = document.createElement("td"); cell.textContent = value; row.append(cell);
    }
    const actions = document.createElement("td");
    const link = document.createElement("a");
    link.textContent = "Öffnen"; link.className = "action"; link.href = p.url;
    link.target = "_blank"; link.rel = "noopener noreferrer"; actions.append(link);
    const copy = document.createElement("button"); copy.textContent = "Link kopieren";
    copy.onclick = async () => {
      try { await navigator.clipboard.writeText(p.url); status("Link kopiert."); }
      catch (_) { status(`Link: ${p.url}`); }
    }; actions.append(copy);
    const remove = document.createElement("button"); remove.textContent = "Löschen"; remove.className = "danger";
    remove.onclick = async () => {
      if (!confirm(`„${p.name}“ endgültig löschen?`)) return;
      remove.disabled = true;
      try { await api(`/api/pages/${p.id}`, {method: "DELETE"}); await load(); status("Datei gelöscht."); }
      catch (error) { status(error.message); remove.disabled = false; }
    }; actions.append(remove);
    const change = document.createElement("button"); change.textContent = "Link ändern";
    change.onclick = async () => {
      const value = prompt("Eigener URL-Name ohne .html (1–64 Kleinbuchstaben, Ziffern, Bindestriche). Leer setzt den Link zurück. Der bisherige eigene Link wird ersetzt; der ID-Link bleibt gültig.", p.slug || "");
      if (value === null) return;
      const slug = value.trim();
      if (slug && (!/^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$/.test(slug) || /^[a-f0-9]{32}$/.test(slug))) {
        status("Ungültiger URL-Name. Keine Pfade, Dateiendungen, Großbuchstaben oder IDs verwenden."); return;
      }
      change.disabled = true;
      try {
        const saved = await api(`/api/pages/${encodeURIComponent(p.id)}/link`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify({slug})});
        await load(); status(`Link gespeichert: ${saved.url}`);
      } catch (error) { status(error.message); }
      finally { change.disabled = false; }
    }; actions.append(change);
    const download = document.createElement("a"); download.textContent = "Herunterladen";
    download.className = "action"; download.href = `/api/pages/${encodeURIComponent(p.id)}/download`;
    download.download = p.name; actions.append(download);
    row.append(actions); $("pages").append(row);
  }
}
async function load() { pages = await api("/api/pages"); render(); }
function sendFile(file, progress) {
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest(); request.open("POST", "/api/upload");
    request.timeout = 35000;
    request.setRequestHeader("Content-Type", "text/html; charset=utf-8");
    request.setRequestHeader("X-File-Name", encodeURIComponent(file.name));
    request.upload.onprogress = e => { if (e.lengthComputable) progress(e.loaded / e.total); };
    request.onerror = () => reject(new Error("Netzwerkfehler"));
    request.ontimeout = () => reject(new Error("Upload-Zeitlimit überschritten"));
    request.onload = () => {
      let data = {}; try { data = JSON.parse(request.responseText); } catch (_) {}
      if (request.status === 201) resolve(data); else reject(new Error(data.error || `HTTP ${request.status}`));
    };
    request.send(file);
  });
}
async function upload(fileList) {
  if (busy) { status("Bitte den laufenden Import abwarten."); return; }
  const selected = Array.from(fileList); if (!selected.length) return;
  busy = true; $("files").disabled = true; $("progress").hidden = false;
  const results = [];
  try {
    for (let i = 0; i < selected.length; i++) {
      const file = selected[i];
      status(`Importiere ${i + 1}/${selected.length}: ${file.name}`);
      try {
        if (!/\.html?$/i.test(file.name)) throw new Error("Nur .html/.htm erlaubt");
        await sendFile(file, fraction => { $("progress").value = ((i + fraction) / selected.length) * 100; });
        results.push(`✓ ${file.name}`);
      } catch (error) { results.push(`✗ ${file.name}: ${error.message}`); }
      $("progress").value = ((i + 1) / selected.length) * 100;
    }
    await load(); status(results.join("\n"));
  } catch (error) { status([...results, error.message].join("\n")); }
  finally { busy = false; $("files").disabled = false; $("files").value = ""; $("progress").hidden = true; }
}
$("files").onchange = e => upload(e.target.files);
$("search").oninput = render;
$("refresh").onclick = () => load().catch(e => status(e.message));
const drop = $("drop");
drop.addEventListener("click", e => { if (e.target !== $("files") && !busy) $("files").click(); });
drop.addEventListener("keydown", e => { if (e.target === drop && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); if (!busy) $("files").click(); } });
for (const name of ["dragenter", "dragover"]) drop.addEventListener(name, e => { e.preventDefault(); drop.classList.add("drag"); });
for (const name of ["dragleave", "drop"]) drop.addEventListener(name, e => { e.preventDefault(); drop.classList.remove("drag"); });
drop.addEventListener("drop", e => upload(e.dataTransfer.files));
load().catch(e => status(e.message));
