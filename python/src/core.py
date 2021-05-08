# pylint:disable-msg=E0611,I1101
"""
Module bundling all functions needed to extract the text in a webpage.
"""

# This file is available from https://github.com/adbar/trafilatura
# under GNU GPL v3 license


# standard
import logging
import re  # import regex as re

from collections import OrderedDict
from copy import deepcopy

from lxml import etree, html

# own
from .external import justext_rescue, sanitize_tree, SANITIZED_XPATH, try_readability
from .filters import (check_html_lang, content_fingerprint, duplicate_test,
                      language_filter, text_chars_test)
from .htmlprocessing import (convert_tags, discard_unwanted,
                             discard_unwanted_comments, handle_textnode,
                             link_density_test, link_density_test_tables,
                             process_node, tree_cleaning)
from .metadata import extract_metadata, METADATA_LIST
from .settings import use_config, DEFAULT_CONFIG, TAG_CATALOG
from .utils import load_html, trim, txttocsv, is_image_file
from .xml import (build_json_output, build_xml_output, build_tei_output,
                  control_xml_output, xmltotxt)
from .xpaths import BODY_XPATH, COMMENTS_XPATH  # , REMOVE_COMMENTS_XPATH


LOGGER = logging.getLogger(__name__)


def determine_returnstring(docmeta, output_format, include_formatting, include_links, tei_validation):
    '''Convert XML tree to chosen format, clean the result and output it as a string'''
    # XML (TEI) steps
    if 'xml' in output_format:
        # last cleaning
        for element in docmeta['body'].iter():
            if element.tag != 'graphic' and len(element) == 0 and not element.text and not element.tail:
                parent = element.getparent()
                if parent is not None:
                    parent.remove(element)
        # build output trees
        if output_format == 'xml':
            output = build_xml_output(docmeta)
        elif output_format == 'xmltei':
            output = build_tei_output(docmeta)
        # can be improved
        returnstring = control_xml_output(
            output, output_format, tei_validation, docmeta)
    # CSV, JSON and TXT output
    else:
        if output_format == 'csv':
            posttext = xmltotxt(
                docmeta['body'], include_formatting, include_links)
            if docmeta['commentsbody'] is not None:
                commentstext = xmltotxt(
                    docmeta['commentsbody'], include_formatting, include_links)
            else:
                commentstext = ''
            returnstring = txttocsv(posttext, commentstext, docmeta)
        elif output_format == 'json':
            returnstring = build_json_output(docmeta)
        else:  # txt
            returnstring = xmltotxt(
                docmeta['body'], include_formatting, include_links)
            if docmeta['commentsbody'] is not None:
                returnstring += '\n' + \
                    xmltotxt(docmeta['commentsbody'],
                             include_formatting, include_links)
                returnstring = returnstring.strip()
    return returnstring


def extract(filecontent, url=None, record_id=None, no_fallback=False,
            include_comments=True, output_format='txt',
            tei_validation=False, target_language=None,
            include_tables=True, include_images=False, include_formatting=False,
            include_links=False, deduplicate=False,
            date_extraction_params=None, with_metadata=False, max_tree_size=None, url_blacklist=None,
            settingsfile=None, config=DEFAULT_CONFIG):
    """Main function exposed by the package:
       Wrapper for text extraction and conversion to chosen output format.

    Args:
        filecontent: HTML code as string.
        url: URL of the webpage.
        record_id: Add an ID to the metadata.
        no_fallback: Skip the backup extraction with readability-lxml and justext.
        include_comments: Extract comments along with the main text.
        output_format: Define an output format:
            'txt', 'csv', 'json', 'xml', or 'xmltei'.
        tei_validation: Validate the XML-TEI output with respect to the TEI standard.
        target_language: Define a language to discard invalid documents (ISO 639-1 format).
        include_tables: Take into account information within the HTML <table> element.
        include_images: Take images into account (experimental).
        include_formatting: Keep structural elements related to formatting
            (only valuable if output_format is set to XML).
        include_links: Keep links along with their targets (experimental).
        deduplicate: Remove duplicate segments and documents.
        date_extraction_params: Provide extraction parameters to htmldate as dict().
        with_metadata: Only keep documents featuring all essential metadata
            (date, title, url).
        max_tree_size: Discard documents with too many elements.
        url_blacklist: Provide a blacklist of URLs as set() to filter out documents.
        settingsfile: Use a configuration file to override the standard settings.
        config: Directly provide a configparser configuration.

    Returns:
        A string in the desired format or None.

    """
    # configuration init
    config = use_config(settingsfile, config)
    if url_blacklist is None:
        url_blacklist = set()
    # extraction
    docmeta = bare_extraction(
        filecontent, url=url, no_fallback=no_fallback,
        include_comments=include_comments, output_format=output_format,
        target_language=target_language, include_tables=include_tables, include_images=include_images,
        include_formatting=include_formatting, include_links=include_links,
        deduplicate=deduplicate,
        date_extraction_params=date_extraction_params, with_metadata=with_metadata,
        max_tree_size=max_tree_size, url_blacklist=url_blacklist, config=config,
    )
    if docmeta is None:
        return None
    if output_format != 'txt':
        # add record ID to metadata
        docmeta['id'] = record_id
        # calculate fingerprint
        docmeta['fingerprint'] = content_fingerprint(docmeta['raw-text'])
    # return
    return determine_returnstring(docmeta, output_format, include_formatting, include_links, tei_validation)


# for legacy and backwards compatibility
process_record = extract
