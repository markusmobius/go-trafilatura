# pylint:disable-msg=I1101
"""
Functions to process nodes in HTML code.
"""

# This file is available from https://github.com/adbar/trafilatura
# under GNU GPL v3 license

import logging
import re

from lxml import etree
from lxml.html.clean import Cleaner

from .filters import duplicate_test, textfilter
from .settings import CUT_EMPTY_ELEMS, DEFAULT_CONFIG, MANUALLY_CLEANED, MANUALLY_STRIPPED
from .utils import trim
from .xpaths import COMMENTS_DISCARD_XPATH, DISCARD_XPATH

LOGGER = logging.getLogger(__name__)


def convert_tags(tree, include_formatting=False, include_tables=False, include_images=False, include_links=False):
    '''Simplify markup and convert relevant HTML tags to an XML standard'''
    # ul/ol → list / li → item
    for elem in tree.iter('ul', 'ol', 'dl'):
        elem.tag = 'list'
        for subelem in elem.iter('dd', 'dt', 'li'):
            subelem.tag = 'item'
        for subelem in elem.iter('a'):
            subelem.tag = 'ref'
    # divs
    for elem in tree.xpath('//div//a'):
        elem.tag = 'ref'
    # tables
    if include_tables is True:
        for elem in tree.xpath('//table//a'):
            elem.tag = 'ref'
    # images
    if include_images is True:
        for elem in tree.iter('img'):
            elem.tag = 'graphic'
    # delete links for faster processing
    if include_links is False:
        etree.strip_tags(tree, 'a')
    else:
        for elem in tree.iter('a', 'ref'):
            elem.tag = 'ref'
            # replace href attribute and delete the rest
            for attribute in elem.attrib:
                if attribute == 'href':
                    elem.set('target', elem.get('href'))
                else:
                    del elem.attrib[attribute]
            # if elem.attrib['href']:
            #    del elem.attrib['href']
    # head tags + delete attributes
    for elem in tree.iter('h1', 'h2', 'h3', 'h4', 'h5', 'h6'):
        elem.attrib.clear()
        elem.set('rend', elem.tag)
        elem.tag = 'head'
    # br → lb
    for elem in tree.iter('br', 'hr'):
        elem.tag = 'lb'
    # wbr
    # blockquote, pre, q → quote
    for elem in tree.iter('blockquote', 'pre', 'q'):
        elem.tag = 'quote'
    # include_formatting
    if include_formatting is False:
        etree.strip_tags(tree, 'em', 'i', 'b', 'strong', 'u',
                         'kbd', 'samp', 'tt', 'var', 'sub', 'sup')
    else:
        # italics
        for elem in tree.iter('em', 'i'):
            elem.tag = 'hi'
            elem.set('rend', '#i')
        # bold font
        for elem in tree.iter('b', 'strong'):
            elem.tag = 'hi'
            elem.set('rend', '#b')
        # u (very rare)
        for elem in tree.iter('u'):
            elem.tag = 'hi'
            elem.set('rend', '#u')
        # tt (very rare)
        for elem in tree.iter('kbd', 'samp', 'tt', 'var'):
            elem.tag = 'hi'
            elem.set('rend', '#t')
        # sub and sup (very rare)
        for elem in tree.iter('sub'):
            elem.tag = 'hi'
            elem.set('rend', '#sub')
        for elem in tree.iter('sup'):
            elem.tag = 'hi'
            elem.set('rend', '#sup')
    # del | s | strike → <del rend="overstrike">
    for elem in tree.iter('del', 's', 'strike'):
        elem.tag = 'del'
        elem.set('rend', 'overstrike')
    return tree


def process_node(element, deduplicate=True, config=DEFAULT_CONFIG):
    '''Convert, format, and probe potential text elements (light format)'''
    if element.tag == 'done':
        return None
    if len(element) == 0 and not element.text and not element.tail:
        return None
    # trim
    element.text, element.tail = trim(element.text), trim(element.tail)
    # adapt content string
    if element.tag != 'lb' and not element.text and element.tail:
        element.text = element.tail
    # content checks
    if element.text or element.tail:
        if textfilter(element) is True:
            return None
        if deduplicate is True and duplicate_test(element, config) is True:
            return None
    return element
