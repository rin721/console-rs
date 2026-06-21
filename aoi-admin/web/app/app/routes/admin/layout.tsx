import { AdminThemeLayout, AdminSidebarNav } from "~/theme/generated/templates";

export {
  adminNavGroups,
  adminNavItemMatchesPath,
  findAdminNavGroupId,
  normalizeAdminNavPath,
} from "~/features/admin/navigation";
export type { AdminNavGroup, AdminNavItem } from "~/features/admin/navigation";
export { AdminSidebarNav };

export default AdminThemeLayout;
