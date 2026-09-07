export type Route = {
  pattern: RegExp;
  render: (outlet: HTMLElement, ...params: string[]) => void | Promise<void>;
};

let currentRoutes: Route[] = [];
let currentOutlet: HTMLElement | null = null;

/** Navigates to a hash route, forcing a re-render even if the hash is unchanged. */
export function navigate(hash: string): void {
  if (window.location.hash === hash) {
    void handleRoute();
    return;
  }
  window.location.hash = hash;
}

async function handleRoute(): Promise<void> {
  if (!currentOutlet) return;
  const hash = window.location.hash || '#/document/new';

  for (const route of currentRoutes) {
    const match = route.pattern.exec(hash);
    if (match) {
      await route.render(currentOutlet, ...match.slice(1));
      return;
    }
  }
  currentOutlet.innerHTML = '<p class="empty">Pagina nu a fost găsită.</p>';
}

/** Installs the routes and renders the current hash. */
export function startRouter(routes: Route[], outlet: HTMLElement): void {
  currentRoutes = routes;
  currentOutlet = outlet;
  window.addEventListener('hashchange', () => {
    void handleRoute();
  });
  void handleRoute();
}
