import './style.css';
import { renderSidebar } from './sidebar';
import { startRouter } from './router';

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
      render: (el) => {
        el.innerHTML = '<p class="empty">Document nou (in curs de implementare)</p>';
      },
    },
    {
      pattern: /^#\/document\/(\d+)$/,
      render: (el, id) => {
        el.innerHTML = `<p class="empty">Document ${id} (in curs de implementare)</p>`;
      },
    },
    {
      pattern: /^#\/setari$/,
      render: (el) => {
        el.innerHTML = '<p class="empty">Setări (in curs de implementare)</p>';
      },
    },
  ],
  outlet,
);
