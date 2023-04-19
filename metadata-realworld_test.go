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

import (
	nurl "net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Metadata_RealPages(t *testing.T) {
	var url string
	var opts Options
	var parsedURL *nurl.URL
	var metadata Metadata

	url = "http://blog.python.org/2016/12/python-360-is-now-available.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Python 3.6.0 is now available!", metadata.Title)
	assert.Equal(t, "Python 3.6.0 is now available! Python 3.6.0 is the newest major release of the Python language, and it contains many new features and opti...", metadata.Description)
	assert.Equal(t, "Ned Deily", metadata.Author)
	assert.Equal(t, url, metadata.URL)
	assert.Equal(t, "blog.python.org", metadata.Sitename)

	url = "https://en.blog.wordpress.com/2019/06/19/want-to-see-a-more-diverse-wordpress-contributor-community-so-do-we/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Want to See a More Diverse WordPress Contributor Community? So Do We.", metadata.Title)
	assert.Equal(t, "More diverse speakers at WordCamps means a more diverse community contributing to WordPress — and that results in better software for everyone.", metadata.Description)
	assert.Equal(t, "The WordPress.com Blog", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://creativecommons.org/about/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "What we do - Creative Commons", metadata.Title)
	assert.Equal(t, "What is Creative Commons? Creative Commons helps you legally share your knowledge and creativity to build a more equitable, accessible, and innovative world. We unlock the full potential of the internet to drive a new era of development, growth and productivity. With a network of staff, board, and affiliates around the world, Creative Commons provides … Read More \"What we do\"", metadata.Description)
	assert.Equal(t, "Creative Commons", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.creativecommons.at/faircoin-hackathon"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "FairCoin hackathon beim Sommercamp", metadata.Title)

	url = "https://netzpolitik.org/2016/die-cider-connection-abmahnungen-gegen-nutzer-von-creative-commons-bildern/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Die Cider Connection: Abmahnungen gegen Nutzer von Creative-Commons-Bildern", metadata.Title)
	assert.Equal(t, "Markus Reuter", metadata.Author)
	assert.Equal(t, "Seit Dezember 2015 verschickt eine Cider Connection zahlreiche Abmahnungen wegen fehlerhafter Creative-Commons-Referenzierungen. Wir haben recherchiert und legen jetzt das Netzwerk der Abmahner offen.", metadata.Description)
	assert.Equal(t, "netzpolitik.org", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.befifty.de/home/2017/7/12/unter-uns-montauk"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Das vielleicht schönste Ende der Welt: Montauk", metadata.Title)
	assert.Equal(t, "Beate Finken", metadata.Author)
	assert.Equal(t, "Ein Strand, ist ein Strand, ist ein Strand Ein Strand, ist ein Strand, ist ein Strand. Von wegen! In Italien ist alles wohl organisiert, Handtuch an Handtuch oder Liegestuhl an Liegestuhl. In der Karibik liegt man unter Palmen im Sand und in Marbella dominieren Beton und eine kerzengerade Promenade", metadata.Description)
	assert.Equal(t, "BeFifty", metadata.Sitename)
	assert.Equal(t, []string{"Travel", "Amerika"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.soundofscience.fr/1927"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Une candidature collective à la présidence du HCERES", metadata.Title)
	assert.Equal(t, "Martin Clavey", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "En réaction à la candidature du conseiller recherche"))
	assert.Equal(t, "The Sound Of Science", metadata.Sitename)
	assert.Equal(t, []string{"Politique scientifique française"}, metadata.Categories)
	assert.Equal(t, []string{"évaluation", "HCERES"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://laviedesidees.fr/L-evaluation-et-les-listes-de.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "L’évaluation et les listes de revues", metadata.Title)
	assert.Equal(t, "Florence Audier", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "L'évaluation, et la place"))
	assert.Equal(t, "La Vie des idées", metadata.Sitename)
	// assert.Equal(t, []string{"Essai", "Économie"}, metadata.Categories)
	assert.Empty(t, metadata.Tags)
	assert.Equal(t, "http://www.laviedesidees.fr/L-evaluation-et-les-listes-de.html", metadata.URL)

	url = "https://www.theguardian.com/education/2020/jan/20/thousands-of-uk-academics-treated-as-second-class-citizens"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Thousands of UK academics 'treated as second-class citizens'", metadata.Title)
	assert.Equal(t, "Richard Adams", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "Report claims higher education institutions"))
	assert.Equal(t, "The Guardian", metadata.Sitename)
	assert.Equal(t, []string{"Education"}, metadata.Categories)
	assert.Contains(t, metadata.Tags, "Higher education")
	// meta name="keywords"
	assert.Equal(t, "http://www.theguardian.com/education/2020/jan/20/thousands-of-uk-academics-treated-as-second-class-citizens", metadata.URL)

	url = "https://phys.org/news/2019-10-flint-flake-tool-partially-birch.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Flint flake tool partially covered by birch tar adds to evidence of Neanderthal complex thinking", metadata.Title)
	assert.Equal(t, "Bob Yirka", metadata.Author)
	assert.Equal(t, "A team of researchers affiliated with several institutions in The Netherlands has found evidence in small a cutting tool of Neanderthals using birch tar. In their paper published in Proceedings of the National Academy of Sciences, the group describes the tool and what it revealed about Neanderthal technology.", metadata.Description)
	assert.Equal(t, "Phys.org", metadata.Sitename)
	// assert.Equal(t, []string{"Archeology", "Fossils"}, metadata.Categories)
	assert.Equal(t, []string{"Science", "Physics News", "Science news", "Technology News",
		"Physics", "Materials", "Nanotech", "Technology"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://gregoryszorc.com/blog/2020/01/13/mercurial%27s-journey-to-and-reflections-on-python-3/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Mercurial's Journey to and Reflections on Python 3", metadata.Title)
	// assert metadata['title'] == "Mercurial's Journey to and Reflections on Python 3"
	// assert.Equal(t, "Gregory Szorc", metadata.Author)
	// assert.Equal(t, "Description of the experience of making Mercurial work with Python 3", metadata.Description)
	// assert.Equal(t, "gregoryszorc", metadata.Sitename)
	// assert metadata['categories'] == ['Mercurial', 'Python']

	url = "https://www.pluralsight.com/tech-blog/managing-python-environments/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Managing Python Environments", metadata.Title)
	assert.Equal(t, "John Walk", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "If you're not careful,"))
	assert.Equal(t, "pluralsight.com", metadata.Sitename) // Pluralsight
	// assert.Equal(t, []string{"practices"}, metadata.Categories)
	// assert.Equal(t, []string{"python", "docker", " getting started"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://stackoverflow.blog/2020/01/20/what-is-rust-and-why-is-it-so-popular/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "What is Rust and why is it so popular? - Stack Overflow Blog", metadata.Title)
	assert.Equal(t, "Jake Goulding", metadata.Author)
	assert.Equal(t, "Stack Overflow Blog", metadata.Sitename)
	assert.Equal(t, []string{"Bulletin"}, metadata.Categories)
	assert.Equal(t, []string{"programming", "rust"}, metadata.Tags)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.dw.com/en/berlin-confronts-germanys-colonial-past-with-new-initiative/a-52060881"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.Contains(metadata.Title, "Berlin confronts Germany's colonial past with new initiative"))
	assert.Equal(t, "Deutsche Welle", metadata.Author) // "actually 'Ben Knight'
	assert.Equal(t, "The German capital has launched a five-year project to mark its part in European colonialism. Streets which still honor leaders who led the Reich's imperial expansion will be renamed — and some locals aren't happy.", metadata.Description)
	assert.Equal(t, "DW.COM", metadata.Sitename) // DW - Deutsche Welle
	assert.Contains(t, metadata.Tags, "Africa")
	assert.Equal(t, url, metadata.URL)

	url = "https://www.theplanetarypress.com/2020/01/management-of-intact-forestlands-by-indigenous-peoples-key-to-protecting-climate/"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.HasPrefix(metadata.Title, "Management of Intact Forestlands by Indigenous Peoples Key to Protecting Climate"))
	assert.Equal(t, "The Planetary Press", metadata.Author) // actually "Julie Mollins"
	assert.Equal(t, "The Planetary Press", metadata.Sitename)
	assert.Contains(t, metadata.Categories, "Climate")
	assert.Equal(t, url, metadata.URL)

	url = "https://wikimediafoundation.org/news/2020/01/15/access-to-wikipedia-restored-in-turkey-after-more-than-two-and-a-half-years/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Access to Wikipedia restored in Turkey after more than two and a half years", metadata.Title)
	assert.Equal(t, "Wikimedia Foundation", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "Today, on Wikipedia’s 19th birthday"))
	assert.Equal(t, "Wikimedia Foundation", metadata.Sitename)
	// assert.Equal(t, []string{"Politics", "Turkey", "Wikipedia"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.reuters.com/article/us-awards-sag/parasite-scores-upset-at-sag-awards-boosting-oscar-chances-idUSKBN1ZI0EH"
	metadata = testGetMetadataFromURL(url)
	assert.True(t, strings.HasSuffix(metadata.Title, "scores historic upset at SAG awards, boosting Oscar chances"))
	assert.Equal(t, "Jill Serjeant", metadata.Author)
	assert.Equal(t, "2020-01-20", metadata.Date.Format("2006-01-02"))
	// assert.Equal(t, "“Parasite,” the Korean language social satire about the wealth gap in South Korea, was the first film in a foreign language to win the top prize of best cast ensemble in the 26 year-history of the SAG awards.", metadata.Description)
	assert.Contains(t, metadata.Tags, "Film")
	assert.Contains(t, metadata.Tags, "South Korea")
	assert.Equal(t, "https://www.reuters.com/article/us-awards-sag-idUSKBN1ZI0EH", metadata.URL)
	// TODO: I'm not sure where the original got "Media" as categories, so here I'll skip it.
	// assert.Contains(t, metadata.Categories, "Media")
	// TODO: It should be "Reuters", but their OpenGraph tag say otherwise.
	assert.Equal(t, "U.S.", metadata.Sitename)

	url = "https://www.nationalgeographic.co.uk/environment-and-conservation/2020/01/ravenous-wild-goats-ruled-island-over-century-now-its-being"
	metadata = testGetMetadataFromURL(url)
	// assert.Equal(t, "National Geographic", metadata.Author)
	assert.Equal(t, "Michael Hingston", metadata.Author)
	assert.Equal(t, "Ravenous wild goats ruled this island for over a century. Now, it's being reborn.", metadata.Title)
	assert.True(t, strings.HasPrefix(metadata.Description, "The rocky island of Redonda, once stripped of its flora and fauna"))
	assert.Equal(t, "National Geographic", metadata.Sitename)
	assert.Equal(t, []string{"Environment and Conservation"}, metadata.Categories) // "Goats", "Environment", "Redonda"
	assert.Equal(t, url, metadata.URL)

	url = "https://www.nature.com/articles/d41586-019-02790-3"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Gigantic Chinese telescope opens to astronomers worldwide", metadata.Title)
	assert.Equal(t, "Elizabeth Gibney", metadata.Author)
	assert.Equal(t, "FAST has superior sensitivity to detect cosmic phenomena, including fast radio bursts and pulsars.", metadata.Description)
	assert.Equal(t, "Nature", metadata.Sitename)
	assert.Contains(t, metadata.Categories, "Exoplanets") // "Astronomy", "Telescope", "China"
	assert.Equal(t, url, metadata.URL)

	url = "https://www.scmp.com/comment/opinion/article/3046526/taiwanese-president-tsai-ing-wens-political-playbook-should-be"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, `Carrie Lam should study Tsai Ing-wen’s playbook`, metadata.Title)
	// author exist in JSON-LD, but it's in botched JSON so it'll be empty
	assert.Equal(t, "Alice Wu", metadata.Author)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.faz.net/aktuell/wirtschaft/nutzerbasierte-abrechnung-musik-stars-fordern-neues-streaming-modell-16604622.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Nutzerbasierte Abrechnung: Musik-Stars fordern neues Streaming-Modell", metadata.Title)
	// author overriden from JSON-LD + double name
	assert.Contains(t, strings.Split(metadata.Author, "; "), "Benjamin Fischer")
	assert.Equal(t, "Frankfurter Allgemeine Zeitung", metadata.Sitename)
	assert.Equal(t, "https://www.faz.net/1.6604622", metadata.URL)

	url = "https://boingboing.net/2013/07/19/hating-millennials-the-preju.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Hating Millennials - the prejudice you're allowed to boast about", metadata.Title)
	assert.Equal(t, "Cory Doctorow", metadata.Author)
	assert.Equal(t, "Boing Boing", metadata.Sitename)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.gofeminin.de/abnehmen/wie-kann-ich-schnell-abnehmen-s1431651.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Wie kann ich schnell abnehmen? Der Schlachtplan zum Wunschgewicht", metadata.Title)
	assert.Equal(t, "Diane Buckstegge", metadata.Author)
	assert.Equal(t, "Gofeminin", metadata.Sitename) // originally "gofeminin"
	assert.Equal(t, url, metadata.URL)

	url = "https://github.blog/2019-03-29-leader-spotlight-erin-spiceland/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Leader spotlight: Erin Spiceland", metadata.Title)
	assert.Equal(t, "Jessica Rudder", metadata.Author)
	assert.True(t, strings.HasPrefix(metadata.Description, "We’re spending Women’s History"))
	assert.Equal(t, "The GitHub Blog", metadata.Sitename)
	assert.Equal(t, []string{"Community"}, metadata.Categories)
	assert.Equal(t, url, metadata.URL)

	url = "https://www.spiegel.de/spiegel/print/d-161500790.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Ein Albtraum", metadata.Title)
	// print(metadata)
	// assert.Equal(t, "Clemens Höges", metadata.Author)

	url = "https://www.salon.com/2020/01/10/despite-everything-u-s-emissions-dipped-in-2019_partner/"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, "Despite everything, U.S. emissions dipped in 2019", metadata.Title)
	// in JSON-LD
	assert.Equal(t, "Nathanael Johnson", metadata.Author)
	assert.Equal(t, "Salon.com", metadata.Sitename)
	// in header
	assert.Contains(t, metadata.Categories, "Science & Health")
	assert.Contains(t, metadata.Tags, "Gas Industry")
	assert.Contains(t, metadata.Tags, "coal emissions")
	assert.Equal(t, url, metadata.URL)

	url = "https://www.ndr.de/nachrichten/info/16-Coronavirus-Update-Wir-brauchen-Abkuerzungen-bei-der-Impfstoffzulassung,podcastcoronavirus140.html"
	parsedURL, _ = nurl.ParseRequestURI(url)
	opts = Options{OriginalURL: parsedURL}
	metadata = testGetMetadataFromURL(url, opts)
	assert.Equal(t, url, metadata.URL)
	assert.Contains(t, metadata.Author, "Korinna Hennig")
	assert.Contains(t, metadata.Tags, "Ältere Menschen")

	url = "https://www.dailymail.co.uk/news/article-9831365/UKs-daily-Covid-cases-fall-SEVENTH-day-Infections-plummet-50-23-511.html"
	metadata = testGetMetadataFromURL(url)
	assert.Equal(t, url, metadata.URL)
	assert.Equal(t, metadata.Author, "Luke Andrews; James Tapsfield")
	assert.Contains(t, metadata.Tags, "news")

	url = "https://www.mercurynews.com/2023/01/16/letters-1119/"
	metadata = testGetMetadataFromURL(url)
}

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
	"https://www.dailymail.co.uk/news/article-9831365/UKs-daily-Covid-cases-fall-SEVENTH-day-Infections-plummet-50-23-511.html":                 "dailymail.co.uk.html",
	"https://www.mercurynews.com/2023/01/16/letters-1119/":                                                                                      "mercurynews.com.2023.01.16.letters-1119.html",
}
