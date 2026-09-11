// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
//
// Copyright (C) 2021 Markus Mobius
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under Apache 2.0 license.

package trafilatura

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
)

func Test_Python220_RealWorld(test *testing.T) {
	runPython220Assertions(test, "test-files/python-2.2.0-realworld.json")
}

func Test_Python220_RealWorldFormatting(test *testing.T) {
	input, err := os.Open("test-files/mock/pcgamer.com.skyrim.html")
	if !assert.NoError(test, err) {
		return
	}
	defer input.Close()
	address, err := url.Parse("http://www.pcgamer.com/2012/08/09/skyrim-part-1/")
	if !assert.NoError(test, err) {
		return
	}
	result, err := Extract(input, Options{OriginalURL: address, EnableFallback: true, IncludeLinks: true})
	if !assert.NoError(test, err) {
		return
	}
	link := dom.QuerySelector(result.ContentNode, `a[href="https://www.pcgamer.com/best-skyrim-mods/"]`)
	if assert.NotNil(test, link) {
		assert.Equal(test, "Skyrim", dom.TextContent(link))
	}
	var emphasized []string
	for _, element := range dom.QuerySelectorAll(result.ContentNode, "em, i") {
		emphasized = append(emphasized, strings.TrimSpace(dom.TextContent(element)))
	}
	assert.Contains(test, emphasized, "Legends")
	assert.Contains(test, emphasized, "houses")
}

func Test_Extract(test *testing.T) {
	// Prepare helper function
	resContains := func(result *ExtractResult, str string) bool {
		return strings.Contains(result.ContentText, str) ||
			strings.Contains(result.CommentsText, str)
	}

	htmlContains := func(result *ExtractResult, str string) bool {
		return strings.Contains(dom.OuterHTML(result.ContentNode), str) ||
			strings.Contains(dom.OuterHTML(result.CommentsNode), str)
	}

	test.Run(rwMockFiles["https://die-partei.net/luebeck/2012/05/31/das-ministerium-fur-club-kultur-informiert/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://die-partei.net/luebeck/2012/05/31/das-ministerium-fur-club-kultur-informiert/")
		assert.False(test, resContains(result, "Impressum"))
		assert.True(test, resContains(result, "Die GEMA dreht völlig am Zeiger!"))

	})
	test.Run(rwMockFiles["https://www.bmjv.de/DE/Verbraucherportal/KonsumImAlltag/TransparenzPreisanpassung/TransparenzPreisanpassung_node.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.bmjv.de/DE/Verbraucherportal/KonsumImAlltag/TransparenzPreisanpassung/TransparenzPreisanpassung_node.html")
		assert.False(test, resContains(result, "Impressum"))
		assert.True(test, resContains(result, "Anbieter von Fernwärme haben innerhalb ihres Leitungsnetzes ein Monopol"))

	})
	test.Run(rwMockFiles["https://denkanstoos.wordpress.com/2012/04/11/denkanstoos-april-2012/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://denkanstoos.wordpress.com/2012/04/11/denkanstoos-april-2012/")
		assert.True(test, resContains(result, "Two or three 10-15 min"))
		assert.True(test, resContains(result, "What type? Etc. (30 mins)"))
		assert.False(test, resContains(result, "Dieser Eintrag wurde veröffentlicht"))
		assert.False(test, resContains(result, "Mit anderen Teillen"))

	})
	test.Run(rwMockFiles["https://www.ebrosia.de/beringer-zinfandel-rose-stone-cellars-lieblich-suess"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.ebrosia.de/beringer-zinfandel-rose-stone-cellars-lieblich-suess")
		assert.True(test, resContains(result, "Das Bukett präsentiert sich"))
		assert.False(test, resContains(result, "Kunden kauften auch"))
		assert.False(test, resContains(result, "Gutschein sichern"))
		assert.True(test, resContains(result, "Besonders gut passt er zu asiatischen Gerichten"))

	})
	test.Run(rwMockFiles["https://www.landwirt.com/Precision-Farming-Moderne-Sensortechnik-im-Kuhstall,,4229,,Bericht.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.landwirt.com/Precision-Farming-Moderne-Sensortechnik-im-Kuhstall,,4229,,Bericht.html")
		assert.True(test, resContains(result, "Überwachung der somatischen Zellen"))
		assert.True(test, resContains(result, "tragbaren Ultraschall-Geräten"))
		assert.True(test, resContains(result, "Kotkonsistenz"))
		assert.False(test, resContains(result, "Anzeigentarife"))
		assert.False(test, resContains(result, "Aktuelle Berichte aus dieser Kategorie"))

	})
	test.Run(rwMockFiles["http://www.rs-ingenieure.de/de/hochbau/leistungen/tragwerksplanung"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.rs-ingenieure.de/de/hochbau/leistungen/tragwerksplanung")
		assert.True(test, resContains(result, "Wir bearbeiten alle Leistungsbilder"))
		assert.False(test, resContains(result, "Brückenbau"))

	})
	test.Run(rwMockFiles["http://www.shingon-reiki.de/reiki-und-schamanismus/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.shingon-reiki.de/reiki-und-schamanismus/")
		assert.False(test, resContains(result, "Catch Evolution"))
		assert.False(test, resContains(result, "und gekennzeichnet mit"))
		assert.True(test, resContains(result, "Heut geht es"))
		assert.True(test, resContains(result, "Ich komme dann zu dir vor Ort."))

	})
	test.Run(rwMockFiles["http://love-hina.ch/news/0409.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://love-hina.ch/news/0409.html")
		assert.True(test, resContains(result, "Kapitel 121 ist"))
		assert.False(test, resContains(result, "Besucher online"))
		assert.False(test, resContains(result, "Kommentare schreiben"))

	})
	test.Run(rwMockFiles["http://www.cdu-fraktion-erfurt.de/inhalte/aktuelles/entwicklung-der-waldorfschule-ermoeglicht/index.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.cdu-fraktion-erfurt.de/inhalte/aktuelles/entwicklung-der-waldorfschule-ermoeglicht/index.html")
		assert.True(test, resContains(result, "der steigenden Nachfrage gerecht zu werden."))
		assert.False(test, resContains(result, "Zurück zur Übersicht"))
		assert.False(test, resContains(result, "Erhöhung für Zoo-Eintritt"))

	})
	test.Run(rwMockFiles["https://de.creativecommons.org/index.php/2014/03/20/endlich-wird-es-spannend-die-nc-einschraenkung-nach-deutschem-recht/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://de.creativecommons.org/index.php/2014/03/20/endlich-wird-es-spannend-die-nc-einschraenkung-nach-deutschem-recht/")
		assert.True(test, resContains(result, "das letzte Wort sein kann."))
		assert.False(test, resContains(result, "Ähnliche Beiträge"))

	})
	test.Run(rwMockFiles["https://piratenpartei-mv.de/blog/2013/09/12/grundeinkommen-ist-ein-menschenrecht/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://piratenpartei-mv.de/blog/2013/09/12/grundeinkommen-ist-ein-menschenrecht/")
		assert.True(test, resContains(result, "Unter diesem Motto findet am 14. September"))
		assert.True(test, resContains(result, "Volksinitiative Schweiz zum Grundeinkommen."))
		assert.False(test, resContains(result, "getaggt mit:"))
		assert.False(test, resContains(result, "Was denkst du?"))

	})
	test.Run(rwMockFiles["https://scilogs.spektrum.de/engelbart-galaxis/die-ablehnung-der-gendersprache/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://scilogs.spektrum.de/engelbart-galaxis/die-ablehnung-der-gendersprache/")
		assert.True(test, resContains(result, "Zweitens wird der Genderstern"))
		assert.True(test, resContains(result, "alldem leider – nichts."))

	})
	test.Run(rwMockFiles["http://www.wehranlage-horka.de/veranstaltung/887/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.wehranlage-horka.de/veranstaltung/887/")
		assert.True(test, resContains(result, "In eine andere Zeit"))
		assert.True(test, resContains(result, "Während Sie über den Markt schlendern"))
		assert.False(test, resContains(result, "Infos zum Verein"))
		assert.False(test, resContains(result, "nach oben"))
		assert.False(test, resContains(result, "Datenschutzerklärung"))

		// Modified by taking only 1st article element...
	})
	test.Run(rwMockFiles["https://www.demokratiewebstatt.at/thema/thema-umwelt-und-klima/woher-kommt-die-dicke-luft"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.demokratiewebstatt.at/thema/thema-umwelt-und-klima/woher-kommt-die-dicke-luft")
		assert.True(test, resContains(result, "Millionen Menschen fahren jeden Tag"))
		assert.False(test, resContains(result, "Clipdealer"))
		assert.False(test, resContains(result, "Teste dein Wissen"))
		assert.False(test, resContains(result, "Thema: Fußball"))
		// assert.True(test, resContains(result, "Eines der großen Probleme,"))
		// assert.True(test, resContains(result, "versteinerte Dinosaurierknochen."))

	})
	test.Run(rwMockFiles["http://www.simplyscience.ch/teens-liesnach-archiv/articles/wie-entsteht-erdoel.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.simplyscience.ch/teens-liesnach-archiv/articles/wie-entsteht-erdoel.html")
		assert.True(test, resContains(result, "Erdöl bildet nach Millionen"))
		assert.True(test, resContains(result, "Warum wird das Erdöl knapp?"))
		assert.False(test, resContains(result, "Die Natur ist aus chemischen Elementen aufgebaut"))

	})
	test.Run(rwMockFiles["https://www.rnz.de/nachrichten_artikel,-zz-dpa-Schlaglichter-Frank-Witzel-erhaelt-Deutschen-Buchpreis-2015-_arid,133484.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.rnz.de/nachrichten_artikel,-zz-dpa-Schlaglichter-Frank-Witzel-erhaelt-Deutschen-Buchpreis-2015-_arid,133484.html")
		assert.True(test, resContains(result, "Für einen Roman"))
		assert.True(test, resContains(result, "Auszeichnung der Branche."))

	})
	test.Run(rwMockFiles["https://buchperlen.wordpress.com/2013/10/20/leandra-lou-der-etwas-andere-modeblog-jetzt-auch-zwischen-buchdeckeln/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://buchperlen.wordpress.com/2013/10/20/leandra-lou-der-etwas-andere-modeblog-jetzt-auch-zwischen-buchdeckeln/")
		assert.True(test, resContains(result, "Dann sollten Sie erst recht"))
		assert.True(test, resContains(result, "als saure Gürkchen entlarvte Ex-Boyfriends."))
		assert.False(test, resContains(result, "Ähnliche Beiträge"))

	})
	test.Run(rwMockFiles["http://www.toralin.de/schmierfett-reparierend-verschlei-y-910.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.toralin.de/schmierfett-reparierend-verschlei-y-910.html")
		assert.True(test, resContains(result, "künftig das XADO-Schutzfett verwenden."))
		assert.True(test, resContains(result, "bis zu 50% Verschleiß."))
		assert.True(test, resContains(result, "Die Lebensdauer von Bauteilen erhöht sich beträchtlich."))
		assert.False(test, resContains(result, "Newsletter"))
		assert.False(test, resContains(result, "Sie könnten auch an folgenden Artikeln interessiert sein"))

	})
	test.Run(rwMockFiles["https://www.fairkom.eu/about"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.fairkom.eu/about")
		assert.True(test, resContains(result, "ein gemeinwohlorientiertes Partnerschaftsnetzwerk"))
		assert.True(test, resContains(result, "Stimmberechtigung bei der Generalversammlung."))
		assert.False(test, resContains(result, "support@fairkom.eu"))

	})
	test.Run(rwMockFiles["https://futurezone.at/digital-life/uber-konkurrent-lyft-startet-mit-waymo-robotertaxis-in-usa/400487461"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://futurezone.at/digital-life/uber-konkurrent-lyft-startet-mit-waymo-robotertaxis-in-usa/400487461")
		assert.True(test, resContains(result, "Einige Kunden des Fahrdienst-Vermittler Lyft"))
		assert.True(test, resContains(result, "zeitweise rund vier Prozent."))
		assert.False(test, resContains(result, "Allgemeine Nutzungsbedingungen"))
		assert.False(test, resContains(result, "Waymo bittet Autohersteller um Geld"))

	})
	test.Run(rwMockFiles["http://www.hundeverein-kreisunna.de/unserverein.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.hundeverein-kreisunna.de/unserverein.html")
		assert.True(test, resContains(result, "Beate und Norbert Olschewski"))
		assert.True(test, resContains(result, "ein Familienmitglied und unser Freund."))
		assert.False(test, resContains(result, "zurück zur Startseite"))

	})
	test.Run(rwMockFiles["https://viehbacher.com/de/steuerrecht"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://viehbacher.com/de/steuerrecht")
		assert.True(test, resContains(result, "und wirtschaftlich orientierte Privatpersonen"))
		assert.True(test, resContains(result, "rund um die Uhr."))
		assert.True(test, resContains(result, "Mensch im Mittelpunkt."))
		assert.False(test, resContains(result, "Was sind Cookies?"))

	})
	test.Run(rwMockFiles["http://www.jovelstefan.de/2011/09/11/gefallt-mir/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.jovelstefan.de/2011/09/11/gefallt-mir/")
		assert.True(test, resContains(result, "Manchmal überrascht einen"))
		assert.True(test, resContains(result, "kein Meisterwerk war!"))
		assert.False(test, resContains(result, "Pingback von"))
		assert.False(test, resContains(result, "Kommentare geschlossen"))

	})
	test.Run(rwMockFiles["https://www.stuttgart.de/item/show/132240/1"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.stuttgart.de/item/show/132240/1")
		assert.True(test, resContains(result, "Das Bohnenviertel entstand"))
		assert.True(test, resContains(result, "sich herrlich entspannen."))
		assert.False(test, resContains(result, "Nützliche Links"))
		assert.False(test, resContains(result, "Mehr zum Thema"))

	})
	test.Run(rwMockFiles["http://kulinariaathome.wordpress.com/2012/12/08/mandelplatzchen/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://kulinariaathome.wordpress.com/2012/12/08/mandelplatzchen/")
		assert.True(test, resContains(result, "zu einem glatten Teig verarbeiten."))
		assert.True(test, resContains(result, "goldbraun sind."))
		assert.True(test, resContains(result, "200 g Zucker"))
		assert.True(test, resContains(result, "Ein Backblech mit Backpapier auslegen."))
		assert.False(test, resContains(result, "Sei der Erste"))
		assert.False(test, resContains(result, "Gefällt mir"))
		assert.False(test, resContains(result, "Trotz sorgfältiger inhaltlicher Kontrolle"))

	})
	test.Run(rwMockFiles["http://schleifen.ucoz.de/blog/briefe/2010-10-26-18"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://schleifen.ucoz.de/blog/briefe/2010-10-26-18")
		assert.True(test, resContains(result, "Es war gesagt,"))
		assert.True(test, resContains(result, "Symbol auf dem Finger haben"))
		assert.False(test, resContains(result, "Aufrufe:"))

	})
	test.Run(rwMockFiles["https://www.austria.info/de/aktivitaten/radfahren/radfahren-in-der-weltstadt-salzburg"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.austria.info/de/aktivitaten/radfahren/radfahren-in-der-weltstadt-salzburg")
		assert.True(test, resContains(result, "Salzburg liebt seine Radfahrer."))
		assert.True(test, resContains(result, "Puls einsaugen zu lassen."))
		assert.False(test, resContains(result, "Das könnte Sie auch interessieren ..."))
		assert.False(test, resContains(result, "So macht Radfahren sonst noch Spaß"))

	})
	test.Run(rwMockFiles["https://www.modepilot.de/2019/05/21/geht-euch-auch-so-oder-auf-reisen-nie-ohne-meinen-duschkopf/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.modepilot.de/2019/05/21/geht-euch-auch-so-oder-auf-reisen-nie-ohne-meinen-duschkopf/")
		assert.True(test, resContains(result, "Allerdings sieht es wie ein Dildo aus,"))
		assert.True(test, resContains(result, "gibt Bescheid, ne?"))
		assert.False(test, resContains(result, "Ähnliche Beiträge"))
		assert.False(test, resContains(result, "Deine E-Mail (bleibt natürlich unter uns)"))

	})
	test.Run(rwMockFiles["https://www.otto.de/twoforfashion/strohtasche/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.otto.de/twoforfashion/strohtasche/")
		assert.True(test, resContains(result, "Ob rund oder kastenförmig, ob dezent oder auffällig"))
		assert.True(test, resContains(result, "XX, Die Redaktion"))
		assert.False(test, resContains(result, " Kommentieren"))
		assert.False(test, resContains(result, "Dienstag, 4. Juni 2019"))

	})
	test.Run(rwMockFiles["http://iloveponysmag.com/2018/05/24/barbour-coastal/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://iloveponysmag.com/2018/05/24/barbour-coastal/")
		assert.True(test, resContains(result, "Eine meiner besten Entscheidungen bisher:"))
		assert.True(test, resContains(result, "Verlassenes Gewächshaus meets versteckter Deich"))
		assert.True(test, resContains(result, "Der Hundestrand in Stein an der Ostsee"))
		assert.False(test, resContains(result, "Tags: Barbour,"))
		assert.True(test, resContains(result, "Bitte (noch) mehr Bilder von Helle"))
		assert.False(test, resContains(result, "Hinterlasse einen Kommentar"))

	})
	test.Run(rwMockFiles["https://moritz-meyer.net/blog/vreni-frost-instagram-abmahnung/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://moritz-meyer.net/blog/vreni-frost-instagram-abmahnung/")
		assert.True(test, resContains(result, "Das ist alles nicht gekennzeichnet, wie soll ich wissen"))
		assert.True(test, resContains(result, "Instagramshops machen es Abmahnanwälten leicht"))
		assert.False(test, resContains(result, "Diese Geschichte teilen"))
		assert.False(test, resContains(result, "Ähnliche Beiträge "))
		assert.True(test, resContains(result, "Ich bin der Ansicht, abwarten und Tee trinken."))
		assert.True(test, resContains(result, "Danke für dein Feedback. Auch zum Look meiner Seite."))
		assert.False(test, resContains(result, "Diese Website verwendet Akismet, um Spam zu reduzieren."))

	})
	test.Run(rwMockFiles["http://www.womencantalksports.com/top-10-women-talking-sports/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.womencantalksports.com/top-10-women-talking-sports/")
		assert.True(test, resContains(result, "Keep Talking Sports!"))
		assert.False(test, resContains(result, "Category: Blog Popular"))
		assert.False(test, resContains(result, "Copyright Women Can Talk Sports."))
		assert.False(test, resContains(result, "Submit your sports question below"))
		assert.True(test, resContains(result, "3.Charlotte Jones Anderson"))

	})
	test.Run(rwMockFiles["https://plentylife.blogspot.com/2017/05/strong-beautiful-pamela-reif-rezension.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://plentylife.blogspot.com/2017/05/strong-beautiful-pamela-reif-rezension.html")
		assert.True(test, resContains(result, "Schönheit kommt für Pamela von Innen und Außen"))
		assert.True(test, resContains(result, "Die Workout Übungen kannte ich bereits"))
		assert.True(test, resContains(result, "Great post, I like your blog"))
		assert.False(test, resContains(result, "Links zu diesem Post"))
		assert.False(test, resContains(result, "mehr über mich ♥"))
		assert.False(test, resContains(result, "Bitte beachte auch die Datenschutzerklärung von Google."))

	})
	test.Run(rwMockFiles["https://www.luxuryhaven.co/2019/05/nam-nghi-phu-quoc-unbound-collection-by-hyatt-officially-opens.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.luxuryhaven.co/2019/05/nam-nghi-phu-quoc-unbound-collection-by-hyatt-officially-opens.html")
		assert.True(test, resContains(result, "Grounded in sustainable architecture and refined Vietnamese craftsmanship,"))
		assert.True(test, resContains(result, "and Carmelo Resort"))
		assert.True(test, resContains(result, "OMG what a beautiful place to stay! "))
		assert.False(test, resContains(result, "Food Advertising by"))
		assert.True(test, resContains(result, "Dining and Drinking"))
		assert.False(test, resContains(result, "A lovely note makes a beautiful day!"))

	})
	test.Run(rwMockFiles["https://www.luxuriousmagazine.com/2019/06/royal-salute-polo-rome/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.luxuriousmagazine.com/2019/06/royal-salute-polo-rome/")
		assert.True(test, resContains(result, "Argentina, the birthplace of polo."))
		assert.True(test, resContains(result, "Simon Wittenberg travels to the Eternal City in Italy"))
		assert.False(test, resContains(result, "Luxury and lifestyle articles"))
		assert.False(test, resContains(result, "Pinterest"))

	})
	test.Run(rwMockFiles["https://www.gruen-digital.de/2015/01/digitalpolitisches-jahrestagung-2015-der-heinrich-boell-stiftung-baden-wuerttemberg/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.gruen-digital.de/2015/01/digitalpolitisches-jahrestagung-2015-der-heinrich-boell-stiftung-baden-wuerttemberg/")
		assert.True(test, resContains(result, "Prof. Dr. Caja Thimm"))
		assert.True(test, resContains(result, "zur Anmeldung."))
		assert.False(test, resContains(result, "Next post"))
		assert.False(test, resContains(result, "Aus den Ländern"))

	})
	test.Run(rwMockFiles["https://www.rechtambild.de/2011/10/bgh-marions-kochbuch-de/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.rechtambild.de/2011/10/bgh-marions-kochbuch-de/")
		assert.True(test, resContains(result, "Leitsätze des Gerichts"))
		assert.False(test, resContains(result, "twittern"))
		assert.False(test, resContains(result, "Ähnliche Beiträge"))
		assert.False(test, resContains(result, "d.toelle[at]rechtambild.de"))

	})
	test.Run(rwMockFiles["http://www.internet-law.de/2011/07/verstost-der-ausschluss-von-pseudonymen-bei-google-gegen-deutsches-recht.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.internet-law.de/2011/07/verstost-der-ausschluss-von-pseudonymen-bei-google-gegen-deutsches-recht.html")
		assert.True(test, resContains(result, "Wann Blogs einer Impressumspflicht unterliegen,"))
		assert.False(test, resContains(result, "Über mich"))
		assert.False(test, resContains(result, "Gesetzes- und Rechtsprechungszitate werden automatisch"))
		assert.True(test, resContains(result, "Mit Verlaub, ich halte das für groben Unsinn."))

	})
	test.Run(rwMockFiles["https://www.telemedicus.info/article/2766-Rezension-Haerting-Internetrecht,-5.-Auflage-2014.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.telemedicus.info/article/2766-Rezension-Haerting-Internetrecht,-5.-Auflage-2014.html")
		assert.True(test, resContains(result, "Aufbau und Inhalt"))
		assert.True(test, resContains(result, "Verlag Dr. Otto Schmidt"))
		assert.False(test, resContains(result, "Handbuch"))
		assert.False(test, resContains(result, "Drucken"))
		assert.False(test, resContains(result, "Ähnliche Artikel"))
		assert.False(test, resContains(result, "Anzeige:"))

	})
	test.Run(rwMockFiles["https://www.cnet.de/88130484/so-koennen-internet-user-nach-dem-eugh-urteil-fuer-den-schutz-sensibler-daten-sorgen"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.cnet.de/88130484/so-koennen-internet-user-nach-dem-eugh-urteil-fuer-den-schutz-sensibler-daten-sorgen")
		assert.True(test, resContains(result, "Auch der Verweis auf ehrverletzende Bewertungen"))
		assert.False(test, resContains(result, "Fanden Sie diesen Artikel nützlich?"))
		assert.False(test, resContains(result, "Kommentar hinzufügen"))
		assert.False(test, resContains(result, "Anja Schmoll-Trautmann"))
		assert.False(test, resContains(result, "Aktuell"))

	})
	test.Run(rwMockFiles["https://correctiv.org/aktuelles/neue-rechte/2019/05/14/wir-haben-bereits-die-zusage"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://correctiv.org/aktuelles/neue-rechte/2019/05/14/wir-haben-bereits-die-zusage")
		assert.False(test, resContains(result, "Alle Artikel zu unseren Recherchen"))
		assert.True(test, resContains(result, "Vorweg: Die beteiligten AfD-Politiker"))
		assert.True(test, resContains(result, "ist heute Abend um 21 Uhr auch im ZDF-Magazin Frontal"))
		assert.False(test, resContains(result, "Wir informieren Sie regelmäßig zum Thema Neue Rechte"))
		assert.False(test, resContains(result, "Kommentar verfassen"))
		assert.False(test, resContains(result, "weiterlesen"))

	})
	test.Run(rwMockFiles["https://www.sueddeutsche.de/wirtschaft/bahn-flixbus-flixtrain-deutschlandtakt-fernverkehr-1.4445845"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.sueddeutsche.de/wirtschaft/bahn-flixbus-flixtrain-deutschlandtakt-fernverkehr-1.4445845")
		assert.False(test, resContains(result, "05:28 Uhr"))
		assert.True(test, resContains(result, "Bahn-Konkurrenten wie Flixbus fürchten durch den geplanten Deutschlandtakt"))
		assert.False(test, resContains(result, "ICE im S-Bahn-Takt"))
		assert.False(test, resContains(result, "Diskussion zu diesem Artikel auf:"))
		assert.False(test, resContains(result, "Berater-Affäre bringt Bahnchef Lutz in Bedrängnis"))
		assert.True(test, resContains(result, "auch der Bus ein klimafreundliches Verkehrsmittel sei"))

	})
	test.Run(rwMockFiles["https://www.adac.de/rund-ums-fahrzeug/tests/kindersicherheit/kindersitztest-2018/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.adac.de/rund-ums-fahrzeug/tests/kindersicherheit/kindersitztest-2018/")
		assert.False(test, resContains(result, "Rund ums Fahrzeug"))
		assert.True(test, resContains(result, "in punkto Sicherheit, Bedienung, Ergonomie"))
		assert.True(test, resContains(result, "Grenzwert der Richtlinie 2014/79/EU"))
		assert.False(test, resContains(result, "Diesel-Umtauschprämien"))
		assert.True(test, resContains(result, "Besonders bei Babyschalen sollte geprüft werden"))

	})
	test.Run(rwMockFiles["https://www.caktusgroup.com/blog/2015/06/08/testing-client-side-applications-django-post-mortem/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.caktusgroup.com/blog/2015/06/08/testing-client-side-applications-django-post-mortem/")
		assert.True(test, resContains(result, "Was I losing my mind?"))
		assert.True(test, resContains(result, "being cached after their first access."))
		assert.True(test, resContains(result, "Finding a Fix"))
		assert.True(test, resContains(result, "from django.conf import settings"))
		assert.False(test, resContains(result, "New Call-to-action"))
		assert.False(test, resContains(result, "Contact us"))
		assert.False(test, resContains(result, "Back to blog"))
		assert.False(test, resContains(result, "You might also like:"))

	})
	test.Run(rwMockFiles["https://www.computerbase.de/2007-06/htc-touch-bald-bei-o2-als-xda-nova/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.computerbase.de/2007-06/htc-touch-bald-bei-o2-als-xda-nova/")
		assert.True(test, resContains(result, "Vor knapp zwei Wochen"))
		assert.True(test, resContains(result, "gibt es in der dazugehörigen Vorstellungs-News."))
		assert.False(test, resContains(result, "Themen:"))
		assert.False(test, resContains(result, "bis Januar 2009 Artikel für ComputerBase verfasst."))
		assert.False(test, resContains(result, "Warum Werbebanner?"))
		assert.False(test, resContains(result, "71 Kommentare"))

	})
	test.Run(rwMockFiles["http://www.chineselyrics4u.com/2011/07/zhi-neng-xiang-nian-ni-jam-hsiao-jing.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.chineselyrics4u.com/2011/07/zhi-neng-xiang-nian-ni-jam-hsiao-jing.html")
		assert.True(test, resContains(result, "就放心去吧"))
		assert.True(test, resContains(result, "Repeat Chorus"))
		assert.False(test, resContains(result, "Older post"))
		assert.False(test, resContains(result, "Thank you for your support!"))

	})
	test.Run(rwMockFiles["https://www.basicthinking.de/blog/2018/12/05/erfolgreiche-tweets-zutaten/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.basicthinking.de/blog/2018/12/05/erfolgreiche-tweets-zutaten/")
		assert.True(test, resContains(result, "Frank Thelen, Investor"))
		assert.True(test, resContains(result, "Female founders must constantly consider"))
		assert.True(test, resContains(result, "Thema des öffentlichen Interesses"))
		assert.False(test, resContains(result, "Nach langjähriger Tätigkeit im Ausland"))
		assert.True(test, resContains(result, "Schaut man ganz genau hin, ist der Habeck-Kommentar"))
		assert.False(test, resContains(result, "Mit Absendung des Formulars willige ich"))
		assert.False(test, resContains(result, "Kommentieren"))

	})
	test.Run(rwMockFiles["https://meedia.de/2016/03/08/einstieg-ins-tv-geschaeft-wie-freenet-privatkunden-fuer-antennen-tv-in-hd-qualitaet-gewinnen-will/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://meedia.de/2016/03/08/einstieg-ins-tv-geschaeft-wie-freenet-privatkunden-fuer-antennen-tv-in-hd-qualitaet-gewinnen-will/")
		assert.True(test, resContains(result, "Welche Werbeeinnahmen erwarten Sie hier langfristig?"))
		assert.True(test, resContains(result, "wir haben keinerlei Pläne, das zu verändern."))
		assert.False(test, resContains(result, "Nachrichtenüberblick abonnieren"))
		assert.False(test, resContains(result, "über alle aktuellen Entwicklungen auf dem Laufenden."))
		assert.False(test, resContains(result, "Schlagworte"))
		assert.False(test, resContains(result, "Teilen"))
		assert.False(test, resContains(result, "Dauerzoff um drohenden UKW-Blackout"))
		assert.True(test, resContains(result, "Mobilcom Debitel has charged me for third party"))

	})
	test.Run(rwMockFiles["https://www.incurvy.de/trends-grosse-groessen/wellness-gesichtsbehandlung-plaisir-daromes/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.incurvy.de/trends-grosse-groessen/wellness-gesichtsbehandlung-plaisir-daromes/")
		assert.True(test, resContains(result, "Zeit für Loslassen und Entspannung."))
		assert.True(test, resContains(result, "Wie sieht dein Alltag aus?"))
		assert.True(test, resContains(result, "Erfrischende, abschwellende Augencreme Phyto Contour"))
		assert.True(test, resContains(result, "Vielen Dank Anja für deine Tipps rund um Beauty"))
		assert.False(test, resContains(result, "Betreiberin von incurvy Plus Size"))
		assert.False(test, resContains(result, "Wir verwenden Cookies"))

	})
	test.Run(rwMockFiles["https://www.dw.com/en/uncork-the-mystery-of-germanys-fr%C3%BChburgunder/a-16863843"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.dw.com/en/uncork-the-mystery-of-germanys-fr%C3%BChburgunder/a-16863843")
		assert.True(test, resContains(result, "No grape variety invites as much intrigue"))
		assert.True(test, resContains(result, "With just 0.9 hectares"))
		assert.False(test, resContains(result, "Related Subjects"))
		assert.False(test, resContains(result, "Audios and videos on the topic"))

	})
	test.Run(rwMockFiles["https://www.jolie.de/stars/adele-10-kilo-abgenommen-sie-zeigt-sich-schlanker-denn-je-200226.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.jolie.de/stars/adele-10-kilo-abgenommen-sie-zeigt-sich-schlanker-denn-je-200226.html")
		assert.True(test, resContains(result, "Adele feierte ausgelassen mit den Spice Girls"))
		assert.True(test, resContains(result, "wie sich Adele weiterentwickelt."))
		assert.False(test, resContains(result, "Sommerzeit ist Urlaubszeit,"))
		assert.False(test, resContains(result, "Lade weitere Inhalte"))

	})
	test.Run(rwMockFiles["https://www.speicherguide.de/digitalisierung/faktor-mensch/schwierige-gespraeche-so-gehts-24376.aspx"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.speicherguide.de/digitalisierung/faktor-mensch/schwierige-gespraeche-so-gehts-24376.aspx")
		assert.True(test, resContains(result, "Konflikte mag keiner."))
		assert.True(test, resContains(result, "Gespräche meistern können."))
		assert.False(test, resContains(result, "Flexible Wege in die"))
		assert.False(test, resContains(result, "Storage für den Mittelstand"))
		assert.False(test, resContains(result, "Weiterführender Link"))

	})
	test.Run(rwMockFiles["https://novalanalove.com/ear-candy/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://novalanalove.com/ear-candy/")
		assert.True(test, resContains(result, "Earcuff: Zoeca"))
		assert.True(test, resContains(result, "mit längeren Ohrringen (:"))
		assert.True(test, resContains(result, "Kreole: Stella Hoops"))
		assert.False(test, resContains(result, "Jetzt heißt es schnell sein:"))
		assert.False(test, resContains(result, "Diese Website speichert Cookies"))
		assert.False(test, resContains(result, "VON Sina Giebel"))

	})
	test.Run(rwMockFiles["http://www.franziska-elea.de/2019/02/10/das-louis-vuitton-missgeschick/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.franziska-elea.de/2019/02/10/das-louis-vuitton-missgeschick/")
		assert.True(test, resContains(result, "Zuerst dachte ich, ich könnte das"))
		assert.True(test, resContains(result, "x Franzi"))
		assert.True(test, resContains(result, "Flauschjacke: Bershka"))
		assert.False(test, resContains(result, "Palm Springs Mini (links)"))
		assert.False(test, resContains(result, "Diese Website verwendet Akismet"))
		assert.False(test, resContains(result, "New York, New York"))
		assert.True(test, htmlContains(result, "Flauschjacke: <strong>Bershka</strong>"))

	})
	test.Run(rwMockFiles["https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html")
		assert.True(test, resContains(result, "Die Psyche spielt eine nicht unerhebliche Rolle"))
		assert.False(test, resContains(result, "Sportskanone oder Sportmuffel"))
		assert.False(test, resContains(result, "PINNEN"))
		assert.True(test, resContains(result, "2. Satt essen bei den Mahlzeiten"))
		assert.False(test, resContains(result, "Bringt die Kilos zum Purzeln!"))
		assert.False(test, resContains(result, "Crash-Diäten ziehen meist den Jojo-Effekt"))

	})
	test.Run(rwMockFiles["https://www.brigitte.de/liebe/persoenlichkeit/ikigai-macht-dich-sofort-gluecklicher--10972896.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.brigitte.de/liebe/persoenlichkeit/ikigai-macht-dich-sofort-gluecklicher--10972896.html")
		assert.True(test, resContains(result, "Glücks-Trend Konkurrenz"))
		assert.True(test, resContains(result, "Praktiziere Dankbarkeit"))
		assert.True(test, resContains(result, "dein Ikigai schon gefunden?"))
		assert.True(test, resContains(result, "14,90 Euro."))
		assert.False(test, resContains(result, "Neu in Liebe"))
		assert.False(test, resContains(result, "Erfahre mehr"))
		assert.False(test, resContains(result, "Erfahrung mit privater Arbeitsvermittlung?"))

	})
	test.Run(rwMockFiles["https://www.changelog.blog/zwischenbilanz-jan-kegelberg-ueber-tops-und-flops-bei-der-transformation-von-sportscheck/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.changelog.blog/zwischenbilanz-jan-kegelberg-ueber-tops-und-flops-bei-der-transformation-von-sportscheck/")
		assert.True(test, resContains(result, "Gibt es weitere Top-Maßnahmen für Multi-Channel?"))
		assert.True(test, resContains(result, "Vielen Dank für das interessante Interview!"))
		assert.False(test, resContains(result, "akzeptiere die Datenschutzbestimmungen"))
		assert.False(test, resContains(result, "Diese Beiträge solltest du nicht verpassen"))
		assert.False(test, resContains(result, "Annette Henkel"))

	})
	test.Run(rwMockFiles["https://threatpost.com/android-ransomware-spreads-via-sex-simulation-game-links-on-reddit-sms/146774/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://threatpost.com/android-ransomware-spreads-via-sex-simulation-game-links-on-reddit-sms/146774/")
		assert.True(test, resContains(result, "These messages include links to the ransomware"))
		assert.True(test, resContains(result, "using novel techniques to exfiltrate data."))
		assert.False(test, resContains(result, "Share this article:"))
		assert.False(test, resContains(result, "Write a comment"))
		assert.False(test, resContains(result, "Notify me when new comments are added."))
		assert.False(test, resContains(result, "uses Akismet to reduce spam."))

	})
	test.Run(rwMockFiles["https://www.vice.com/en_uk/article/d3avvm/the-amazon-is-on-fire-and-the-smoke-can-be-seen-from-space"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.vice.com/en_uk/article/d3avvm/the-amazon-is-on-fire-and-the-smoke-can-be-seen-from-space")
		assert.True(test, resContains(result, "Brazil went dark."))
		assert.True(test, resContains(result, "the highest number of deforestation warnings.”"))
		assert.False(test, resContains(result, "Tagged:"))
		assert.False(test, resContains(result, "to the VICE newsletter."))
		assert.False(test, resContains(result, "Watch this next"))

	})
	test.Run(rwMockFiles["https://www.heise.de/newsticker/meldung/Lithium-aus-dem-Schredder-4451133.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.heise.de/newsticker/meldung/Lithium-aus-dem-Schredder-4451133.html")
		assert.True(test, resContains(result, "Die Ökobilanz von Elektroautos"))
		assert.True(test, resContains(result, "Nur die Folie bleibt zurück"))
		assert.False(test, resContains(result, "Forum zum Thema:"))
		// assert.False(test, resContains(result, "TR 7/2019"))

	})
	test.Run(rwMockFiles["https://www.theverge.com/2019/7/3/20680681/ios-13-beta-3-facetime-attention-correction-eye-contact"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.theverge.com/2019/7/3/20680681/ios-13-beta-3-facetime-attention-correction-eye-contact")
		assert.True(test, resContains(result, "Normally, video calls tend to"))
		assert.True(test, resContains(result, "across both the eyes and nose."))
		assert.True(test, resContains(result, "Added ARKit explanation and tweet."))
		assert.False(test, resContains(result, "Singapore’s public health program"))
		assert.False(test, resContains(result, "Command Line delivers daily updates"))

	})
	test.Run(rwMockFiles["https://crazy-julia.de/beauty-tipps-die-jede-braut-kennen-sollte/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://crazy-julia.de/beauty-tipps-die-jede-braut-kennen-sollte/")
		assert.True(test, resContains(result, "in keinem Braut-Beauty-Programm fehlen darf?"))
		assert.True(test, resContains(result, "nicht nur vor der Hochzeit ein absolutes Muss."))
		assert.True(test, resContains(result, "Gesundes, glänzendes Haar"))
		assert.False(test, resContains(result, "Neue Wandbilder von Posterlounge"))
		assert.False(test, resContains(result, "mit meinen Texten und mit meinen Gedanken."))
		assert.False(test, resContains(result, "Erforderliche Felder sind mit * markiert."))

	})
	test.Run(rwMockFiles["https://www.politische-bildung-brandenburg.de/themen/land-und-leute/homo-brandenburgensis"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.politische-bildung-brandenburg.de/themen/land-und-leute/homo-brandenburgensis")
		assert.True(test, resContains(result, "Stilles Rackern, statt lautem Deklamieren."))
		assert.True(test, resContains(result, "Watt jibt’s n hier zu lachen?"))
		assert.True(test, resContains(result, "Das Brandenbuch. Ein Land in Stichworten."))
		assert.False(test, resContains(result, "Bürgerbeteiligung"))
		assert.False(test, resContains(result, "Anmelden"))
		assert.False(test, resContains(result, "Foto: Timur"))
		assert.False(test, resContains(result, "Schlagworte"))
		assert.False(test, resContains(result, "Zeilenumbrüche und Absätze werden automatisch erzeugt."))

	})
	test.Run(rwMockFiles["https://skateboardmsm.de/news/the-captains-quest-2017-contest-auf-schwimmender-miniramp-am-19-august-in-dormagen.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://skateboardmsm.de/news/the-captains-quest-2017-contest-auf-schwimmender-miniramp-am-19-august-in-dormagen.html")
		assert.True(test, resContains(result, "Wakebeach 257"))
		assert.True(test, resContains(result, "Be there or be square!"))
		assert.True(test, resContains(result, "Hier geht’s zur Facebook Veranstaltung"))
		assert.False(test, resContains(result, "More from News"))
		assert.False(test, resContains(result, "von Redaktion MSM"))
		assert.False(test, resContains(result, "add yours."))

	})
	test.Run(rwMockFiles["https://knowtechie.com/rocket-pass-4-in-rocket-league-brings-with-it-a-new-rally-inspired-car/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://knowtechie.com/rocket-pass-4-in-rocket-league-brings-with-it-a-new-rally-inspired-car/")
		assert.True(test, resContains(result, "Rocket Pass 4 will begin at 10:00 a.m. PDT"))
		assert.True(test, resContains(result, "Holy shit, Mortal Kombat 11"))
		assert.True(test, resContains(result, "Let us know down below in the comments"))
		assert.False(test, resContains(result, "Related Topics"))
		assert.False(test, resContains(result, "You can keep up with me on Twitter"))
		assert.False(test, resContains(result, "Hit the track today with Mario Kart Tour"))

	})
	test.Run(rwMockFiles["https://en.wikipedia.org/wiki/T-distributed_stochastic_neighbor_embedding"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://en.wikipedia.org/wiki/T-distributed_stochastic_neighbor_embedding")
		assert.True(test, resContains(result, "Given a set of high-dimensional objects"))
		assert.True(test, resContains(result, "Herein a heavy-tailed Student t-distribution"))
		assert.False(test, resContains(result, "Categories:"))
		assert.False(test, resContains(result, "Conditional random field"))

	})
	test.Run(rwMockFiles["https://mixed.de/vrodo-deals-vr-taugliches-notebook-fuer-83215-euro-99-cent-leihfilme-bei-amazon-psvr/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://mixed.de/vrodo-deals-vr-taugliches-notebook-fuer-83215-euro-99-cent-leihfilme-bei-amazon-psvr/")
		assert.True(test, resContains(result, "Niedlicher Roboter-Spielkamerad: Anki Cozmo"))
		assert.True(test, resContains(result, "Empfehlungen von Dennis:"))
		assert.False(test, resContains(result, "Unterstütze unsere Arbeit"))
		assert.False(test, resContains(result, "Deepfake-Hollywood"))
		assert.False(test, resContains(result, "Avengers"))
		assert.False(test, resContains(result, "Katzenschreck"))

	})
	test.Run(rwMockFiles["http://www.spreeblick.com/blog/2006/07/29/aus-aus-alles-vorbei-habeck-macht-die-stahnke/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.spreeblick.com/blog/2006/07/29/aus-aus-alles-vorbei-habeck-macht-die-stahnke/")
		assert.True(test, resContains(result, "Hunderttausende von jungen Paaren"))
		assert.True(test, resContains(result, "wie flatterhaft das Mädl ist? :)"))
		assert.False(test, resContains(result, "Malte Welding"))
		assert.False(test, resContains(result, "YouTube und die Alten"))
		assert.False(test, resContains(result, "Autokorrektur"))

	})
	test.Run(rwMockFiles["https://majkaswelt.com/top-5-fashion-must-haves-2018-werbung/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://majkaswelt.com/top-5-fashion-must-haves-2018-werbung/")
		assert.True(test, resContains(result, "Rüschen und Volants."))
		assert.True(test, resContains(result, "ihr jedes Jahr tragen könnt?"))
		assert.False(test, resContains(result, "Das könnte dich auch interessieren"))
		assert.False(test, resContains(result, "Catherine Classic Lac 602"))

	})
	test.Run(rwMockFiles["https://erp-news.info/erp-interview-mit-um-digitale-assistenten-und-kuenstliche-intelligenz-ki/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://erp-news.info/erp-interview-mit-um-digitale-assistenten-und-kuenstliche-intelligenz-ki/")
		assert.True(test, resContains(result, "Einblicke in die Vision zukünftiger Softwaregenerationen"))
		assert.True(test, resContains(result, "Frage 4: Welche Rolle spielt Big Data in Bezug auf Assistenz-Systeme und KI?"))
		assert.True(test, resContains(result, "von The unbelievable Machine Company (*um) zur Verfügung gestellt."))
		assert.False(test, resContains(result, "Matthias Weber ist ERP-Experte mit langjähriger Berufserfahrung."))
		assert.False(test, resContains(result, "Die Top 5 digitalen Trends für den Mittelstand"))
		assert.False(test, resContains(result, ", leading edge,"))
		assert.True(test, htmlContains(result, `<strong>Vision zukünftiger Softwaregenerationen</strong>.`))
		assert.True(test, htmlContains(result, `von <b>The unbelievable Machine Company (*um)</b> zur Verfügung gestellt.`))

	})
	test.Run(rwMockFiles["https://boingboing.net/2013/07/19/hating-millennials-the-preju.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://boingboing.net/2013/07/19/hating-millennials-the-preju.html")
		assert.True(test, resContains(result, "Click through for the whole thing."))
		assert.True(test, resContains(result, "The generation we love to dump on"))
		assert.False(test, resContains(result, "GET THE BOING BOING NEWSLETTER"))

	})
	test.Run(rwMockFiles["https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/")
		assert.True(test, resContains(result, "Erin Spiceland is a Software Engineer for SpaceX."))
		assert.True(test, resContains(result, "make effective plans and goals for the future"))
		assert.True(test, resContains(result, "looking forward to next?"))
		assert.True(test, resContains(result, "Research Consultant at Adelard LLP"))
		assert.False(test, resContains(result, "Related posts"))
		assert.False(test, resContains(result, "Jeremy Epling"))
		assert.False(test, resContains(result, "Missed the main event?"))
		assert.False(test, resContains(result, "Privacy"))

	})
	test.Run(rwMockFiles["https://lady50plus.de/2019/06/19/sekre-mystery-bag/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://lady50plus.de/2019/06/19/sekre-mystery-bag/")
		assert.True(test, resContains(result, "ist eine echte Luxushandtasche"))
		assert.True(test, resContains(result, "Insgesamt 160 weibliche „Designerinnen“"))
		assert.True(test, resContains(result, "Sei herzlich gegrüßt"))
		assert.True(test, resContains(result, "Ein Mann alleine hätte niemals"))
		assert.False(test, resContains(result, "Erforderliche Felder sind mit"))
		assert.False(test, resContains(result, "Benachrichtige mich"))
		assert.False(test, resContains(result, "Reisen ist meine große Leidenschaft"))
		assert.False(test, resContains(result, "Styling Tipps für Oktober"))
		assert.True(test, resContains(result, "in den Bann ziehen!"))

	})
	test.Run(rwMockFiles["https://www.sonntag-sachsen.de/emanuel-scobel-wird-thomanerchor-geschaeftsfuehrer"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.sonntag-sachsen.de/emanuel-scobel-wird-thomanerchor-geschaeftsfuehrer")
		assert.True(test, resContains(result, "Neuer Geschäftsführender Leiter"))
		assert.True(test, resContains(result, "nach Leipzig wechseln."))
		assert.False(test, resContains(result, "Mehr zum Thema"))
		assert.False(test, resContains(result, "Folgen Sie uns auf Facebook und Twitter"))
		assert.False(test, resContains(result, "Aktuelle Ausgabe"))

	})
	test.Run(rwMockFiles["https://www.psl.eu/actualites/luniversite-psl-quand-les-grandes-ecoles-font-universite"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.psl.eu/actualites/luniversite-psl-quand-les-grandes-ecoles-font-universite")
		assert.True(test, resContains(result, "Le décret n°2019-1130 validant"))
		assert.True(test, resContains(result, "restructurant à cet effet »."))
		assert.False(test, resContains(result, " utilise des cookies pour"))
		assert.False(test, resContains(result, "En savoir plus"))

	})
	test.Run(rwMockFiles["https://www.chip.de/test/Beef-Maker-von-Aldi-im-Test_154632771.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.chip.de/test/Beef-Maker-von-Aldi-im-Test_154632771.html")
		assert.True(test, resContains(result, "Starke Hitze nur in der Mitte"))
		assert.True(test, resContains(result, "ca. 35,7×29,4 cm"))
		assert.True(test, resContains(result, "Wir sind im Steak-Himmel!"))
		assert.False(test, resContains(result, "Samsung Galaxy S10 128GB"))
		assert.False(test, resContains(result, "Für Links auf dieser Seite"))

	})
	test.Run(rwMockFiles["http://www.sauvonsluniversite.fr/spip.php?article8532"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.sauvonsluniversite.fr/spip.php?article8532")
		assert.True(test, resContains(result, "L’AG Éducation Île-de-France inter-degrés"))
		assert.True(test, resContains(result, "Grève et mobilisation pour le climat"))
		assert.True(test, resContains(result, "suivi.reformes.blanquer@gmail.com"))
		assert.False(test, resContains(result, "Sauvons l’Université !"))
		assert.False(test, resContains(result, "La semaine de SLU"))

	})
	test.Run(rwMockFiles["https://www.spiegel.de/spiegel/print/d-161500790.html"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.spiegel.de/spiegel/print/d-161500790.html")
		assert.True(test, resContains(result, "Wie konnte es dazu kommen?"))
		assert.True(test, resContains(result, "Die Geschichte beginnt am 26. Oktober"))
		assert.True(test, resContains(result, "Es stützt seine Version."))
		assert.False(test, resContains(result, "und Vorteile sichern!"))
		assert.False(test, resContains(result, "Verschickt"))
		assert.False(test, resContains(result, "Die digitale Welt der Nachrichten."))
		assert.False(test, resContains(result, "Vervielfältigung nur mit Genehmigung"))

	})
	test.Run(rwMockFiles["https://lemire.me/blog/2019/08/02/json-parsing-simdjson-vs-json-for-modern-c/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://lemire.me/blog/2019/08/02/json-parsing-simdjson-vs-json-for-modern-c/")
		assert.True(test, resContains(result, "I use a Skylake processor with GNU GCC 8.3."))
		assert.True(test, resContains(result, "gsoc-2018"))
		assert.True(test, resContains(result, "0.091 GB/s"))
		assert.True(test, resContains(result, "version 0.2 on vcpkg."))
		assert.False(test, resContains(result, "Leave a Reply"))
		assert.False(test, resContains(result, "Science and Technology links"))
		assert.False(test, resContains(result, "Proudly powered by WordPress"))

	})
	test.Run(rwMockFiles["https://www.zeit.de/mobilitaet/2020-01/zugverkehr-christian-lindner-hochgeschwindigkeitsstrecke-eu-kommission"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.zeit.de/mobilitaet/2020-01/zugverkehr-christian-lindner-hochgeschwindigkeitsstrecke-eu-kommission")
		assert.True(test, resContains(result, "36 Stunden."))
		assert.True(test, resContains(result, "Nationale Egoismen"))
		assert.True(test, resContains(result, "Deutschland kaum beschleunigt."))
		assert.False(test, resContains(result, "Durchgehende Tickets fehlen"))
		assert.True(test, resContains(result, "geprägte Fehlentscheidung."))
		assert.True(test, resContains(result, "horrende Preise für miserablen Service bezahlen?"))
		assert.False(test, resContains(result, "Bitte melden Sie sich an, um zu kommentieren."))

	})
	test.Run(rwMockFiles["https://www.franceculture.fr/emissions/le-journal-des-idees/le-journal-des-idees-emission-du-mardi-14-janvier-2020"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.franceculture.fr/emissions/le-journal-des-idees/le-journal-des-idees-emission-du-mardi-14-janvier-2020")
		assert.True(test, resContains(result, "Performativité"))
		assert.True(test, resContains(result, "Les individus productifs communiquent"))
		assert.True(test, resContains(result, "de nos espoirs et de nos désirs."))
		assert.False(test, resContains(result, "A la tribune je monterai"))
		assert.False(test, resContains(result, "À découvrir"))
		assert.False(test, resContains(result, "Le fil culture"))

	})
	test.Run(rwMockFiles["https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/")
		assert.True(test, resContains(result, "as further access is restored."))
		assert.False(test, resContains(result, "Read further in the pursuit of knowledge"))
		assert.False(test, resContains(result, "Here’s what that means."))
		assert.False(test, resContains(result, "Stay up-to-date on our work."))
		assert.False(test, resContains(result, "Photo credits"))

	})
	test.Run(rwMockFiles["https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH")
		assert.False(test, resContains(result, "4 Min Read"))
		assert.False(test, resContains(result, "Factbox: Key winners"))
		assert.True(test, resContains(result, "Despite an unknown cast,"))
		assert.True(test, resContains(result, "Additional reporting by"))
		// assert.False(test, resContains(result, "The Thomson Reuters Trust Principles"))

	})
	test.Run(rwMockFiles["https://vancouversun.com/technology/microsoft-moves-to-erase-its-carbon-footprint-from-the-atmosphere-in-climate-push/wcm/76e426d9-56de-40ad-9504-18d5101013d2"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "https://vancouversun.com/technology/microsoft-moves-to-erase-its-carbon-footprint-from-the-atmosphere-in-climate-push/wcm/76e426d9-56de-40ad-9504-18d5101013d2")
		assert.True(test, resContains(result, "Microsoft Corp said on Thursday"))
		assert.True(test, resContains(result, "Postmedia is committed"))
		assert.False(test, resContains(result, "I consent to receiving"))
		assert.True(test, resContains(result, "It was not immediately clear if"))
		assert.False(test, resContains(result, "turns CO2 into soap"))
		assert.False(test, resContains(result, "Reuters files"))

		// Test extract with links
	})
	test.Run(rwMockFiles["http://www.pcgamer.com/2012/08/09/skyrim-part-1/"], func(test *testing.T) {
		result := extractMockFile(rwMockFiles, "http://www.pcgamer.com/2012/08/09/skyrim-part-1/", true)
		assert.True(test, htmlContains(result, `In <a href="https://www.pcgamer.com/best-skyrim-mods/">Skyrim</a>, a mage`))
		assert.True(test, htmlContains(result, `<em>Legends </em>don&#39;t destroy <em>houses</em>,`))
	})
}
