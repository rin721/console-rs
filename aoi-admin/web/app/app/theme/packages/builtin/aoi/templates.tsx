import { ChevronDown, Globe2, LayoutDashboard, ShieldCheck } from "lucide-react";
import { useId, useMemo, useState, type ReactNode } from "react";
import { Link, NavLink, Outlet, useLocation } from "react-router";
import { useTranslation } from "react-i18next";

import { PreferenceMenu } from "~/components/aoi/patterns/PreferenceMenu";
import { RequireAuth } from "~/features/auth/RequireAuth";
import { AdminHeader } from "~/features/admin/AdminHeader";
import { adminNavGroups, findAdminNavGroupId } from "~/features/admin/navigation";
import { usePublicSettings } from "~/hooks/usePublicSettings";
import type { AppLocale } from "~/i18n/resources";
import { supportedLocales } from "~/i18n/locales";

type TemplateProps = {
  children?: ReactNode;
};

export function PublicThemeLayout() {
  const { t } = useTranslation();
  const { brandName } = usePublicSettings();

  return (
    <div className="aoi-public-shell">
      <header className="aoi-public-header">
        <div className="aoi-public-header__inner">
          <Link className="aoi-brand" to="/">
            <span className="aoi-brand__mark" aria-hidden="true">
              <Globe2 size={18} />
            </span>
            <span>{brandName}</span>
          </Link>
          <nav className="aoi-nav" aria-label={t("a11y.primaryNavigation")}>
            <NavLink to="/">{t("site.nav.home")}</NavLink>
            <NavLink to="/about">{t("site.nav.about")}</NavLink>
            <NavLink to="/blog">{t("site.nav.blog")}</NavLink>
            <NavLink to="/login">{t("common.actions.login")}</NavLink>
            <NavLink to="/admin">{t("site.nav.admin")}</NavLink>
          </nav>
          <PreferenceMenu />
        </div>
      </header>
      <Outlet />
      <footer className="aoi-footer">
        <div className="aoi-footer__inner">
          <p>{t("site.footer.description")}</p>
          <nav className="aoi-footer__links" aria-label={t("a11y.footerNavigation")}>
            <Link to="/terms">{t("site.nav.terms")}</Link>
            <Link to="/privacy">{t("site.nav.privacy")}</Link>
          </nav>
          <p>
            {brandName} {t("site.footer.copyright")}
          </p>
        </div>
      </footer>
    </div>
  );
}

export function AdminThemeLayout() {
  const location = useLocation();
  const { t } = useTranslation();

  return (
    <RequireAuth>
      <div className="aoi-admin-shell">
        <aside className="aoi-admin-sidebar">
          <div className="aoi-brand">
            <span className="aoi-brand__mark" aria-hidden="true">
              <LayoutDashboard size={18} />
            </span>
            <span>{t("site.nav.admin")}</span>
          </div>
          <AdminSidebarNav pathname={location.pathname} />
        </aside>
        <div className="aoi-admin-content">
          <AdminHeader pathname={location.pathname} />
          <main className="aoi-admin-main">
            <Outlet />
          </main>
        </div>
      </div>
    </RequireAuth>
  );
}

export function AdminSidebarNav({ pathname }: { pathname: string }) {
  const { t } = useTranslation();
  const baseId = useId();
  const activeGroupId = useMemo(() => findAdminNavGroupId(pathname), [pathname]);
  const [navState, setNavState] = useState(() => ({
    openGroupId: activeGroupId,
    routeGroupId: activeGroupId,
  }));
  const openGroupId =
    navState.routeGroupId === activeGroupId ? navState.openGroupId : activeGroupId;

  return (
    <nav className="aoi-admin-nav" aria-label={t("a11y.adminNavigation")}>
      {adminNavGroups.map((group) => {
        const GroupIcon = group.icon;
        const open = group.id === openGroupId;
        const active = group.id === activeGroupId;
        const contentId = `${baseId}-${group.id}`;

        return (
          <section
            className="aoi-admin-nav-group"
            data-active={active ? "true" : "false"}
            data-state={open ? "open" : "closed"}
            key={group.id}
          >
            <button
              aria-controls={contentId}
              aria-expanded={open}
              className="aoi-admin-nav-group__trigger"
              type="button"
              onClick={() =>
                setNavState({
                  openGroupId: group.id,
                  routeGroupId: activeGroupId,
                })
              }
            >
              <span className="aoi-admin-nav-group__label">
                <GroupIcon aria-hidden="true" size={17} />
                <span>{t(group.labelKey)}</span>
              </span>
              <ChevronDown aria-hidden="true" className="aoi-admin-nav-group__chevron" size={16} />
            </button>
            {open ? (
              <div
                aria-label={t(group.labelKey)}
                className="aoi-admin-nav-group__content"
                id={contentId}
                role="group"
              >
                {group.items.map((item) => {
                  const ItemIcon = item.icon;

                  return (
                    <NavLink
                      className="aoi-admin-nav__link"
                      end={item.end}
                      key={item.id}
                      to={item.to}
                    >
                      <ItemIcon aria-hidden="true" size={17} />
                      <span>{t(item.labelKey)}</span>
                    </NavLink>
                  );
                })}
              </div>
            ) : null}
          </section>
        );
      })}
    </nav>
  );
}

export function SetupThemeLayout() {
  const { i18n, t } = useTranslation();

  return (
    <div className="aoi-setup-shell">
      <header className="aoi-setup-header">
        <div className="aoi-setup-header__inner">
          <div className="aoi-brand">
            <span className="aoi-brand__mark" aria-hidden="true">
              <ShieldCheck size={18} />
            </span>
            <span>{t("setup.title")}</span>
          </div>
          <select
            className="aoi-language-select"
            aria-label={t("a11y.languageSwitcher")}
            value={i18n.language}
            onChange={(event) => {
              const locale = event.target.value;
              if (supportedLocales.includes(locale as AppLocale)) {
                void i18n.changeLanguage(locale);
              }
            }}
          >
            {supportedLocales.map((locale) => (
              <option key={locale} value={locale}>
                {locale}
              </option>
            ))}
          </select>
        </div>
      </header>
      <Outlet />
    </div>
  );
}

export function AuthThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function DashboardThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function DetailPageThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function ErrorThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function ListPageThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function LoadingThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}

export function SettingsThemeTemplate({ children }: TemplateProps) {
  return <>{children}</>;
}
