import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';
import { renderDocumentView } from './views/document';

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
      pattern: /^#\/document\/new$/,
      render: (el) => renderDocumentView(el, undefined, refreshSidebar),
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => renderDocumentView(el, id, refreshSidebar),
    },
    {
      pattern: /^#\/setari$/,
      render: (el) => {
        el.innerHTML = '<p class="empty">Setări (în curs de implementare)</p>';
      },
    },
  ],
  outlet,
);
