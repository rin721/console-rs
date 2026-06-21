package i18n

const DefaultLanguage = "zh-CN"

const (
	HeaderLocale     = "X-Locale"
	LanguageHeader   = "Accept-Language"
	LanguageEnglish  = "en-US"
	LanguageChinese  = "zh-CN"
	LanguageJapanese = "ja-JP"

	FilenameFormatJson = "json"
	FilenameFormatYaml = "yaml"
	FilenameFormatYml  = "yml"
)

var SupportedLanguagesStringSlice = []string{LanguageChinese, LanguageEnglish}
