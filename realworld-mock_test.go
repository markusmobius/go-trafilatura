// This file is part of go-trafilatura, Go package for extracting readable
// content, comments and metadata from a web page. Source available in
// <https://github.com/markusmobius/go-trafilatura>.
// Copyright (C) 2021 Markus Mobius
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or (at your
// option) any later version.
//
// This program is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
// or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License
// for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program. If not, see <https://www.gnu.org/licenses/>.

// Code in this file is ported from <https://github.com/adbar/trafilatura>
// which available under GNU GPL v3 license.

package trafilatura

var rwMockFiles = map[string]string{
	"http://exotic_tags": "exotic_tags.html",
	"https://die-partei.net/luebeck/2012/05/31/das-ministerium-fur-club-kultur-informiert/":                                        "die-partei.net.luebeck.html",
	"https://www.bmjv.de/DE/Verbraucherportal/KonsumImAlltag/TransparenzPreisanpassung/TransparenzPreisanpassung_node.html":        "bmjv.de.konsum.html",
	"http://kulinariaathome.wordpress.com/2012/12/08/mandelplatzchen/":                                                             "kulinariaathome.com.mandelplätzchen.html",
	"https://denkanstoos.wordpress.com/2012/04/11/denkanstoos-april-2012/":                                                         "denkanstoos.com.2012.html",
	"https://www.demokratiewebstatt.at/thema/thema-umwelt-und-klima/woher-kommt-die-dicke-luft":                                    "demokratiewebstatt.at.luft.html",
	"http://www.toralin.de/schmierfett-reparierend-verschlei-y-910.html":                                                           "toralin.de.schmierfett.html",
	"https://www.ebrosia.de/beringer-zinfandel-rose-stone-cellars-lieblich-suess":                                                  "ebrosia.de.zinfandel.html",
	"https://www.landwirt.com/Precision-Farming-Moderne-Sensortechnik-im-Kuhstall,,4229,,Bericht.html":                             "landwirt.com.sensortechnik.html",
	"http://schleifen.ucoz.de/blog/briefe/2010-10-26-18":                                                                           "schleifen.ucoz.de.briefe.html",
	"http://www.rs-ingenieure.de/de/hochbau/leistungen/tragwerksplanung":                                                           "rs-ingenieure.de.tragwerksplanung.html",
	"http://www.simplyscience.ch/teens-liesnach-archiv/articles/wie-entsteht-erdoel.html":                                          "simplyscience.ch.erdoel.html",
	"http://www.shingon-reiki.de/reiki-und-schamanismus/":                                                                          "shingon-reiki.de.schamanismus.html",
	"http://love-hina.ch/news/0409.html":                                                                                           "love-hina.ch.0409.html",
	"http://www.cdu-fraktion-erfurt.de/inhalte/aktuelles/entwicklung-der-waldorfschule-ermoeglicht/index.html":                     "cdu-fraktion-erfurt.de.waldorfschule.html",
	"http://www.wehranlage-horka.de/veranstaltung/887/":                                                                            "wehranlage-horka.de.887.html",
	"https://de.creativecommons.org/index.php/2014/03/20/endlich-wird-es-spannend-die-nc-einschraenkung-nach-deutschem-recht/":     "de.creativecommons.org.endlich.html",
	"https://piratenpartei-mv.de/blog/2013/09/12/grundeinkommen-ist-ein-menschenrecht/":                                            "piratenpartei-mv.de.grundeinkommen.html",
	"https://scilogs.spektrum.de/engelbart-galaxis/die-ablehnung-der-gendersprache/":                                               "spektrum.de.engelbart.html",
	"https://www.rnz.de/nachrichten_artikel,-zz-dpa-Schlaglichter-Frank-Witzel-erhaelt-Deutschen-Buchpreis-2015-_arid,133484.html": "rnz.de.witzel.html",
	"https://www.austria.info/de/aktivitaten/radfahren/radfahren-in-der-weltstadt-salzburg":                                        "austria.info.radfahren.html",
	"https://buchperlen.wordpress.com/2013/10/20/leandra-lou-der-etwas-andere-modeblog-jetzt-auch-zwischen-buchdeckeln/":           "buchperlen.wordpress.com.html",
	"https://www.fairkom.eu/about": "fairkom.eu.about.html",
	"https://futurezone.at/digital-life/uber-konkurrent-lyft-startet-mit-waymo-robotertaxis-in-usa/400487461":                                                        "futurezone.at.lyft.html",
	"http://www.hundeverein-kreisunna.de/unserverein.html":                                                                                                           "hundeverein-kreisunna.de.html",
	"https://viehbacher.com/de/steuerrecht":                                                                                                                          "viehbacher.com.steuerrecht.html",
	"http://www.jovelstefan.de/2011/09/11/gefallt-mir/":                                                                                                              "jovelstefan.de.gefallt.html",
	"https://www.stuttgart.de/item/show/132240/1":                                                                                                                    "stuttgart.de.html",
	"https://www.modepilot.de/2019/05/21/geht-euch-auch-so-oder-auf-reisen-nie-ohne-meinen-duschkopf/":                                                               "modepilot.de.duschkopf.html",
	"https://www.otto.de/twoforfashion/strohtasche/":                                                                                                                 "otto.de.twoforfashion.html",
	"http://iloveponysmag.com/2018/05/24/barbour-coastal/":                                                                                                           "iloveponysmag.com.barbour.html",
	"https://moritz-meyer.net/blog/vreni-frost-instagram-abmahnung/":                                                                                                 "moritz-meyer.net.vreni.html",
	"http://www.womencantalksports.com/top-10-women-talking-sports/":                                                                                                 "womencantalksports.com.top10.html",
	"https://plentylife.blogspot.com/2017/05/strong-beautiful-pamela-reif-rezension.html":                                                                            "plentylife.blogspot.pamela-reif.html",
	"https://www.luxuryhaven.co/2019/05/nam-nghi-phu-quoc-unbound-collection-by-hyatt-officially-opens.html":                                                         "luxuryhaven.co.hyatt.html",
	"https://www.luxuriousmagazine.com/2019/06/royal-salute-polo-rome/":                                                                                              "luxuriousmagazine.com.polo.html",
	"https://www.chip.de/tests/akkuschrauber-werkzeug-co,82197/5":                                                                                                    "chip.de.tests.html",
	"https://www.gruen-digital.de/2015/01/digitalpolitisches-jahrestagung-2015-der-heinrich-boell-stiftung-baden-wuerttemberg/":                                      "gruen-digital.de.jahrestagung.html",
	"https://www.rechtambild.de/2011/10/bgh-marions-kochbuch-de/":                                                                                                    "rechtambild.de.kochbuch.html",
	"http://www.internet-law.de/2011/07/verstost-der-ausschluss-von-pseudonymen-bei-google-gegen-deutsches-recht.html":                                               "internet-law.de.pseudonymen.html",
	"https://www.telemedicus.info/article/2766-Rezension-Haerting-Internetrecht,-5.-Auflage-2014.html":                                                               "telemedicus.info.rezension.html",
	"https://www.cnet.de/88130484/so-koennen-internet-user-nach-dem-eugh-urteil-fuer-den-schutz-sensibler-daten-sorgen":                                              "cnet.de.schutz.html",
	"https://correctiv.org/aktuelles/neue-rechte/2019/05/14/wir-haben-bereits-die-zusage":                                                                            "correctiv.org.zusage.html",
	"https://www.sueddeutsche.de/wirtschaft/bahn-flixbus-flixtrain-deutschlandtakt-fernverkehr-1.4445845":                                                            "sueddeutsche.de.flixtrain.html",
	"https://www.adac.de/rund-ums-fahrzeug/tests/kindersicherheit/kindersitztest-2018/":                                                                              "adac.de.kindersitze.html",
	"https://www.caktusgroup.com/blog/2015/06/08/testing-client-side-applications-django-post-mortem/":                                                               "caktusgroup.com.django.html",
	"https://www.computerbase.de/2007-06/htc-touch-bald-bei-o2-als-xda-nova/":                                                                                        "computerbase.de.htc.html",
	"http://www.chineselyrics4u.com/2011/07/zhi-neng-xiang-nian-ni-jam-hsiao-jing.html":                                                                              "chineselyrics4u.com.zhineng.html",
	"https://www.basicthinking.de/blog/2018/12/05/erfolgreiche-tweets-zutaten/":                                                                                      "basicthinking.de.tweets.html",
	"https://meedia.de/2016/03/08/einstieg-ins-tv-geschaeft-wie-freenet-privatkunden-fuer-antennen-tv-in-hd-qualitaet-gewinnen-will/":                                "meedia.de.freenet.html",
	"https://www.incurvy.de/trends-grosse-groessen/wellness-gesichtsbehandlung-plaisir-daromes/":                                                                     "incurvy.de.wellness.html",
	"https://www.dw.com/en/uncork-the-mystery-of-germanys-fr%C3%BChburgunder/a-16863843":                                                                             "dw.com.uncork.html",
	"https://www.jolie.de/stars/adele-10-kilo-abgenommen-sie-zeigt-sich-schlanker-denn-je-200226.html":                                                               "jolie.de.adele.html",
	"https://www.speicherguide.de/digitalisierung/faktor-mensch/schwierige-gespraeche-so-gehts-24376.aspx":                                                           "speicherguide.de.schwierige.html",
	"https://novalanalove.com/ear-candy/":                                                                                                                            "novalanalove.com.ear-candy.html",
	"http://www.franziska-elea.de/2019/02/10/das-louis-vuitton-missgeschick/":                                                                                        "franziska-elea.de.vuitton.html",
	"https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html":                                                                                  "gofeminin.de.abnehmen.html",
	"https://www.brigitte.de/liebe/persoenlichkeit/ikigai-macht-dich-sofort-gluecklicher--10972896.html":                                                             "brigitte.de.ikigai.html",
	"https://www.changelog.blog/zwischenbilanz-jan-kegelberg-ueber-tops-und-flops-bei-der-transformation-von-sportscheck/":                                           "changelog.blog.zwischenbilanz.html",
	"https://threatpost.com/android-ransomware-spreads-via-sex-simulation-game-links-on-reddit-sms/146774/":                                                          "threatpost.com.android.html",
	"https://www.vice.com/en_uk/article/d3avvm/the-amazon-is-on-fire-and-the-smoke-can-be-seen-from-space":                                                           "vice.com.amazon.html",
	"https://www.heise.de/newsticker/meldung/Lithium-aus-dem-Schredder-4451133.html":                                                                                 "heise.de.lithium.html",
	"https://www.theverge.com/2019/7/3/20680681/ios-13-beta-3-facetime-attention-correction-eye-contact":                                                             "theverge.com.ios13.html",
	"https://crazy-julia.de/beauty-tipps-die-jede-braut-kennen-sollte/":                                                                                              "crazy-julia.de.tipps.html",
	"https://www.politische-bildung-brandenburg.de/themen/land-und-leute/homo-brandenburgensis":                                                                      "brandenburg.de.homo-brandenburgensis.html",
	"https://skateboardmsm.de/news/the-captains-quest-2017-contest-auf-schwimmender-miniramp-am-19-august-in-dormagen.html":                                          "skateboardmsm.de.dormhagen.html",
	"https://knowtechie.com/rocket-pass-4-in-rocket-league-brings-with-it-a-new-rally-inspired-car/":                                                                 "knowtechie.com.rally.html",
	"https://boingboing.net/2013/07/19/hating-millennials-the-preju.html":                                                                                            "boingboing.net.millenials.html",
	"https://en.wikipedia.org/wiki/T-distributed_stochastic_neighbor_embedding":                                                                                      "en.wikipedia.org.tsne.html",
	"https://mixed.de/vrodo-deals-vr-taugliches-notebook-fuer-83215-euro-99-cent-leihfilme-bei-amazon-psvr/":                                                         "mixed.de.vrodo.html",
	"http://www.spreeblick.com/blog/2006/07/29/aus-aus-alles-vorbei-habeck-macht-die-stahnke/":                                                                       "spreeblick.com.habeck.html",
	"https://majkaswelt.com/top-5-fashion-must-haves-2018-werbung/":                                                                                                  "majkaswelt.com.fashion.html",
	"https://erp-news.info/erp-interview-mit-um-digitale-assistenten-und-kuenstliche-intelligenz-ki/":                                                                "erp-news.info.interview.html",
	"https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/":                                                                                                "github.blog.spiceland.html",
	"https://lady50plus.de/2019/06/19/sekre-mystery-bag/":                                                                                                            "lady50plus.de.sekre.html",
	"https://www.sonntag-sachsen.de/emanuel-scobel-wird-thomanerchor-geschaeftsfuehrer":                                                                              "sonntag-sachsen.de.emanuel.html",
	"https://www.psl.eu/actualites/luniversite-psl-quand-les-grandes-ecoles-font-universite":                                                                         "psl.eu.luniversite.html",
	"https://www.chip.de/test/Beef-Maker-von-Aldi-im-Test_154632771.html":                                                                                            "chip.de.beef.html",
	"http://www.sauvonsluniversite.fr/spip.php?article8532":                                                                                                          "sauvonsluniversite.com.spip.html",
	"https://www.spiegel.de/spiegel/print/d-161500790.html":                                                                                                          "spiegel.de.albtraum.html",
	"https://lemire.me/blog/2019/08/02/json-parsing-simdjson-vs-json-for-modern-c/":                                                                                  "lemire.me.json.html",
	"https://www.zeit.de/mobilitaet/2020-01/zugverkehr-christian-lindner-hochgeschwindigkeitsstrecke-eu-kommission":                                                  "zeit.de.zugverkehr.html",
	"https://www.franceculture.fr/emissions/le-journal-des-idees/le-journal-des-idees-emission-du-mardi-14-janvier-2020":                                             "franceculture.fr.idees.html",
	"https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/":                                   "wikimediafoundation.org.turkey.html",
	"https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH":                                         "reuters.com.parasite.html",
	"https://vancouversun.com/technology/microsoft-moves-to-erase-its-carbon-footprint-from-the-atmosphere-in-climate-push/wcm/76e426d9-56de-40ad-9504-18d5101013d2": "vancouversun.com.microsoft.html",
	"https://www.lanouvellerepublique.fr/indre-et-loire/commune/saint-martin-le-beau/family-park-la-derniere-saison-a-saint-martin-le-beau":                          "lanouvellerepublique.fr.martin.html",
}
