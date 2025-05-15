# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web page data: it can extract metadata, main body text and comments while preserving parts of the text formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2], at the time this port is created Trafilatura is the most efficient open-source article extractor. This is especially impressive considering how robust its code, only around 4,000 lines of Python code that separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any improvements from the original can be implemented easily here. Another advantage, hopefully all web page that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Table of Contents

- [Status](#status)
- [Usage as Go package](#usage-as-go-package)
- [Usage as CLI Application](#usage-as-cli-application)
- [Performance](#performance)
- [Comparison with Other Go Packages](#comparison-with-other-go-packages)
- [Comparison with Original Trafilatura](#comparison-with-original-trafilatura)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Status

This package is stable enough for use and up to date with the original Trafilatura [v2.0.0][last-version] (commit [c6e8340][last-commit]).

There are some difference between this port and the original Trafilatura:

- In the original, metadata from JSON+LD is extracted using regular expressions while in this port it's done using a JSON parser. Thanks to this, our metadata extraction is more accurate than the original, but it will skip metadata that might exist in JSON with invalid format.
- In the original, `python-readability` and `justext` are used as fallback extractors. In this port we use `go-readability` and `go-domdistiller` instead. Therefore, there will be some difference in extraction result between our port and the original.
- In our port we can also specify custom fallback value, so we don't limited to only default extractors.
- The main output of the original Trafilatura is XML, while in our port the main output is HTML. Thanks to this, there are some difference in handling formatting tags (e.g. `<b>`, `<i>`) and paragraphs.

## Usage as Go package

Run following command inside your Go project :

```
go get -u -v github.com/markusmobius/go-trafilatura
```

Next, include it in your application :

```go
import "github.com/markusmobius/go-trafilatura"
```

Now you can use Trafilatura to extract content of a web page. For basic usage you can check the [example](examples/from-url.go).

## Usage as CLI Application

To use CLI, you need to build it from source. Make sure you use `go >= 1.16` then run following commands :

```
go get -u -v github.com/markusmobius/go-trafilatura/cmd/go-trafilatura
```

Once installed, you can use it from your terminal:

```
$ go-trafilatura -h
Extract readable content from a specified source which can be either a HTML file or url.
It also has supports for batch download url either from a file which contains list of url,
RSS feeds and sitemap.

Usage:
  go-trafilatura [flags] [source]
  go-trafilatura [command]

Available Commands:
  batch       Download and extract pages from list of urls that specified in the file
  feed        Download and extract pages from a feed
  help        Help about any command
  sitemap     Download and extract pages from a sitemap

Flags:
      --deduplicate         filter out duplicate segments and sections
  -f, --format string       output format for the extract result, either 'html' (default), 'txt' or 'json'
      --has-metadata        only output documents with title, URL and date
  -h, --help                help for go-trafilatura
      --images              include images in extraction result (experimental)
  -l, --language string     target language (ISO 639-1 codes)
      --links               keep links in extraction result (experimental)
      --no-comments         exclude comments  extraction result
      --no-fallback         disable fallback extraction using readability and dom-distiller
      --no-tables           include tables in extraction result
      --skip-tls            skip X.509 (TLS) certificate verification
  -t, --timeout int         timeout for downloading web page in seconds (default 30)
  -u, --user-agent string   set custom user agent (default "Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")
  -v, --verbose             enable log message

Use "go-trafilatura [command] --help" for more information about a command
```

Here are some example of common usage

- Fetch readable content from a specified URL

  ```
  go-trafilatura http://www.domain.com/some/path
  ```

  The output will be printed in stdout.

- Use `batch` command to fetch readable content from file which contains list of urls. So, say we have file
  named `input.txt` with following content:

  ```
  http://www.domain1.com/some/path
  http://www.domain2.com/some/path
  http://www.domain3.com/some/path
  ```

  We want to fetch them and save the result in directory `extract`. To do so, we can run:

  ```
  go-trafilatura batch -o extract input.txt
  ```

- Use `sitemap` to crawl sitemap then fetch all web pages that listed under the sitemap. We can explicitly
  specify the sitemap:

  ```
  go-trafilatura sitemap -o extract http://www.domain.com/sitemap.xml
  ```

  Or you can just put the domain and let Trafitula to look for the sitemap:

  ```
  go-trafilatura sitemap -o extract http://www.domain.com
  ```

- Use `feed` to crawl RSS or Atom feed, then fetch all web pages that listed under it. We can explicitly
  specify the feed url:

  ```
  go-trafilatura feed -o extract http://www.domain.com/feed-rss.php
  ```

  Or you can just put the domain and let Trafitula to look for the feed url:

  ```
  go-trafilatura feed -o extract http://www.domain.com
  ```

## Performance

This package and its dependencies heavily use regular expression for various purposes. Unfortunately, as commonly known, Go's regular expression is pretty [slow][go-regex-slow]. This is because:

- The regex engine in other language usually implemented in C, while in Go it's implemented from scratch in Go language. As expected, C implementation is still faster than Go's.
- Since Go is usually used for web service, its regex is designed to finish in time linear to the length of the input, which useful for protecting server from ReDoS attack. However, this comes with performance cost.

To solve this issue, we compile several important regexes into Go code using [re2go]. Thanks to this we are able to get a great speed without using cgo or external regex packages.

## Comparison with Other Go Packages

As far as we know, currently there are three content extractors built for Go:

- [Go-DomDistiller][dom-distiller]
- [Go-Readability][readability]
- Go-Trafilatura

Since every extractors use its own algorithms, their results are a bit different. In general they give satisfactory results, however we found out that there are some cases where DOM Distiller is better and vice versa. Here is the short summary of pros and cons for each extractor:

Dom Distiller:

- Very fast.
- Good at extracting images from article.
- Able to find next page in sites that separated its article to several partial pages.
- Since the original library was embedded in Chromium browser, its tests are pretty thorough.
- CON: has a huge codebase, mostly because it mimics the original Java code.
- CON: the original library is not maintained anymore and has been archived.

Readability:

- Fast, although not as fast as Dom Distiller.
- Better than DOM Distiller at extracting wiki and documentation pages.
- The original library in Readability.js is still actively used and maintained by Firefox.
- The codebase is pretty small.
- CON: the unit tests are not as thorough as the other extractors.

Trafilatura:

- Has the best accuracy compared to other extractors.
- Better at extracting web page's metadata, including its language and publish date.
- Its unit tests are thorough and focused on removing noise while making sure the real contents are still captured.
- Designed to be used in academic domain e.g. natural language processing.
- Actively maintained with new release almost every month.
- CON: slower than the other extractors, mostly because it also looks for language and publish date.
- CON: doesn't really good at extracting images.

The benchmark that compares these extractors is available in [this repository][benchmark]. It uses each extractor to process 983 web pages in single thread. Here is its benchmark result when tested in my PC (Intel i7-8550U @ 4.000GHz, RAM 16 GB):

Here we compare the extraction result between `go-trafilatura`, `go-readability` and `go-domdistiller`. To reproduce this test, clone this repository then run:

```
go run scripts/comparison/*.go content
```

For the test, we use 960 documents taken from various sources (2025-05-01). Here is the result when tested in my PC (AMD Ryzen 5 7535HS @ 4.6GHz, RAM 16 GB):

|            Package             | Precision | Recall | Accuracy | F-Score | Time (s) |
| :----------------------------: | :-------: | :----: | :------: | :-----: | :------: |
|        `go-readability`        |   0.871   | 0.891  |  0.880   |  0.881  |   2.87   |
|       `go-domdistiller`        |   0.873   | 0.872  |  0.873   |  0.872  |   2.66   |
|        `go-trafilatura`        |   0.912   | 0.897  |  0.906   |  0.904  |   4.25   |
| `go-trafilatura` with fallback |   0.909   | 0.921  |  0.914   |  0.915  |   8.39   |

## Comparison with Original Trafilatura

Here is the result when compared with the original Trafilatura v1.12.2:

|                 Package                 | Precision | Recall | Accuracy | F-Score | Time (s) |
| :-------------------------------------: | :-------: | :----: | :------: | :-----: | :------: |
|              `trafilatura`              |   0.918   | 0.898  |  0.909   |  0.908  |  10.38   |
|        `trafilatura` + fallback         |   0.919   | 0.915  |  0.917   |  0.917  |  14.53   |
|  `trafilatura` + fallback + precision   |   0.932   | 0.889  |  0.912   |  0.910  |  19.34   |
|    `trafilatura` + fallback + recall    |   0.907   | 0.919  |  0.913   |  0.913  |  11.63   |
|            `go-trafilatura`             |   0.912   | 0.897  |  0.906   |  0.904  |   4.25   |
|       `go-trafilatura` + fallback       |   0.909   | 0.921  |  0.914   |  0.915  |   8.39   |
| `go-trafilatura` + fallback + precision |   0.921   | 0.900  |  0.912   |  0.910  |   7.68   |
|  `go-trafilatura` + fallback + recall   |   0.893   | 0.927  |  0.908   |  0.910  |   6.43   |

From the table above we can see that our port has almost similar performance as the original Trafilatura. This is thanks to the fact that most of code is ported line by line from Python to Go (excluding some difference that mentioned above). The small performance difference between our port and the original, I believe is happened not because of incorrectly ported code but because we are using different fallback extractors compared to the original.

For the speed, our Go port is far faster than the original. This is mainly thanks to re2go that compiles several important regex ahead of time into Go code, so we don't get performance hit from using Go's regex.

By the way, this package is thread-safe (as far as we test it anyway) so depending on your use case you might want to use it concurrently for additional speed. As example, here is the result on my PC when the comparison script run concurrently in all thread:

```
go run scripts/comparison/*.go content -j -1
```

|                 Package                 | Time (s) |
| :-------------------------------------: | :------: |
|            `go-trafilatura`             |  0.931   |
|       `go-trafilatura` + fallback       |  1.976   |
| `go-trafilatura` + fallback + precision |  1.856   |
|  `go-trafilatura` + fallback + recall   |  1.599   |

## Acknowledgements

This package won't be exist without effort by Adrien Barbaresi, the author of the original Python package. He created `trafilatura` as part of effort to [build text databases for research][k-web], to facilitate a better text data collection which lead to a better corpus quality. For more information:

```
@inproceedings{barbaresi-2021-trafilatura,
  title = {{Trafilatura: A Web Scraping Library and Command-Line Tool for Text Discovery and Extraction}},
  author = "Barbaresi, Adrien",
  booktitle = "Proceedings of the Joint Conference of the 59th Annual Meeting of the Association for Computational Linguistics and the 11th International Joint Conference on Natural Language Processing: System Demonstrations",
  pages = "122--131",
  publisher = "Association for Computational Linguistics",
  url = "https://aclanthology.org/2021.acl-demo.15",
  year = 2021,
}
```

- Barbaresi, A. [Trafilatura: A Web Scraping Library and Command-Line Tool for Text Discovery and Extraction][paper-1], Proceedings of ACL/IJCNLP 2021: System Demonstrations, 2021, p. 122-131.
- Barbaresi, A. ["Generic Web Content Extraction with Open-Source Software"][paper-2], Proceedings of KONVENS 2019, Kaleidoscope Abstracts, 2019.
- Barbaresi, A. ["Efficient construction of metadata-enhanced web corpora"][paper-3], Proceedings of the [10th Web as Corpus Workshop (WAC-X)][wac-x], 2016.

## License

Like the original, `go-trafilatura` is distributed under the [Apache v2.0](LICENSE).

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[2]: https://github.com/scrapinghub/article-extraction-benchmark
[3]: https://chromium.googlesource.com/chromium/dom-distiller
[last-version]: https://github.com/adbar/trafilatura/releases/tag/v2.0.0
[last-commit]: https://github.com/adbar/trafilatura/commit/c6e8340
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
[go-regex-slow]: https://github.com/golang/go/issues/26623
[re2go]: https://re2c.org/manual/manual_go.html
[dom-distiller]: https://github.com/markusmobius/go-domdistiller/
[readability]: https://github.com/go-shiori/go-readability
[benchmark]: https://github.com/markusmobius/content-extractor-benchmark
