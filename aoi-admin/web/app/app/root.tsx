import {
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  isRouteErrorResponse,
  useRouteError,
} from "react-router";

import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { AppProviders } from "~/providers/AppProviders";
import { fallbackLanguage, resources } from "~/i18n/resources";

import "./components/aoi/tokens/tokens.css";
import "./styles/app.css";

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang={fallbackLanguage}>
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <title>{resources[fallbackLanguage].seo.home.title}</title>
        <meta name="description" content={resources[fallbackLanguage].seo.home.description} />
        <meta property="og:type" content="website" />
        <meta property="og:title" content={resources[fallbackLanguage].seo.home.title} />
        <meta
          property="og:description"
          content={resources[fallbackLanguage].seo.home.description}
        />
        <meta name="twitter:card" content="summary" />
        <Meta />
        <Links />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export function HydrateFallback() {
  return (
    <div className="aoi-hydrate" role="status" aria-live="polite">
      {resources[fallbackLanguage].loading.app}
    </div>
  );
}

export default function App() {
  return (
    <AppProviders>
      <Outlet />
    </AppProviders>
  );
}

export function ErrorBoundary() {
  const error = useRouteError();
  const title = isRouteErrorResponse(error)
    ? resources[fallbackLanguage].errors.route.title
    : resources[fallbackLanguage].errors.route.unexpectedTitle;
  const description = isRouteErrorResponse(error)
    ? resources[fallbackLanguage].errors.route.description
    : resources[fallbackLanguage].errors.route.unexpectedDescription;

  return (
    <AppProviders>
      <main className="aoi-page aoi-page--narrow">
        <StateBlock intent="danger" title={title} description={description} />
      </main>
    </AppProviders>
  );
}
