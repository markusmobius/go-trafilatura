# Go-Trafilatura

Go-Trafilatura is a Go package and command-line tool which seamlessly downloads, parses, and scrapes web page data: it can extract metadata, main body text and comments while preserving parts of the text formatting and page structure.

As implied by its name, this package is based on [Trafilatura][0] which is a Python package that created by [Adrien Barbaresi][1]. We decided to port this package because according to ScrapingHub [benchmark][2], at the time this port is created Trafilatura is the most efficient open-source article extractor. This is especially impressive considering how robust its code, only around 4,000 lines of Python code that separated in 26 files. As comparison, [Dom Distiller][3] has 148 files with around 17,000 lines of code.

The structure of this package is arranged following the structure of original Python code. This way, any improvements from the original can be implemented easily here. Another advantage, hopefully all web page that can be parsed by the original Trafilatura can be parsed by this package as well with identical result.

## Status

This package is still in development and the port process is still not finished. There are 24 files with 4,310 lines of code that haven’t been ported, so there is still long way to go.

## Changelog

### 25 April 2021

- Porting process started

[0]: https://github.com/adbar/trafilatura
[1]: https://github.com/adbar
[2]: https://github.com/scrapinghub/article-extraction-benchmark
[3]: https://chromium.googlesource.com/chromium/dom-distiller