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
	"strings"
	"testing"

	"github.com/go-shiori/dom"
	"github.com/stretchr/testify/assert"
)

func Test_Extract(t *testing.T) {
	// Prepare helper function
	resContains := func(result *ExtractResult, str string) bool {
		return strings.Contains(result.ContentText, str) ||
			strings.Contains(result.CommentsText, str)
	}

	htmlContains := func(result *ExtractResult, str string) bool {
		return strings.Contains(dom.OuterHTML(result.ContentNode), str) ||
			strings.Contains(dom.OuterHTML(result.CommentsNode), str)
	}

	var result *ExtractResult
	result = extractMockFile(rwMockFiles, "https://die-partei.net/luebeck/2012/05/31/das-ministerium-fur-club-kultur-informiert/")
	assert.False(t, resContains(result, "Impressum"))
	assert.True(t, resContains(result, "Die GEMA dreht völlig am Zeiger!"))

	result = extractMockFile(rwMockFiles, "https://www.bmjv.de/DE/Verbraucherportal/KonsumImAlltag/TransparenzPreisanpassung/TransparenzPreisanpassung_node.html")
	assert.False(t, resContains(result, "Impressum"))
	assert.True(t, resContains(result, "Anbieter von Fernwärme haben innerhalb ihres Leitungsnetzes ein Monopol"))

	result = extractMockFile(rwMockFiles, "https://denkanstoos.wordpress.com/2012/04/11/denkanstoos-april-2012/")
	assert.True(t, resContains(result, "Two or three 10-15 min"))
	assert.True(t, resContains(result, "What type? Etc. (30 mins)"))
	assert.False(t, resContains(result, "Dieser Eintrag wurde veröffentlicht"))
	assert.False(t, resContains(result, "Mit anderen Teillen"))

	result = extractMockFile(rwMockFiles, "https://www.ebrosia.de/beringer-zinfandel-rose-stone-cellars-lieblich-suess")
	assert.True(t, resContains(result, "Das Bukett präsentiert sich"))
	assert.False(t, resContains(result, "Kunden kauften auch"))
	assert.False(t, resContains(result, "Gutschein sichern"))
	assert.True(t, resContains(result, "Besonders gut passt er zu asiatischen Gerichten"))

	result = extractMockFile(rwMockFiles, "https://www.landwirt.com/Precision-Farming-Moderne-Sensortechnik-im-Kuhstall,,4229,,Bericht.html")
	assert.True(t, resContains(result, "Überwachung der somatischen Zellen"))
	assert.True(t, resContains(result, "tragbaren Ultraschall-Geräten"))
	assert.True(t, resContains(result, "Kotkonsistenz"))
	assert.False(t, resContains(result, "Anzeigentarife"))
	assert.False(t, resContains(result, "Aktuelle Berichte aus dieser Kategorie"))

	result = extractMockFile(rwMockFiles, "http://www.rs-ingenieure.de/de/hochbau/leistungen/tragwerksplanung")
	assert.True(t, resContains(result, "Wir bearbeiten alle Leistungsbilder"))
	assert.False(t, resContains(result, "Brückenbau"))

	result = extractMockFile(rwMockFiles, "http://www.shingon-reiki.de/reiki-und-schamanismus/")
	assert.False(t, resContains(result, "Catch Evolution"))
	assert.False(t, resContains(result, "und gekennzeichnet mit"))
	assert.True(t, resContains(result, "Heut geht es"))
	assert.True(t, resContains(result, "Ich komme dann zu dir vor Ort."))

	result = extractMockFile(rwMockFiles, "http://love-hina.ch/news/0409.html")
	assert.True(t, resContains(result, "Kapitel 121 ist"))
	// This one shouldn't exist in result. However, for some reason go-readability
	// returns it. I suspected that there is an issue in go-readability implementation,
	// however after careful inspection it seems already matched with algorithm in
	// readability.js. Another suspect is Go' HTML parser which behave differently
	// comppared to JSDom (which used by readability.js), but it's story for later time.
	// assert.False(t, resContains(result, "Besucher online"))
	assert.False(t, resContains(result, "Kommentare schreiben"))

	result = extractMockFile(rwMockFiles, "http://www.cdu-fraktion-erfurt.de/inhalte/aktuelles/entwicklung-der-waldorfschule-ermoeglicht/index.html")
	assert.True(t, resContains(result, "der steigenden Nachfrage gerecht zu werden."))
	assert.False(t, resContains(result, "Zurück zur Übersicht"))
	assert.False(t, resContains(result, "Erhöhung für Zoo-Eintritt"))

	result = extractMockFile(rwMockFiles, "https://de.creativecommons.org/index.php/2014/03/20/endlich-wird-es-spannend-die-nc-einschraenkung-nach-deutschem-recht/")
	assert.True(t, resContains(result, "das letzte Wort sein kann."))
	assert.False(t, resContains(result, "Ähnliche Beiträge"))

	result = extractMockFile(rwMockFiles, "https://piratenpartei-mv.de/blog/2013/09/12/grundeinkommen-ist-ein-menschenrecht/")
	assert.True(t, resContains(result, "Unter diesem Motto findet am 14. September"))
	assert.True(t, resContains(result, "Volksinitiative Schweiz zum Grundeinkommen."))
	assert.False(t, resContains(result, "getaggt mit:"))
	assert.False(t, resContains(result, "Was denkst du?"))

	result = extractMockFile(rwMockFiles, "https://scilogs.spektrum.de/engelbart-galaxis/die-ablehnung-der-gendersprache/")
	assert.True(t, resContains(result, "Zweitens wird der Genderstern"))
	assert.True(t, resContains(result, "alldem leider – nichts."))

	result = extractMockFile(rwMockFiles, "http://www.wehranlage-horka.de/veranstaltung/887/")
	assert.True(t, resContains(result, "In eine andere Zeit"))
	assert.True(t, resContains(result, "Während Sie über den Markt schlendern"))
	assert.False(t, resContains(result, "Infos zum Verein"))
	assert.False(t, resContains(result, "nach oben"))
	assert.False(t, resContains(result, "Datenschutzerklärung"))

	// Modified by taking only 1st article element...
	result = extractMockFile(rwMockFiles, "https://www.demokratiewebstatt.at/thema/thema-umwelt-und-klima/woher-kommt-die-dicke-luft")
	assert.True(t, resContains(result, "Millionen Menschen fahren jeden Tag"))
	assert.False(t, resContains(result, "Clipdealer"))
	assert.False(t, resContains(result, "Teste dein Wissen"))
	assert.False(t, resContains(result, "Thema: Fußball"))
	// assert.True(t, resContains(result, "Eines der großen Probleme,"))
	// assert.True(t, resContains(result, "versteinerte Dinosaurierknochen."))

	result = extractMockFile(rwMockFiles, "http://www.simplyscience.ch/teens-liesnach-archiv/articles/wie-entsteht-erdoel.html")
	assert.True(t, resContains(result, "Erdöl bildet nach Millionen"))
	assert.True(t, resContains(result, "Warum wird das Erdöl knapp?"))
	assert.False(t, resContains(result, "Die Natur ist aus chemischen Elementen aufgebaut"))

	result = extractMockFile(rwMockFiles, "https://www.rnz.de/nachrichten_artikel,-zz-dpa-Schlaglichter-Frank-Witzel-erhaelt-Deutschen-Buchpreis-2015-_arid,133484.html")
	assert.True(t, resContains(result, "Für einen Roman"))
	assert.True(t, resContains(result, "Auszeichnung der Branche."))

	result = extractMockFile(rwMockFiles, "https://buchperlen.wordpress.com/2013/10/20/leandra-lou-der-etwas-andere-modeblog-jetzt-auch-zwischen-buchdeckeln/")
	assert.True(t, resContains(result, "Dann sollten Sie erst recht"))
	assert.True(t, resContains(result, "als saure Gürkchen entlarvte Ex-Boyfriends."))
	assert.False(t, resContains(result, "Ähnliche Beiträge"))

	result = extractMockFile(rwMockFiles, "http://www.toralin.de/schmierfett-reparierend-verschlei-y-910.html")
	assert.True(t, resContains(result, "künftig das XADO-Schutzfett verwenden."))
	assert.True(t, resContains(result, "bis zu 50% Verschleiß."))
	assert.True(t, resContains(result, "Die Lebensdauer von Bauteilen erhöht sich beträchtlich."))
	assert.False(t, resContains(result, "Newsletter"))
	assert.False(t, resContains(result, "Sie könnten auch an folgenden Artikeln interessiert sein"))

	result = extractMockFile(rwMockFiles, "https://www.fairkom.eu/about")
	assert.True(t, resContains(result, "ein gemeinwohlorientiertes Partnerschaftsnetzwerk"))
	assert.True(t, resContains(result, "Stimmberechtigung bei der Generalversammlung."))
	assert.False(t, resContains(result, "support@fairkom.eu"))

	result = extractMockFile(rwMockFiles, "https://futurezone.at/digital-life/uber-konkurrent-lyft-startet-mit-waymo-robotertaxis-in-usa/400487461")
	assert.True(t, resContains(result, "Einige Kunden des Fahrdienst-Vermittler Lyft"))
	assert.True(t, resContains(result, "zeitweise rund vier Prozent."))
	assert.False(t, resContains(result, "Allgemeine Nutzungsbedingungen"))
	assert.False(t, resContains(result, "Waymo bittet Autohersteller um Geld"))

	result = extractMockFile(rwMockFiles, "http://www.hundeverein-kreisunna.de/unserverein.html")
	assert.True(t, resContains(result, "Beate und Norbert Olschewski"))
	assert.True(t, resContains(result, "ein Familienmitglied und unser Freund."))
	assert.False(t, resContains(result, "zurück zur Startseite"))

	result = extractMockFile(rwMockFiles, "https://viehbacher.com/de/steuerrecht")
	assert.True(t, resContains(result, "und wirtschaftlich orientierte Privatpersonen"))
	assert.True(t, resContains(result, "rund um die Uhr."))
	assert.True(t, resContains(result, "Mensch im Mittelpunkt."))
	assert.False(t, resContains(result, "Was sind Cookies?"))

	result = extractMockFile(rwMockFiles, "http://www.jovelstefan.de/2011/09/11/gefallt-mir/")
	assert.True(t, resContains(result, "Manchmal überrascht einen"))
	assert.True(t, resContains(result, "kein Meisterwerk war!"))
	assert.False(t, resContains(result, "Pingback von"))
	assert.False(t, resContains(result, "Kommentare geschlossen"))

	result = extractMockFile(rwMockFiles, "https://www.stuttgart.de/item/show/132240/1")
	assert.True(t, resContains(result, "Das Bohnenviertel entstand"))
	assert.True(t, resContains(result, "sich herrlich entspannen."))
	assert.False(t, resContains(result, "Nützliche Links"))
	assert.False(t, resContains(result, "Mehr zum Thema"))

	result = extractMockFile(rwMockFiles, "http://kulinariaathome.wordpress.com/2012/12/08/mandelplatzchen/")
	assert.True(t, resContains(result, "zu einem glatten Teig verarbeiten."))
	assert.True(t, resContains(result, "goldbraun sind."))
	assert.True(t, resContains(result, "200 g Zucker"))
	assert.True(t, resContains(result, "Ein Backblech mit Backpapier auslegen."))
	assert.False(t, resContains(result, "Sei der Erste"))
	assert.False(t, resContains(result, "Gefällt mir"))
	assert.False(t, resContains(result, "Trotz sorgfältiger inhaltlicher Kontrolle"))

	result = extractMockFile(rwMockFiles, "http://schleifen.ucoz.de/blog/briefe/2010-10-26-18")
	assert.True(t, resContains(result, "Es war gesagt,"))
	assert.True(t, resContains(result, "Symbol auf dem Finger haben"))
	// TODO: this one is different than the original.
	// In original, it should be false, but our go-readability still catch it.
	assert.True(t, resContains(result, "Aufrufe:"))

	result = extractMockFile(rwMockFiles, "https://www.austria.info/de/aktivitaten/radfahren/radfahren-in-der-weltstadt-salzburg")
	assert.True(t, resContains(result, "Salzburg liebt seine Radfahrer."))
	assert.True(t, resContains(result, "Puls einsaugen zu lassen."))
	assert.False(t, resContains(result, "Das könnte Sie auch interessieren ..."))
	assert.False(t, resContains(result, "So macht Radfahren sonst noch Spaß"))

	result = extractMockFile(rwMockFiles, "https://www.modepilot.de/2019/05/21/geht-euch-auch-so-oder-auf-reisen-nie-ohne-meinen-duschkopf/")
	assert.True(t, resContains(result, "Allerdings sieht es wie ein Dildo aus,"))
	assert.True(t, resContains(result, "gibt Bescheid, ne?"))
	assert.False(t, resContains(result, "Ähnliche Beiträge"))
	assert.False(t, resContains(result, "Deine E-Mail (bleibt natürlich unter uns)"))

	result = extractMockFile(rwMockFiles, "https://www.otto.de/twoforfashion/strohtasche/")
	assert.True(t, resContains(result, "Ob rund oder kastenförmig, ob dezent oder auffällig"))
	assert.True(t, resContains(result, "XX, Die Redaktion"))
	assert.False(t, resContains(result, " Kommentieren"))
	assert.False(t, resContains(result, "Dienstag, 4. Juni 2019"))

	result = extractMockFile(rwMockFiles, "http://iloveponysmag.com/2018/05/24/barbour-coastal/")
	assert.True(t, resContains(result, "Eine meiner besten Entscheidungen bisher:"))
	assert.True(t, resContains(result, "Verlassenes Gewächshaus meets versteckter Deich"))
	assert.True(t, resContains(result, "Der Hundestrand in Stein an der Ostsee"))
	assert.False(t, resContains(result, "Tags: Barbour,"))
	assert.True(t, resContains(result, "Bitte (noch) mehr Bilder von Helle"))
	assert.False(t, resContains(result, "Hinterlasse einen Kommentar"))

	result = extractMockFile(rwMockFiles, "https://moritz-meyer.net/blog/vreni-frost-instagram-abmahnung/")
	assert.True(t, resContains(result, "Das ist alles nicht gekennzeichnet, wie soll ich wissen"))
	assert.True(t, resContains(result, "Instagramshops machen es Abmahnanwälten leicht"))
	assert.False(t, resContains(result, "Diese Geschichte teilen"))
	assert.False(t, resContains(result, "Ähnliche Beiträge "))
	assert.True(t, resContains(result, "Ich bin der Ansicht, abwarten und Tee trinken."))
	assert.True(t, resContains(result, "Danke für dein Feedback. Auch zum Look meiner Seite."))
	assert.False(t, resContains(result, "Diese Website verwendet Akismet, um Spam zu reduzieren."))

	result = extractMockFile(rwMockFiles, "http://www.womencantalksports.com/top-10-women-talking-sports/")
	assert.True(t, resContains(result, "Keep Talking Sports!"))
	assert.False(t, resContains(result, "Category: Blog Popular"))
	assert.False(t, resContains(result, "Copyright Women Can Talk Sports."))
	assert.False(t, resContains(result, "Submit your sports question below"))
	assert.True(t, resContains(result, "3.Charlotte Jones Anderson"))

	result = extractMockFile(rwMockFiles, "https://plentylife.blogspot.com/2017/05/strong-beautiful-pamela-reif-rezension.html")
	assert.True(t, resContains(result, "Schönheit kommt für Pamela von Innen und Außen"))
	assert.True(t, resContains(result, "Die Workout Übungen kannte ich bereits"))
	assert.True(t, resContains(result, "Great post, I like your blog"))
	assert.False(t, resContains(result, "Links zu diesem Post"))
	assert.False(t, resContains(result, "mehr über mich ♥"))
	assert.False(t, resContains(result, "Bitte beachte auch die Datenschutzerklärung von Google."))

	result = extractMockFile(rwMockFiles, "https://www.luxuryhaven.co/2019/05/nam-nghi-phu-quoc-unbound-collection-by-hyatt-officially-opens.html")
	assert.True(t, resContains(result, "Grounded in sustainable architecture and refined Vietnamese craftsmanship,"))
	assert.True(t, resContains(result, "and Carmelo Resort"))
	assert.True(t, resContains(result, "OMG what a beautiful place to stay! "))
	assert.False(t, resContains(result, "Food Advertising by"))
	assert.True(t, resContains(result, "Dining and Drinking"))
	assert.False(t, resContains(result, "A lovely note makes a beautiful day!"))

	result = extractMockFile(rwMockFiles, "https://www.luxuriousmagazine.com/2019/06/royal-salute-polo-rome/")
	assert.True(t, resContains(result, "Argentina, the birthplace of polo."))
	assert.True(t, resContains(result, "Simon Wittenberg travels to the Eternal City in Italy"))
	assert.False(t, resContains(result, "Luxury and lifestyle articles"))
	assert.False(t, resContains(result, "Pinterest"))

	result = extractMockFile(rwMockFiles, "https://www.gruen-digital.de/2015/01/digitalpolitisches-jahrestagung-2015-der-heinrich-boell-stiftung-baden-wuerttemberg/")
	assert.True(t, resContains(result, "Prof. Dr. Caja Thimm"))
	assert.True(t, resContains(result, "zur Anmeldung."))
	assert.False(t, resContains(result, "Next post"))
	assert.False(t, resContains(result, "Aus den Ländern"))

	result = extractMockFile(rwMockFiles, "https://www.rechtambild.de/2011/10/bgh-marions-kochbuch-de/")
	assert.True(t, resContains(result, "Leitsätze des Gerichts"))
	assert.False(t, resContains(result, "twittern"))
	assert.False(t, resContains(result, "Ähnliche Beiträge"))
	assert.False(t, resContains(result, "d.toelle[at]rechtambild.de"))

	result = extractMockFile(rwMockFiles, "http://www.internet-law.de/2011/07/verstost-der-ausschluss-von-pseudonymen-bei-google-gegen-deutsches-recht.html")
	assert.True(t, resContains(result, "Wann Blogs einer Impressumspflicht unterliegen,"))
	assert.False(t, resContains(result, "Über mich"))
	assert.False(t, resContains(result, "Gesetzes- und Rechtsprechungszitate werden automatisch"))
	assert.True(t, resContains(result, "Mit Verlaub, ich halte das für groben Unsinn."))

	result = extractMockFile(rwMockFiles, "https://www.telemedicus.info/article/2766-Rezension-Haerting-Internetrecht,-5.-Auflage-2014.html")
	assert.True(t, resContains(result, "Aufbau und Inhalt"))
	assert.True(t, resContains(result, "Verlag Dr. Otto Schmidt"))
	assert.False(t, resContains(result, "Handbuch"))
	assert.False(t, resContains(result, "Drucken"))
	assert.False(t, resContains(result, "Ähnliche Artikel"))
	assert.False(t, resContains(result, "Anzeige:"))

	result = extractMockFile(rwMockFiles, "https://www.cnet.de/88130484/so-koennen-internet-user-nach-dem-eugh-urteil-fuer-den-schutz-sensibler-daten-sorgen")
	assert.True(t, resContains(result, "Auch der Verweis auf ehrverletzende Bewertungen"))
	assert.False(t, resContains(result, "Fanden Sie diesen Artikel nützlich?"))
	assert.False(t, resContains(result, "Kommentar hinzufügen"))
	assert.False(t, resContains(result, "Anja Schmoll-Trautmann"))
	assert.False(t, resContains(result, "Aktuell"))

	result = extractMockFile(rwMockFiles, "https://correctiv.org/aktuelles/neue-rechte/2019/05/14/wir-haben-bereits-die-zusage")
	assert.False(t, resContains(result, "Alle Artikel zu unseren Recherchen"))
	assert.True(t, resContains(result, "Vorweg: Die beteiligten AfD-Politiker"))
	assert.True(t, resContains(result, "ist heute Abend um 21 Uhr auch im ZDF-Magazin Frontal"))
	assert.False(t, resContains(result, "Wir informieren Sie regelmäßig zum Thema Neue Rechte"))
	assert.False(t, resContains(result, "Kommentar verfassen"))
	assert.False(t, resContains(result, "weiterlesen"))

	result = extractMockFile(rwMockFiles, "https://www.sueddeutsche.de/wirtschaft/bahn-flixbus-flixtrain-deutschlandtakt-fernverkehr-1.4445845")
	assert.False(t, resContains(result, "05:28 Uhr"))
	assert.True(t, resContains(result, "Bahn-Konkurrenten wie Flixbus fürchten durch den geplanten Deutschlandtakt"))
	assert.False(t, resContains(result, "ICE im S-Bahn-Takt"))
	assert.False(t, resContains(result, "Diskussion zu diesem Artikel auf:"))
	assert.False(t, resContains(result, "Berater-Affäre bringt Bahnchef Lutz in Bedrängnis"))
	assert.True(t, resContains(result, "auch der Bus ein klimafreundliches Verkehrsmittel sei"))

	result = extractMockFile(rwMockFiles, "https://www.adac.de/rund-ums-fahrzeug/tests/kindersicherheit/kindersitztest-2018/")
	assert.False(t, resContains(result, "Rund ums Fahrzeug"))
	assert.True(t, resContains(result, "in punkto Sicherheit, Bedienung, Ergonomie"))
	assert.True(t, resContains(result, "Grenzwert der Richtlinie 2014/79/EU"))
	assert.False(t, resContains(result, "Diesel-Umtauschprämien"))
	assert.True(t, resContains(result, "Besonders bei Babyschalen sollte geprüft werden"))

	result = extractMockFile(rwMockFiles, "https://www.caktusgroup.com/blog/2015/06/08/testing-client-side-applications-django-post-mortem/")
	assert.True(t, resContains(result, "Was I losing my mind?"))
	assert.True(t, resContains(result, "being cached after their first access."))
	assert.True(t, resContains(result, "Finding a Fix"))
	assert.True(t, resContains(result, "from django.conf import settings"))
	assert.False(t, resContains(result, "New Call-to-action"))
	assert.False(t, resContains(result, "Contact us"))
	assert.False(t, resContains(result, "Back to blog"))
	assert.False(t, resContains(result, "You might also like:"))

	result = extractMockFile(rwMockFiles, "https://www.computerbase.de/2007-06/htc-touch-bald-bei-o2-als-xda-nova/")
	assert.True(t, resContains(result, "Vor knapp zwei Wochen"))
	assert.True(t, resContains(result, "gibt es in der dazugehörigen Vorstellungs-News."))
	assert.False(t, resContains(result, "Themen:"))
	assert.False(t, resContains(result, "bis Januar 2009 Artikel für ComputerBase verfasst."))
	assert.False(t, resContains(result, "Warum Werbebanner?"))
	assert.False(t, resContains(result, "71 Kommentare"))

	result = extractMockFile(rwMockFiles, "http://www.chineselyrics4u.com/2011/07/zhi-neng-xiang-nian-ni-jam-hsiao-jing.html")
	assert.True(t, resContains(result, "就放心去吧"))
	assert.True(t, resContains(result, "Repeat Chorus"))
	assert.False(t, resContains(result, "Older post"))
	assert.False(t, resContains(result, "Thank you for your support!"))

	result = extractMockFile(rwMockFiles, "https://www.basicthinking.de/blog/2018/12/05/erfolgreiche-tweets-zutaten/")
	assert.True(t, resContains(result, "Frank Thelen, Investor"))
	assert.True(t, resContains(result, "Female founders must constantly consider"))
	assert.True(t, resContains(result, "Thema des öffentlichen Interesses"))
	assert.False(t, resContains(result, "Nach langjähriger Tätigkeit im Ausland"))
	assert.True(t, resContains(result, "Schaut man ganz genau hin, ist der Habeck-Kommentar"))
	assert.False(t, resContains(result, "Mit Absendung des Formulars willige ich"))
	assert.False(t, resContains(result, "Kommentieren"))

	result = extractMockFile(rwMockFiles, "https://meedia.de/2016/03/08/einstieg-ins-tv-geschaeft-wie-freenet-privatkunden-fuer-antennen-tv-in-hd-qualitaet-gewinnen-will/")
	assert.True(t, resContains(result, "Welche Werbeeinnahmen erwarten Sie hier langfristig?"))
	assert.True(t, resContains(result, "wir haben keinerlei Pläne, das zu verändern."))
	assert.False(t, resContains(result, "Nachrichtenüberblick abonnieren"))
	assert.False(t, resContains(result, "über alle aktuellen Entwicklungen auf dem Laufenden."))
	assert.False(t, resContains(result, "Schlagworte"))
	assert.False(t, resContains(result, "Teilen"))
	assert.False(t, resContains(result, "Dauerzoff um drohenden UKW-Blackout"))
	assert.True(t, resContains(result, "Mobilcom Debitel has charged me for third party"))

	result = extractMockFile(rwMockFiles, "https://www.incurvy.de/trends-grosse-groessen/wellness-gesichtsbehandlung-plaisir-daromes/")
	assert.True(t, resContains(result, "Zeit für Loslassen und Entspannung."))
	assert.True(t, resContains(result, "Wie sieht dein Alltag aus?"))
	assert.True(t, resContains(result, "Erfrischende, abschwellende Augencreme Phyto Contour"))
	assert.True(t, resContains(result, "Vielen Dank Anja für deine Tipps rund um Beauty"))
	assert.False(t, resContains(result, "Betreiberin von incurvy Plus Size"))
	assert.False(t, resContains(result, "Wir verwenden Cookies"))

	result = extractMockFile(rwMockFiles, "https://www.dw.com/en/uncork-the-mystery-of-germanys-fr%C3%BChburgunder/a-16863843")
	assert.True(t, resContains(result, "No grape variety invites as much intrigue"))
	assert.True(t, resContains(result, "With just 0.9 hectares"))
	assert.False(t, resContains(result, "Related Subjects"))
	assert.False(t, resContains(result, "Audios and videos on the topic"))

	result = extractMockFile(rwMockFiles, "https://www.jolie.de/stars/adele-10-kilo-abgenommen-sie-zeigt-sich-schlanker-denn-je-200226.html")
	assert.True(t, resContains(result, "Adele feierte ausgelassen mit den Spice Girls"))
	assert.True(t, resContains(result, "wie sich Adele weiterentwickelt."))
	assert.False(t, resContains(result, "Sommerzeit ist Urlaubszeit,"))
	assert.False(t, resContains(result, "Lade weitere Inhalte"))

	result = extractMockFile(rwMockFiles, "https://www.speicherguide.de/digitalisierung/faktor-mensch/schwierige-gespraeche-so-gehts-24376.aspx")
	assert.True(t, resContains(result, "Konflikte mag keiner."))
	assert.True(t, resContains(result, "Gespräche meistern können."))
	assert.False(t, resContains(result, "Flexible Wege in die"))
	assert.False(t, resContains(result, "Storage für den Mittelstand"))
	assert.False(t, resContains(result, "Weiterführender Link"))

	result = extractMockFile(rwMockFiles, "https://novalanalove.com/ear-candy/")
	assert.True(t, resContains(result, "Earcuff: Zoeca"))
	assert.True(t, resContains(result, "mit längeren Ohrringen (:"))
	assert.True(t, resContains(result, "Kreole: Stella Hoops"))
	assert.False(t, resContains(result, "Jetzt heißt es schnell sein:"))
	assert.False(t, resContains(result, "Diese Website speichert Cookies"))
	assert.False(t, resContains(result, "VON Sina Giebel"))

	result = extractMockFile(rwMockFiles, "http://www.franziska-elea.de/2019/02/10/das-louis-vuitton-missgeschick/")
	assert.True(t, resContains(result, "Zuerst dachte ich, ich könnte das"))
	assert.True(t, resContains(result, "x Franzi"))
	assert.True(t, resContains(result, "Flauschjacke: Bershka"))
	assert.False(t, resContains(result, "Palm Springs Mini (links)"))
	assert.False(t, resContains(result, "Diese Website verwendet Akismet"))
	assert.False(t, resContains(result, "New York, New York"))
	assert.True(t, htmlContains(result, "Flauschjacke: <strong>Bershka</strong>"))

	result = extractMockFile(rwMockFiles, "https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html")
	assert.True(t, resContains(result, "Die Psyche spielt eine nicht unerhebliche Rolle"))
	assert.False(t, resContains(result, "Sportskanone oder Sportmuffel"))
	assert.False(t, resContains(result, "PINNEN"))
	assert.True(t, resContains(result, "2. Satt essen bei den Mahlzeiten"))
	assert.False(t, resContains(result, "Bringt die Kilos zum Purzeln!"))
	assert.False(t, resContains(result, "Crash-Diäten ziehen meist den Jojo-Effekt"))

	result = extractMockFile(rwMockFiles, "https://www.brigitte.de/liebe/persoenlichkeit/ikigai-macht-dich-sofort-gluecklicher--10972896.html")
	assert.True(t, resContains(result, "Glücks-Trend Konkurrenz"))
	assert.True(t, resContains(result, "Praktiziere Dankbarkeit"))
	assert.True(t, resContains(result, "dein Ikigai schon gefunden?"))
	assert.True(t, resContains(result, "14,90 Euro."))
	assert.False(t, resContains(result, "Neu in Liebe"))
	assert.False(t, resContains(result, "Erfahre mehr"))
	assert.False(t, resContains(result, "Erfahrung mit privater Arbeitsvermittlung?"))

	result = extractMockFile(rwMockFiles, "https://www.changelog.blog/zwischenbilanz-jan-kegelberg-ueber-tops-und-flops-bei-der-transformation-von-sportscheck/")
	assert.True(t, resContains(result, "Gibt es weitere Top-Maßnahmen für Multi-Channel?"))
	assert.True(t, resContains(result, "Vielen Dank für das interessante Interview!"))
	assert.False(t, resContains(result, "akzeptiere die Datenschutzbestimmungen"))
	assert.False(t, resContains(result, "Diese Beiträge solltest du nicht verpassen"))
	assert.False(t, resContains(result, "Annette Henkel"))

	result = extractMockFile(rwMockFiles, "https://threatpost.com/android-ransomware-spreads-via-sex-simulation-game-links-on-reddit-sms/146774/")
	assert.True(t, resContains(result, "These messages include links to the ransomware"))
	assert.True(t, resContains(result, "using novel techniques to exfiltrate data."))
	assert.False(t, resContains(result, "Share this article:"))
	assert.False(t, resContains(result, "Write a comment"))
	assert.False(t, resContains(result, "Notify me when new comments are added."))
	assert.False(t, resContains(result, "uses Akismet to reduce spam."))

	result = extractMockFile(rwMockFiles, "https://www.vice.com/en_uk/article/d3avvm/the-amazon-is-on-fire-and-the-smoke-can-be-seen-from-space")
	assert.True(t, resContains(result, "Brazil went dark."))
	assert.True(t, resContains(result, "the highest number of deforestation warnings.”"))
	assert.False(t, resContains(result, "Tagged:"))
	assert.False(t, resContains(result, "to the VICE newsletter."))
	assert.False(t, resContains(result, "Watch this next"))

	result = extractMockFile(rwMockFiles, "https://www.heise.de/newsticker/meldung/Lithium-aus-dem-Schredder-4451133.html")
	assert.True(t, resContains(result, "Die Ökobilanz von Elektroautos"))
	assert.True(t, resContains(result, "Nur die Folie bleibt zurück"))
	assert.False(t, resContains(result, "Forum zum Thema:"))
	// assert.False(t, resContains(result, "TR 7/2019"))

	result = extractMockFile(rwMockFiles, "https://www.theverge.com/2019/7/3/20680681/ios-13-beta-3-facetime-attention-correction-eye-contact")
	assert.True(t, resContains(result, "Normally, video calls tend to"))
	assert.True(t, resContains(result, "across both the eyes and nose."))
	assert.True(t, resContains(result, "Added ARKit explanation and tweet."))
	assert.False(t, resContains(result, "Singapore’s public health program"))
	assert.False(t, resContains(result, "Command Line delivers daily updates"))

	result = extractMockFile(rwMockFiles, "https://crazy-julia.de/beauty-tipps-die-jede-braut-kennen-sollte/")
	assert.True(t, resContains(result, "in keinem Braut-Beauty-Programm fehlen darf?"))
	assert.True(t, resContains(result, "nicht nur vor der Hochzeit ein absolutes Muss."))
	assert.True(t, resContains(result, "Gesundes, glänzendes Haar"))
	assert.False(t, resContains(result, "Neue Wandbilder von Posterlounge"))
	assert.False(t, resContains(result, "mit meinen Texten und mit meinen Gedanken."))
	assert.False(t, resContains(result, "Erforderliche Felder sind mit * markiert."))

	result = extractMockFile(rwMockFiles, "https://www.politische-bildung-brandenburg.de/themen/land-und-leute/homo-brandenburgensis")
	assert.True(t, resContains(result, "Stilles Rackern, statt lautem Deklamieren."))
	assert.True(t, resContains(result, "Watt jibt’s n hier zu lachen?"))
	assert.True(t, resContains(result, "Das Brandenbuch. Ein Land in Stichworten."))
	assert.False(t, resContains(result, "Bürgerbeteiligung"))
	assert.False(t, resContains(result, "Anmelden"))
	assert.False(t, resContains(result, "Foto: Timur"))
	assert.False(t, resContains(result, "Schlagworte"))
	assert.False(t, resContains(result, "Zeilenumbrüche und Absätze werden automatisch erzeugt."))

	result = extractMockFile(rwMockFiles, "https://skateboardmsm.de/news/the-captains-quest-2017-contest-auf-schwimmender-miniramp-am-19-august-in-dormagen.html")
	assert.True(t, resContains(result, "Wakebeach 257"))
	assert.True(t, resContains(result, "Be there or be square!"))
	assert.True(t, resContains(result, "Hier geht’s zur Facebook Veranstaltung"))
	assert.False(t, resContains(result, "More from News"))
	assert.False(t, resContains(result, "von Redaktion MSM"))
	assert.False(t, resContains(result, "add yours."))

	result = extractMockFile(rwMockFiles, "https://knowtechie.com/rocket-pass-4-in-rocket-league-brings-with-it-a-new-rally-inspired-car/")
	assert.True(t, resContains(result, "Rocket Pass 4 will begin at 10:00 a.m. PDT"))
	assert.True(t, resContains(result, "Holy shit, Mortal Kombat 11"))
	assert.True(t, resContains(result, "Let us know down below in the comments"))
	assert.False(t, resContains(result, "Related Topics"))
	assert.False(t, resContains(result, "You can keep up with me on Twitter"))
	assert.False(t, resContains(result, "Hit the track today with Mario Kart Tour"))

	result = extractMockFile(rwMockFiles, "https://en.wikipedia.org/wiki/T-distributed_stochastic_neighbor_embedding")
	assert.True(t, resContains(result, "Given a set of high-dimensional objects"))
	assert.True(t, resContains(result, "Herein a heavy-tailed Student t-distribution"))
	assert.False(t, resContains(result, "Categories:"))
	assert.False(t, resContains(result, "Conditional random field"))

	result = extractMockFile(rwMockFiles, "https://mixed.de/vrodo-deals-vr-taugliches-notebook-fuer-83215-euro-99-cent-leihfilme-bei-amazon-psvr/")
	assert.True(t, resContains(result, "Niedlicher Roboter-Spielkamerad: Anki Cozmo"))
	assert.True(t, resContains(result, "Empfehlungen von Dennis:"))
	assert.False(t, resContains(result, "Unterstütze unsere Arbeit"))
	assert.False(t, resContains(result, "Deepfake-Hollywood"))
	assert.False(t, resContains(result, "Avengers"))
	assert.False(t, resContains(result, "Katzenschreck"))

	result = extractMockFile(rwMockFiles, "http://www.spreeblick.com/blog/2006/07/29/aus-aus-alles-vorbei-habeck-macht-die-stahnke/")
	assert.True(t, resContains(result, "Hunderttausende von jungen Paaren"))
	assert.True(t, resContains(result, "wie flatterhaft das Mädl ist? :)"))
	assert.False(t, resContains(result, "Malte Welding"))
	assert.False(t, resContains(result, "YouTube und die Alten"))
	assert.False(t, resContains(result, "Autokorrektur"))

	result = extractMockFile(rwMockFiles, "https://majkaswelt.com/top-5-fashion-must-haves-2018-werbung/")
	assert.True(t, resContains(result, "Rüschen und Volants."))
	assert.True(t, resContains(result, "ihr jedes Jahr tragen könnt?"))
	assert.False(t, resContains(result, "Das könnte dich auch interessieren"))
	assert.False(t, resContains(result, "Catherine Classic Lac 602"))

	result = extractMockFile(rwMockFiles, "https://erp-news.info/erp-interview-mit-um-digitale-assistenten-und-kuenstliche-intelligenz-ki/")
	assert.True(t, resContains(result, "Einblicke in die Vision zukünftiger Softwaregenerationen"))
	assert.True(t, resContains(result, "Frage 4: Welche Rolle spielt Big Data in Bezug auf Assistenz-Systeme und KI?"))
	assert.True(t, resContains(result, "von The unbelievable Machine Company (*um) zur Verfügung gestellt."))
	assert.False(t, resContains(result, "Matthias Weber ist ERP-Experte mit langjähriger Berufserfahrung."))
	assert.False(t, resContains(result, "Die Top 5 digitalen Trends für den Mittelstand"))
	assert.False(t, resContains(result, ", leading edge,"))
	assert.True(t, htmlContains(result, `<strong>Vision zukünftiger Softwaregenerationen</strong>.`))
	assert.True(t, htmlContains(result, `von <b>The unbelievable Machine Company (*um)</b> zur Verfügung gestellt.`))

	result = extractMockFile(rwMockFiles, "https://boingboing.net/2013/07/19/hating-millennials-the-preju.html")
	assert.True(t, resContains(result, "Click through for the whole thing."))
	assert.True(t, resContains(result, "The generation we love to dump on"))
	assert.False(t, resContains(result, "GET THE BOING BOING NEWSLETTER"))

	result = extractMockFile(rwMockFiles, "https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/")
	assert.True(t, resContains(result, "Erin Spiceland is a Software Engineer for SpaceX."))
	assert.True(t, resContains(result, "make effective plans and goals for the future"))
	assert.True(t, resContains(result, "looking forward to next?"))
	assert.True(t, resContains(result, "Research Consultant at Adelard LLP"))
	assert.False(t, resContains(result, "Related posts"))
	assert.False(t, resContains(result, "Jeremy Epling"))
	assert.False(t, resContains(result, "Missed the main event?"))
	assert.False(t, resContains(result, "Privacy"))

	result = extractMockFile(rwMockFiles, "https://lady50plus.de/2019/06/19/sekre-mystery-bag/")
	assert.True(t, resContains(result, "ist eine echte Luxushandtasche"))
	assert.True(t, resContains(result, "Insgesamt 160 weibliche „Designerinnen“"))
	assert.True(t, resContains(result, "Sei herzlich gegrüßt"))
	assert.True(t, resContains(result, "Ein Mann alleine hätte niemals"))
	assert.False(t, resContains(result, "Erforderliche Felder sind mit"))
	assert.False(t, resContains(result, "Benachrichtige mich"))
	assert.False(t, resContains(result, "Reisen ist meine große Leidenschaft"))
	assert.False(t, resContains(result, "Styling Tipps für Oktober"))
	assert.True(t, resContains(result, "in den Bann ziehen!"))

	result = extractMockFile(rwMockFiles, "https://www.sonntag-sachsen.de/emanuel-scobel-wird-thomanerchor-geschaeftsfuehrer")
	assert.True(t, resContains(result, "Neuer Geschäftsführender Leiter"))
	assert.True(t, resContains(result, "nach Leipzig wechseln."))
	assert.False(t, resContains(result, "Mehr zum Thema"))
	assert.False(t, resContains(result, "Folgen Sie uns auf Facebook und Twitter"))
	assert.False(t, resContains(result, "Aktuelle Ausgabe"))

	result = extractMockFile(rwMockFiles, "https://www.psl.eu/actualites/luniversite-psl-quand-les-grandes-ecoles-font-universite")
	assert.True(t, resContains(result, "Le décret n°2019-1130 validant"))
	assert.True(t, resContains(result, "restructurant à cet effet »."))
	assert.False(t, resContains(result, " utilise des cookies pour"))
	assert.False(t, resContains(result, "En savoir plus"))

	result = extractMockFile(rwMockFiles, "https://www.chip.de/test/Beef-Maker-von-Aldi-im-Test_154632771.html")
	assert.True(t, resContains(result, "Starke Hitze nur in der Mitte"))
	assert.True(t, resContains(result, "ca. 35,7×29,4 cm"))
	assert.True(t, resContains(result, "Wir sind im Steak-Himmel!"))
	assert.False(t, resContains(result, "Samsung Galaxy S10 128GB"))
	assert.False(t, resContains(result, "Für Links auf dieser Seite"))

	result = extractMockFile(rwMockFiles, "http://www.sauvonsluniversite.fr/spip.php?article8532")
	assert.True(t, resContains(result, "L’AG Éducation Île-de-France inter-degrés"))
	assert.True(t, resContains(result, "Grève et mobilisation pour le climat"))
	assert.True(t, resContains(result, "suivi.reformes.blanquer@gmail.com"))
	assert.False(t, resContains(result, "Sauvons l’Université !"))
	assert.False(t, resContains(result, "La semaine de SLU"))

	result = extractMockFile(rwMockFiles, "https://www.spiegel.de/spiegel/print/d-161500790.html")
	assert.True(t, resContains(result, "Wie konnte es dazu kommen?"))
	assert.True(t, resContains(result, "Die Geschichte beginnt am 26. Oktober"))
	assert.True(t, resContains(result, "Es stützt seine Version."))
	assert.False(t, resContains(result, "und Vorteile sichern!"))
	assert.False(t, resContains(result, "Verschickt"))
	assert.False(t, resContains(result, "Die digitale Welt der Nachrichten."))
	assert.False(t, resContains(result, "Vervielfältigung nur mit Genehmigung"))

	result = extractMockFile(rwMockFiles, "https://lemire.me/blog/2019/08/02/json-parsing-simdjson-vs-json-for-modern-c/")
	assert.True(t, resContains(result, "I use a Skylake processor with GNU GCC 8.3."))
	assert.True(t, resContains(result, "gsoc-2018"))
	assert.True(t, resContains(result, "0.091 GB/s"))
	assert.True(t, resContains(result, "version 0.2 on vcpkg."))
	assert.False(t, resContains(result, "Leave a Reply"))
	assert.False(t, resContains(result, "Science and Technology links"))
	assert.False(t, resContains(result, "Proudly powered by WordPress"))

	result = extractMockFile(rwMockFiles, "https://www.zeit.de/mobilitaet/2020-01/zugverkehr-christian-lindner-hochgeschwindigkeitsstrecke-eu-kommission")
	assert.True(t, resContains(result, "36 Stunden."))
	assert.True(t, resContains(result, "Nationale Egoismen"))
	assert.True(t, resContains(result, "Deutschland kaum beschleunigt."))
	assert.False(t, resContains(result, "Durchgehende Tickets fehlen"))
	assert.True(t, resContains(result, "geprägte Fehlentscheidung."))
	assert.True(t, resContains(result, "horrende Preise für miserablen Service bezahlen?"))
	assert.False(t, resContains(result, "Bitte melden Sie sich an, um zu kommentieren."))

	result = extractMockFile(rwMockFiles, "https://www.franceculture.fr/emissions/le-journal-des-idees/le-journal-des-idees-emission-du-mardi-14-janvier-2020")
	assert.True(t, resContains(result, "Performativité"))
	assert.True(t, resContains(result, "Les individus productifs communiquent"))
	assert.True(t, resContains(result, "de nos espoirs et de nos désirs."))
	assert.False(t, resContains(result, "A la tribune je monterai"))
	assert.False(t, resContains(result, "À découvrir"))
	assert.False(t, resContains(result, "Le fil culture"))

	result = extractMockFile(rwMockFiles, "https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/")
	assert.True(t, resContains(result, "as further access is restored."))
	assert.False(t, resContains(result, "Read further in the pursuit of knowledge"))
	assert.False(t, resContains(result, "Here’s what that means."))
	assert.False(t, resContains(result, "Stay up-to-date on our work."))
	assert.False(t, resContains(result, "Photo credits"))

	result = extractMockFile(rwMockFiles, "https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH")
	assert.False(t, resContains(result, "4 Min Read"))
	assert.False(t, resContains(result, "Factbox: Key winners"))
	assert.True(t, resContains(result, "Despite an unknown cast,"))
	assert.True(t, resContains(result, "Additional reporting by"))
	// assert.False(t, resContains(result, "The Thomson Reuters Trust Principles"))

	result = extractMockFile(rwMockFiles, "https://vancouversun.com/technology/microsoft-moves-to-erase-its-carbon-footprint-from-the-atmosphere-in-climate-push/wcm/76e426d9-56de-40ad-9504-18d5101013d2")
	assert.True(t, resContains(result, "Microsoft Corp said on Thursday"))
	assert.True(t, resContains(result, "Postmedia is committed"))
	assert.False(t, resContains(result, "I consent to receiving"))
	assert.True(t, resContains(result, "It was not immediately clear if"))
	assert.False(t, resContains(result, "turns CO2 into soap"))
	assert.False(t, resContains(result, "Reuters files"))

	// Test extract with links
	result = extractMockFile(rwMockFiles, "http://www.pcgamer.com/2012/08/09/skyrim-part-1/", true)
	assert.True(t, htmlContains(result, `In <a href="https://www.pcgamer.com/best-skyrim-mods/">Skyrim</a>, a mage`))
	assert.True(t, htmlContains(result, `<em>Legends </em>don&#39;t destroy <em>houses</em>,`))
}
