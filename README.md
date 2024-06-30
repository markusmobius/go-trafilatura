# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web page data: it can extract metadata, main body text and comments while preserving parts of the text formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2], at the time this port is created Trafilatura is the most efficient open-source article extractor. This is especially impressive considering how robust its code, only around 4,000 lines of Python code that separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any improvements from the original can be implemented easily here. Another advantage, hopefully all web page that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Table of Contents

- [Status](#status)
- [Usage as Go package](#usage-as-go-package)
- [Usage as CLI Application](#usage-as-cli-application)
- [Performance](#performance)
  - [Compiling with cgo under Linux](#compiling-with-cgo-under-linux)
  - [Compiling with cgo under Windows](#compiling-with-cgo-under-windows)
- [Comparison with Other Go Packages](#comparison-with-other-go-packages)
- [Comparison with Original Trafilatura](#comparison-with-original-trafilatura)
- [Acknowledgements](#acknowledgements)
- [License](#license)

## Status

This package is stable enough for use and up to date with the original Trafilatura [v1.10.0][last-version] (commit [b36b6fa][last-commit]).

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

This package and its dependencies heavily use regular expression for various purposes. Unfortunately, as commonly known, Go's regular expression is pretty slow, even compared to Python. This is because:

- The regex engine in other language usually implemented in C, while in Go it's implemented from scratch in Go language. As expected, C implementation is still faster than Go's.
- Since Go is usually used for web service, its regex is designed to finish in time linear to the length of the input, which useful for protecting server from ReDoS attack. However, this comes with performance cost.

If you want to parse a huge amount of data, it would be preferrable to have a better performance. So, this package provides C++ [`re2`][re2] as an alternative regex engine using binding from [go-re2]. To activate it, make sure you have `re2` libraries installed on your system:

- Arch: `sudo pacman -S re2`
- Ubuntu: `sudo apt install libre2-dev`

After `re2` available in your system, you can build your app using tag `re2_wasm` or `re2_cgo`, for example:

```
go build -tags re2_cgo .
```

More detailed instructions in how to prepare your system for compiling with cgo are provided below.

When using `re2_wasm` tag, it will make your app uses `re2` that packaged as WebAssembly module so it should be runnable even without cgo. However, if your input is too small, it might be even slower than using Go's standard regex engine.

When using `re2_cgo` tag, it will make your app uses `re2` library that wrapped using cgo. In most case it's a lot faster than Go's standard regex and `re2_wasm`, however to use it cgo must be available and `re2` should be installed in your system.

Do note that this alternative regex engine is experimental, so use on your own risk.

### Compiling with cgo under Linux

On Ubuntu install the gcc tool chain and the re2 library as follows:

```bash
sudo apt install build-essential
sudo apt-get install -y libre2-dev
```

### Compiling with cgo under Windows

On Windows start by installing [MSYS2][msys2]. Then open the MINGW64 terminal and install the gcc toolchain and re2 via pacman:

```bash
pacman -S mingw-w64-x86_64-gcc
pacman -S mingw-w64-x86_64-re2
```

If you want to run the resulting exe program outside the MINGW64 terminal you need to add a path to the MinGW-w64 libraries to the PATH environmental variable (adjust as needed for your system):

```cmd
SET PATH=C:\msys64\mingw64\bin;%PATH%
```

## Comparison with Other Go Packages

Here we compare the extraction result between `go-trafilatura`, `go-readability` and `go-domdistiller`. To reproduce this test, clone this repository then run:

```
go run scripts/comparison/*.go content
```

For the test, we use 976 documents taken from various sources (2024-06-30). Here is the result when tested in my PC (Intel i7-8550U @ 4.000GHz, RAM 16 GB):

|            Package             | Precision | Recall | Accuracy | F-Score | Speed (s) |
| :----------------------------: | :-------: | :----: | :------: | :-----: | :-------: |
|        `go-readability`        |   0.870   | 0.882  |  0.875   |  0.876  |  11.731   |
|       `go-domdistiller`        |   0.871   | 0.865  |  0.869   |  0.868  |  11.829   |
|        `go-trafilatura`        |   0.908   | 0.885  |  0.898   |  0.896  |  16.167   |
| `go-trafilatura` with fallback |   0.912   | 0.902  |  0.907   |  0.906  |  36.524   |

As you can see, in our benchmark `go-trafilatura` leads the way. However, it does have a weakness. For instance, the image extraction in `go-trafilatura` is still not as good as the other.

## Comparison with Original Trafilatura

Here is the result when compared with the original Trafilatura v1.10.0:

|                 Package                 | Precision | Recall | Accuracy | F-Score |
| :-------------------------------------: | :-------: | :----: | :------: | :-----: |
|              `trafilatura`              |   0.912   | 0.886  |  0.900   |  0.899  |
|        `trafilatura` + fallback         |   0.913   | 0.903  |  0.904   |  0.902  |
|  `trafilatura` + fallback + precision   |   0.925   | 0.879  |  0.904   |  0.902  |
|    `trafilatura` + fallback + recall    |   0.900   | 0.907  |  0.904   |  0.904  |
|            `go-trafilatura`             |   0.908   | 0.885  |  0.898   |  0.896  |
|       `go-trafilatura` + fallback       |   0.912   | 0.902  |  0.907   |  0.906  |
| `go-trafilatura` + fallback + precision |   0.924   | 0.876  |  0.902   |  0.899  |
|  `go-trafilatura` + fallback + recall   |   0.899   | 0.904  |  0.901   |  0.901  |

From the table above we can see that our port has almost similar performance as the original Trafilatura. This is thanks to the fact that most of code is ported line by line from Python to Go (excluding some difference that mentioned above). The small performance difference between our port and the original, I believe is happened not because of incorrectly ported code but because we are using different fallback extractors compared to the original.

For the speed, here is the comparison between our port and the original Trafilatura (all units in seconds):

|             Name              | Standard | Fallback | Fallback + Precision | Fallback + Recall |
| :---------------------------: | :------: | :------: | :------------------: | :---------------: |
|         `trafilatura`         |  19.59   |  29.01   |        39.68         |       22.71       |
|       `go-trafilatura`        |  16.17   |  36.52   |        36.73         |       27.50       |
| `go-trafilatura` + `re2_wasm` |  11.29   |  21.35   |        21.28         |       14.39       |
| `go-trafilatura` + `re2_cgo`  |   9.95   |  21.20   |        20.26         |       13.02       |

As you can see, our Go port is faster when running in standard mode (without fallback), but become slower when fallback extractors is enabled. This is mainly because of date extractor from `go-htmldate` running in extensive mode when fallback enabled, which lead to heavy use of regex, which lead to slow speed. Fortunately, when `re2` is enabled our port become a lot faster in every scenarios.

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
[last-version]: https://github.com/adbar/trafilatura/releases/tag/v1.10.0
[last-commit]: https://github.com/adbar/trafilatura/commit/b36b6fa
[paper-1]: https://aclanthology.org/2021.acl-demo.15/
[paper-2]: https://hal.archives-ouvertes.fr/hal-02447264/document
[paper-3]: https://hal.archives-ouvertes.fr/hal-01371704v2/document
[wac-x]: https://www.sigwac.org.uk/wiki/WAC-X
[k-web]: https://www.dwds.de/d/k-web
[re2]: https://github.com/google/re2
[go-re2]: https://github.com/wasilibs/go-re2
[msys2]: https://www.msys2.org/
