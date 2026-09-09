import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';
import { renderDocumentView } from './views/document';
import { renderSetariView } from './views/setari';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <aside class="sidebar" id="sidebar"></aside>
  <main class="main" id="outlet"></main>
`;

const sidebar = document.getElementById('sidebar') as HTMLElement;
const outlet = document.getElementById('outlet') as HTMLElement;

/** Re-reads the document history; called after any save or delete. */
export function refreshSidebar(): Promise<void> {
  return renderSidebar(sidebar);
}

void refreshSidebar();

startRouter(
  [
    {
      // The bare draft route has no template on it — a stale hash, or the
      // empty-hash fallback on a fresh launch. renderDocumentView sends it to
      // the first template rather than failing.
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar, undefined),
    },
    {
      pattern: /^#\/document\/new\/(\d+)$/,
      render: (el, templateId) =>
        renderDocumentView(el, undefined, refreshSidebar, Number(templateId)),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar, undefined),
    },
    {
      pattern: /^#\/setari$/,
      render: (el) => renderSetariView(el, refreshSidebar),
    },
  ],
  outlet,
);
