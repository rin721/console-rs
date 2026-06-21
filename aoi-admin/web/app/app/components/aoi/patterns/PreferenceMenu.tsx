import { Languages, Monitor, Moon, Palette, Sun, type LucideIcon } from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { IconButton } from "~/components/aoi/primitives/IconButton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "~/components/aoi/patterns/DropdownMenu";
import { supportedLocales } from "~/i18n/locales";
import { usePreferencesStore } from "~/stores/preferences-store";
import type { ThemeMode } from "~/features/preferences/theme";

type PreferenceMenuProps = {
  designSystemTo?: string;
};

const themeOptions: Array<{
  icon: LucideIcon;
  labelKey: string;
  value: ThemeMode;
}> = [
  { icon: Monitor, labelKey: "common.preferences.themeModes.system", value: "system" },
  { icon: Sun, labelKey: "common.preferences.themeModes.light", value: "light" },
  { icon: Moon, labelKey: "common.preferences.themeModes.dark", value: "dark" },
];

export function PreferenceMenu({ designSystemTo = "/admin/design-system" }: PreferenceMenuProps) {
  const { i18n, t } = useTranslation();
  const themeMode = usePreferencesStore((state) => state.themeMode);
  const resolvedThemeMode = usePreferencesStore((state) => state.resolvedThemeMode);
  const setThemeMode = usePreferencesStore((state) => state.setThemeMode);
  const TriggerIcon = resolvedThemeMode === "dark" ? Moon : Sun;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <IconButton
          appearance="ghost"
          icon={<TriggerIcon aria-hidden="true" size={18} />}
          label={t("common.preferences.open")}
        />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="aoi-preference-menu" align="end">
        <DropdownMenuLabel>{t("common.preferences.theme")}</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={themeMode}
          onValueChange={(value) => setThemeMode(value as ThemeMode)}
        >
          {themeOptions.map((option) => {
            const OptionIcon = option.icon;

            return (
              <DropdownMenuRadioItem
                icon={<OptionIcon aria-hidden="true" size={16} />}
                key={option.value}
                value={option.value}
              >
                {t(option.labelKey)}
              </DropdownMenuRadioItem>
            );
          })}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>{t("common.labels.language")}</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={i18n.language}
          onValueChange={(locale) => {
            void i18n.changeLanguage(locale);
          }}
        >
          {supportedLocales.map((locale) => (
            <DropdownMenuRadioItem
              icon={<Languages aria-hidden="true" size={16} />}
              key={locale}
              value={locale}
            >
              {t(`common.locales.${locale}`)}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link to={designSystemTo}>
            <Palette aria-hidden="true" size={16} />
            <span>{t("common.preferences.openTokens")}</span>
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
