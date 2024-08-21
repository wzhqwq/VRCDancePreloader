package i18n

import (
	"embed"
	"log"

	"github.com/cloudfoundry/jibber_jabber"
	"github.com/eduardolat/goeasyi18n"
)

//go:embed translations/en/*.yaml
var enTranslations embed.FS

//go:embed translations/zh_CN/*.yaml
var zhCNTranslations embed.FS

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

	i18n := goeasyi18n.NewI18n()
	// Load the translations
	enTranslations, err := goeasyi18n.LoadFromYamlFS(enTranslations)
	if err != nil {
		panic(err)
	}
	zhCNTranslations, err := goeasyi18n.LoadFromYamlFS(zhCNTranslations)
	if err != nil {
		panic(err)
	}

	// Register the translations
	i18n.AddLanguage("en", enTranslations)
	i18n.AddLanguage("zh_CN", zhCNTranslations)
}

func T(key string, options ...goeasyi18n.Options) string {
	return i18n.Translate(lang, key, options...)
}
