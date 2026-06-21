import {
  AdminSidebarNav,
  AdminThemeLayout,
  AuthThemeTemplate,
  DashboardThemeTemplate,
  DetailPageThemeTemplate,
  ErrorThemeTemplate,
  ListPageThemeTemplate,
  LoadingThemeTemplate,
  PublicThemeLayout,
  SettingsThemeTemplate,
  SetupThemeLayout,
} from "../../builtin/aoi/templates";

export {
  AdminSidebarNav,
  AuthThemeTemplate,
  DashboardThemeTemplate,
  DetailPageThemeTemplate,
  ErrorThemeTemplate,
  ListPageThemeTemplate,
  LoadingThemeTemplate,
  SettingsThemeTemplate,
};

export function PublicThemeLayoutExample() {
  return (
    <div data-aoi-source-theme-template="example/custom">
      <PublicThemeLayout />
    </div>
  );
}

export function AdminThemeLayoutExample() {
  return (
    <div data-aoi-source-theme-template="example/custom">
      <AdminThemeLayout />
    </div>
  );
}

export function SetupThemeLayoutExample() {
  return (
    <div data-aoi-source-theme-template="example/custom">
      <SetupThemeLayout />
    </div>
  );
}

export {
  AdminThemeLayoutExample as AdminThemeLayout,
  PublicThemeLayoutExample as PublicThemeLayout,
  SetupThemeLayoutExample as SetupThemeLayout,
};
