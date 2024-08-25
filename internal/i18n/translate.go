package i18n

import (
	"embed"
	"log"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/eduardolat/goeasyi18n"
)

//go:embed translations/en/*.yaml
var enTranslationsFS embed.FS

//go:embed translations/zh_CN/*.yaml
var zhCNTranslationsFS embed.FS

var i18n *goeasyi18n.I18n
var lang string

func Init() {
	localLang, err := jibber_jabber.DetectIETF()
	if err != nil {
		lang = "en"
	} else {
		lang = localLang
	}
	log.Println("Detected language:", lang)

	i18n = goeasyi18n.NewI18n()
	// Load the translations
	enTranslations, err := goeasyi18n.LoadFromYamlFS(enTranslationsFS, "translations/en/*.yaml")
	if err != nil {
		panic(err)
	}
	zhCNTranslations, err := goeasyi18n.LoadFromYamlFS(zhCNTranslationsFS, "translations/zh_CN/*.yaml")
	if err != nil {
		panic(err)
	}

	// Register the translations
	i18n.AddLanguage("en", enTranslations)
	i18n.AddLanguage("zh-CN", zhCNTranslations)
}

func T(key string, options ...goeasyi18n.Options) string {
	return i18n.Translate(lang, key, options...)
}
