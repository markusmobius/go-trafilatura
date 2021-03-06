# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web
page data: it can extract metadata, main body text and comments while preserving parts of the text
formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created
by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2],
at the time this port is created Trafilatura is the most efficient open-source article extractor. This is
especially impressive considering how robust its code, only around 4,000 lines of Python code that
separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any
improvements from the original can be implemented easily here. Another advantage, hopefully all web page
that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Table of Contents

- [Status](#status)
- [Usage as Go package](#usage-as-go-package)
- [Usage as CLI Application](#usage-as-cli-application)
- [Comparison with Other Go Package](#comparison-with-other-go-packages)
- [Comparison with Original Trafilatura](#comparison-with-original-trafilatura)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Status

This package is stable enough for use and up to date with the original Trafilatura v0.8.2 (commit
[8f512db][4]). I've also picked some latest commits that fix important issues.

There are some difference between this port and the original Trafilatura:

- In the original, metadata from JSON+LD is extracted using regular expressions while in this port it's
  done using a JSON parser. This way, our metadata extraction should be more correct, however the code
  is a bit more complex.
- In the original, they use `python-readability` and `justext` as fallback extractors. In this port we
  use `go-readability` and `go-domdistiller` instead. Thanks to this, while rare, there will be some
  difference in extraction result between our port and the original.
- The main output of the original Trafilatura is XML, while in our port the main output is HTML. Thanks
  to this, there are some difference in handling formatting tags (e.g. `<b>`, `<i>`) and paragraphs.

## Usage as Go package

Run following command inside your Go project :

```
go get -u -v github.com/markusmobius/go-trafilatura
```

Next, include it in your application :

```go
import "github.com/markusmobius/go-trafilatura"
```

Now you can use Trafilatura to extract content of a web page. For basic usage you can check the
[example](examples/from-url.go).

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

## Comparison with Other Go Packages

Here we compare the extraction result between `go-trafilatura`, `go-readability` and `go-domdistiller`.
To reproduce this test, clone this repository then run:

```
go run scripts/comparison/*.go
```

For the test, we use 500 documents, 1,487 text and 1,496 boilerplate segments (data from 2020-11-06).
Here is the result when tested in my PC (Intel i7-8550U @ 4.000GHz, RAM 16 GB):

|            Package             | Precision | Recall | Accuracy | F-Score | Speed (s) |
| :----------------------------: | :-------: | :----: | :------: | :-----: | :-------: |
|       `go-domdistiller`        |   0.900   | 0.831  |  0.870   |  0.864  |   6.795   |
|        `go-readability`        |   0.895   | 0.866  |  0.882   |  0.880  |   6.604   |
|        `go-trafilatura`        |   0.921   | 0.867  |  0.896   |  0.893  |   3.656   |
| `go-trafilatura` with fallback |   0.926   | 0.888  |  0.909   |  0.907  |  10.663   |

As you can see, in our benchmark `go-trafilatura` leads the way. However, it does have a weakness. For
instance, the image extraction in `go-trafilatura` is not as good as the other.

## Comparison with Original Trafilatura

For our purpose, we need a high quality extractor that is as fast as possible, especially since we process
millions of page each day. Because of this, we decided to port Trafilatura from Python to Go since we
believe the accuracy of Trafilatura combined with Go's speed will fulfill our requirement.

With that said, we want to make sure that our port has similar performance as the original. Fortunately
Trafilatura already has a script named `comparison.py` which used to compare it with several other Python
libraries, so we can easily use it to compare the performance between the original and our port:

|             Package             | Precision | Recall | Accuracy | F-Score | Speed (s) |
| :-----------------------------: | :-------: | :----: | :------: | :-----: | :-------: |
|        `go-trafilatura`         |   0.921   | 0.867  |  0.896   |  0.893  |   3.656   |
|   `go-trafilatura` + fallback   |   0.925   | 0.888  |  0.908   |  0.906  |   3.750   |
|      `trafilatura` v0.8.2       |   0.925   | 0.867  |  0.899   |  0.895  |  10.123   |
| `trafilatura` v0.8.2 + fallback |   0.934   | 0.889  |  0.914   |  0.911  |  20.757   |

According to the table above, our port has the same performance as the original Trafilatura. It makes
sense since most of code is ported line by line from Python to Go (excluding some difference that mentioned
above).

There is a small difference in precision between our port and the original. However, I believe this is
happened not because of incorrectly ported code but because we are using different fallback extractors
compared to the original.

Regarding speed, our port is 2.77 times faster when fallback is disabled and 1.95 times faster when fallback
is enabled. So, in this aspect Go fulfill our expectation as well. However, if you check the comparison code,
you'll notice that each comparison process done sequentially. With that said, if you want to achieve higher
performance, you could try to use `go-trafilatura` package inside goroutines to make it run concurrently.

However, I decided to not compare the speed between concurrently running our port and concurrently running
the original Trafilatura. This is because Python doesn't support CPU threading, so even if we run the original
Trafilatura in multiple thread, it will become a lot more slower than running it sequentially in single
thread.

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

Like the original, `go-trafilatura` is distributed under the [GNU General Public License v3.0](LICENSE).

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[2]: https://github.com/scrapinghub/article-extraction-benchmark
[3]: https://chromium.googlesource.com/chromium/dom-distiller
[4]: https://github.com/adbar/trafilatura/commit/25698ebc93e1625f81f2d1f2300caf27425df33e
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
