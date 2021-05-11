# pylint:disable-msg=W1401
"""
Unit tests for the trafilatura library.
"""

import logging
import os
import sys

from unittest.mock import patch

# import pytest
# https://docs.pytest.org/en/latest/


from lxml import etree, html

try:
    import cchardet as chardet
except ImportError:
    import chardet

# language detection
try:
    import cld3
    LANGID_FLAG = True
except ImportError:
    LANGID_FLAG = False

import trafilatura.filters
import trafilatura.htmlprocessing
from trafilatura.core import baseline, bare_extraction, extract, handle_formatting, handle_lists, handle_image, handle_paragraphs, handle_quotes, handle_table, handle_textelem, process_record, sanitize_tree, trim
from trafilatura.lru import LRUCache
from trafilatura.filters import check_html_lang, duplicate_test, textfilter
from trafilatura.metadata import METADATA_LIST
from trafilatura.settings import DEFAULT_CONFIG

from trafilatura import utils, xml

logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)


TEST_DIR = os.path.abspath(os.path.dirname(__file__))
SAMPLE_META = dict.fromkeys(METADATA_LIST)

ZERO_CONFIG = DEFAULT_CONFIG
ZERO_CONFIG['DEFAULT']['MIN_OUTPUT_SIZE'] = '0'
ZERO_CONFIG['DEFAULT']['MIN_EXTRACTED_SIZE'] = '0'

MOCK_PAGES = {
    'http://exotic_tags': 'exotic_tags.html',
}


def test_input():
    '''test if loaded strings/trees are handled properly'''
    assert utils.load_html(123) is None
    assert utils.load_html('<html><body>ÄÖÜ</body></html>') is not None
    assert utils.load_html(
        b'<html><body>\x2f\x2e\x9f</body></html>') is not None
    #assert utils.load_html(b'0'*int(10e3)) is None
    assert extract(None, 'url', '0000', target_language=None) is None
    # legacy
    assert process_record(None, 'url', '0000', target_language=None) is None


def test_txttocsv():
    mymeta = dict.fromkeys(METADATA_LIST)
    assert utils.txttocsv(
        '', '', mymeta) == 'None\tNone\tNone\tNone\tNone\t\t\n'
    mymeta['title'] = 'Test title'
    mymeta['url'] = 'https://example.org'
    mymeta['hostname'] = 'example.org'
    mymeta['id'] = '1'
    assert utils.txttocsv('Test text', 'Test comment',
                          mymeta) == '1\thttps://example.org\tNone\texample.org\tTest title\tNone\tTest text\tTest comment\n'
    mystring = '<html><body><p>ÄÄÄÄÄÄÄÄÄÄÄÄÄÄ</p></body></html>'
    assert extract(mystring, output_format='csv',
                   config=ZERO_CONFIG) is not None
    assert extract(mystring, output_format='csv',
                   include_comments=False, config=ZERO_CONFIG).endswith('\t\n')
    # test json
    assert extract(mystring, output_format='json',
                   config=ZERO_CONFIG).endswith('}')
    # bare extraction for python
    result = bare_extraction(mystring, config=ZERO_CONFIG)
    assert isinstance(result, dict) and len(result) == 13


def test_exotic_tags(xmloutput=False):
    # cover some edge cases with a specially crafted file
    filepath = os.path.join(TEST_DIR, 'cache', 'exotic_tags_tei.html')
    with open(filepath) as f:
        content = etree.fromstring(f.read())
    res = xml.check_tei(content, 'http://dummy')
    assert etree.tostring(res).startswith(
        b'<html>\n<text>\n<body>\n<div>\n\n<hi rend="uppercase">Hello</hi>\n<p>Teletype text</p>')


def test_tei():
    '''test TEI-related functions'''
    # open local resources to avoid redownloading at each run
    resources_dir = os.path.join(TEST_DIR, 'resources')
    with open(os.path.join(resources_dir, 'httpbin_sample.html')) as f:
        teststring = f.read()
    # download, parse and validate simple html file
    result = extract(teststring, "mocked", no_fallback=True,
                     output_format='xmltei', tei_validation=False)
    assert result is not None
    assert xml.validate_tei(etree.fromstring(result)) is True
    assert xml.validate_tei(etree.fromstring(teststring)) is False
    # test with another file
    with open(os.path.join(resources_dir, 'http_sample.html')) as f:
        teststring = f.read()
    # download, parse and validate simple html file
    result = extract(teststring, "mocked", no_fallback=True,
                     output_format='xmltei', tei_validation=False)
    assert result is not None
    assert xml.validate_tei(etree.fromstring(result)) is True
    # include ID in metadata
    result = extract(teststring, "mocked", no_fallback=True,
                     output_format='xmltei', tei_validation=False, record_id='0001')
    assert result is not None
    assert xml.validate_tei(etree.fromstring(result)) is True


def test_htmlprocessing():
    '''test html-related functions'''
    assert trafilatura.htmlprocessing.tree_cleaning(
        etree.Element('html'), True) is not None
    assert trafilatura.htmlprocessing.prune_html(
        etree.Element('unwanted')) is not None
    mydoc = html.fromstring(
        '<html><body><table><a href="">Link</a></table><img src="test.jpg"/><u>Underlined</u><tt>True Type</tt><sub>Text</sub><sup>Text</sup></body></html>')
    myconverted = trafilatura.htmlprocessing.convert_tags(
        mydoc, include_formatting=True, include_tables=True, include_images=True)
    assert myconverted.xpath('.//ref') and myconverted.xpath(
        './/graphic') and myconverted.xpath('.//hi[@rend="#t"]') and myconverted.xpath('.//table')
    myconverted = trafilatura.htmlprocessing.tree_cleaning(
        mydoc, include_tables=False, include_images=True)
    assert myconverted.xpath(
        './/graphic') and not myconverted.xpath('.//table')
    mydoc = html.fromstring(
        '<html><body><article><h1>Test headline</h1><p>Test</p></article></body></html>')
    assert '<head rend="h1">Test headline</head>' in extract(
        mydoc, output_format='xml', config=ZERO_CONFIG, no_fallback=True)
    assert '<fw rend="h1" type="header">Test headline</fw>' in extract(
        mydoc, output_format='xmltei', config=ZERO_CONFIG, no_fallback=True)


def test_fetch():
    '''test URL fetching'''
    assert utils.fetch_url('1234') == ''
    assert utils.fetch_url('https://httpbin.org/status/404') is None
    assert utils.decode_response(b'\x1f\x8babcdef') is not None
    assert utils.fetch_url('https://expired.badssl.com/',
                           no_ssl=True) is not None


if __name__ == '__main__':
    test_input()
    test_exotic_tags()
    test_htmlprocessing()
    test_txttocsv()
    test_fetch()
    test_tei()
