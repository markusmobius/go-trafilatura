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

var metadataMockFiles = map[string]string{
	"http://blog.python.org/2016/12/python-360-is-now-available.html":                                                                           "blog.python.org.html",
	"https://creativecommons.org/about/":                                                                                                        "creativecommons.org.html",
	"https://www.creativecommons.at/faircoin-hackathon":                                                                                         "creativecommons.at.faircoin.html",
	"https://en.blog.wordpress.com/2019/06/19/want-to-see-a-more-diverse-wordpress-contributor-community-so-do-we/":                             "blog.wordpress.com.diverse.html",
	"https://netzpolitik.org/2016/die-cider-connection-abmahnungen-gegen-nutzer-von-creative-commons-bildern/":                                  "netzpolitik.org.abmahnungen.html",
	"https://www.befifty.de/home/2017/7/12/unter-uns-montauk":                                                                                   "befifty.montauk.html",
	"https://www.soundofscience.fr/1927":                                                                                                        "soundofscience.fr.1927.html",
	"https://laviedesidees.fr/L-evaluation-et-les-listes-de.html":                                                                               "laviedesidees.fr.evaluation.html",
	"https://www.theguardian.com/education/2020/jan/20/thousands-of-uk-academics-treated-as-second-class-citizens":                              "theguardian.com.academics.html",
	"https://phys.org/news/2019-10-flint-flake-tool-partially-birch.html":                                                                       "phys.org.tool.html",
	"https://gregoryszorc.com/blog/2020/01/13/mercurial%27s-journey-to-and-reflections-on-python-3/":                                            "gregoryszorc.com.python3.html",
	"https://www.pluralsight.com/tech-blog/managing-python-environments/":                                                                       "pluralsight.com.python.html",
	"https://stackoverflow.blog/2020/01/20/what-is-rust-and-why-is-it-so-popular/":                                                              "stackoverflow.com.rust.html",
	"https://www.dw.com/en/berlin-confronts-germanys-colonial-past-with-new-initiative/a-52060881":                                              "dw.com.colonial.html",
	"https://www.theplanetarypress.com/2020/01/management-of-intact-forestlands-by-indigenous-peoples-key-to-protecting-climate/":               "theplanetarypress.com.forestlands.html",
	"https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/":              "wikimediafoundation.org.turkey.html",
	"https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH":                    "reuters.com.parasite.html",
	"https://www.nationalgeographic.co.uk/environment-and-conservation/2020/01/ravenous-wild-goats-ruled-island-over-century-now-its-being":     "nationalgeographic.co.uk.goats.html",
	"https://www.nature.com/articles/d41586-019-02790-3":                                                                                        "nature.com.telescope.html",
	"https://www.salon.com/2020/01/10/despite-everything-u-s-emissions-dipped-in-2019_partner/":                                                 "salon.com.emissions.html",
	"https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html":                                                             "gofeminin.de.abnehmen.html",
	"https://crazy-julia.de/beauty-tipps-die-jede-braut-kennen-sollte/":                                                                         "crazy-julia.de.tipps.html",
	"https://www.politische-bildung-brandenburg.de/themen/land-und-leute/homo-brandenburgensis":                                                 "brandenburg.de.homo-brandenburgensis.html",
	"https://skateboardmsm.de/news/the-captains-quest-2017-contest-auf-schwimmender-miniramp-am-19-august-in-dormagen.html":                     "skateboardmsm.de.dormhagen.html",
	"https://knowtechie.com/rocket-pass-4-in-rocket-league-brings-with-it-a-new-rally-inspired-car/":                                            "knowtechie.com.rally.html",
	"https://boingboing.net/2013/07/19/hating-millennials-the-preju.html":                                                                       "boingboing.net.millenials.html",
	"http://www.spreeblick.com/blog/2006/07/29/aus-aus-alles-vorbei-habeck-macht-die-stahnke/":                                                  "spreeblick.com.habeck.html",
	"https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/":                                                                           "github.blog.spiceland.html",
	"https://www.sonntag-sachsen.de/emanuel-scobel-wird-thomanerchor-geschaeftsfuehrer":                                                         "sonntag-sachsen.de.emanuel.html",
	"https://www.spiegel.de/spiegel/print/d-161500790.html":                                                                                     "spiegel.de.albtraum.html",
	"https://lemire.me/blog/2019/08/02/json-parsing-simdjson-vs-json-for-modern-c/":                                                             "lemire.me.json.html",
	"https://www.zeit.de/mobilitaet/2020-01/zugverkehr-christian-lindner-hochgeschwindigkeitsstrecke-eu-kommission":                             "zeit.de.zugverkehr.html",
	"https://www.computerbase.de/2007-06/htc-touch-bald-bei-o2-als-xda-nova/":                                                                   "computerbase.de.htc.html",
	"http://www.chineselyrics4u.com/2011/07/zhi-neng-xiang-nian-ni-jam-hsiao-jing.html":                                                         "chineselyrics4u.com.zhineng.html",
	"https://meedia.de/2016/03/08/einstieg-ins-tv-geschaeft-wie-freenet-privatkunden-fuer-antennen-tv-in-hd-qualitaet-gewinnen-will/":           "meedia.de.freenet.html",
	"https://www.telemedicus.info/article/2766-Rezension-Haerting-Internetrecht,-5.-Auflage-2014.html":                                          "telemedicus.info.rezension.html",
	"https://www.cnet.de/88130484/so-koennen-internet-user-nach-dem-eugh-urteil-fuer-den-schutz-sensibler-daten-sorgen":                         "cnet.de.schutz.html",
	"https://www.vice.com/en_uk/article/d3avvm/the-amazon-is-on-fire-and-the-smoke-can-be-seen-from-space":                                      "vice.com.amazon.html",
	"https://www.heise.de/newsticker/meldung/Lithium-aus-dem-Schredder-4451133.html":                                                            "heise.de.lithium.html",
	"https://www.chip.de/test/Beef-Maker-von-Aldi-im-Test_154632771.html":                                                                       "chip.de.beef.html",
	"https://plentylife.blogspot.com/2017/05/strong-beautiful-pamela-reif-rezension.html":                                                       "plentylife.blogspot.pamela-reif.html",
	"https://www.modepilot.de/2019/05/21/geht-euch-auch-so-oder-auf-reisen-nie-ohne-meinen-duschkopf/":                                          "modepilot.de.duschkopf.html",
	"http://iloveponysmag.com/2018/05/24/barbour-coastal/":                                                                                      "iloveponysmag.com.barbour.html",
	"https://moritz-meyer.net/blog/vreni-frost-instagram-abmahnung/":                                                                            "moritz-meyer.net.vreni.html",
	"https://scilogs.spektrum.de/engelbart-galaxis/die-ablehnung-der-gendersprache/":                                                            "spektrum.de.engelbart.html",
	"https://buchperlen.wordpress.com/2013/10/20/leandra-lou-der-etwas-andere-modeblog-jetzt-auch-zwischen-buchdeckeln/":                        "buchperlen.wordpress.com.html",
	"http://kulinariaathome.wordpress.com/2012/12/08/mandelplatzchen/":                                                                          "kulinariaathome.com.mandelplätzchen.html",
	"https://de.creativecommons.org/index.php/2014/03/20/endlich-wird-es-spannend-die-nc-einschraenkung-nach-deutschem-recht/":                  "de.creativecommons.org.endlich.html",
	"https://blog.mondediplo.net/turpitude-et-architecture":                                                                                     "mondediplo.net.turpitude.html",
	"https://www.scmp.com/comment/opinion/article/3046526/taiwanese-president-tsai-ing-wens-political-playbook-should-be":                       "scmp.com.playbook.html",
	"https://www.faz.net/aktuell/wirtschaft/nutzerbasierte-abrechnung-musik-stars-fordern-neues-streaming-modell-16604622.html":                 "faz.net.streaming.html",
	"https://www.ndr.de/nachrichten/info/16-Coronavirus-Update-Wir-brauchen-Abkuerzungen-bei-der-Impfstoffzulassung,podcastcoronavirus140.html": "ndr.de.podcastcoronavirus140.html",
}
